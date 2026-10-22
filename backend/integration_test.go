package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var schemaOnce sync.Once

func testApp(t *testing.T) *app {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL integration tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schemaOnce.Do(func() {
		if err := db.AutoMigrate(&User{}, &Session{}, &Course{}, &Enrollment{}, &Section{}, &Lesson{}, &Completion{}, &Assignment{}, &Submission{}, &Quiz{}, &Question{}, &Attempt{}, &Announcement{}, &Discussion{}, &Cohort{}, &CohortMember{}, &CourseCohort{}, &BankItem{}); err != nil {
			t.Fatal(err)
		}
	})
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback(); sql, _ := db.DB(); sql.Close() })
	if err := seed(tx); err != nil {
		t.Fatal(err)
	}
	return &app{db: tx, logins: map[string]loginLimit{}}
}
func request(t *testing.T, a *app, user uint, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&b).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest(method, path, &b)
	if user > 0 {
		token := fmt.Sprintf("test-%d", user)
		a.db.Where("token = ?", hash(token)).FirstOrCreate(&Session{Token: hash(token), UserID: user, ExpiresAt: time.Now().Add(time.Hour)})
		r.AddCookie(&http.Cookie{Name: "campus_session", Value: token})
	}
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, r)
	return w
}
func status(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("HTTP %d, want %d: %s", w.Code, want, w.Body.String())
	}
}
func unmarshal[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func userID(t *testing.T, a *app, role string) uint {
	t.Helper()
	var u User
	if err := a.db.Where("role = ?", role).First(&u).Error; err != nil {
		t.Fatal(err)
	}
	return u.ID
}
func TestAuthAndPermissions(t *testing.T) {
	a := testApp(t)
	student := userID(t, a, "student")
	teacher := userID(t, a, "teacher")
	adminID := userID(t, a, "admin")
	status(t, request(t, a, 0, "GET", "/api/courses", nil), 401)
	status(t, request(t, a, 0, "POST", "/api/auth/register", map[string]any{"name": "Abcd", "email": "new@test.local", "password": "abcdefgh", "role": "admin"}), 400)
	w := request(t, a, 0, "POST", "/api/auth/login", map[string]string{"email": "student@campus.local", "password": "Learn2026!"})
	status(t, w, 200)
	if strings.Contains(w.Body.String(), "Password") || strings.Contains(w.Body.String(), "Learn2026") {
		t.Fatal("credentials leaked")
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("session cookie protection missing")
	}
	status(t, request(t, a, student, "POST", "/api/courses", map[string]any{"title": "Forbidden"}), 403)
	status(t, request(t, a, student, "GET", "/api/users", nil), 403)
	status(t, request(t, a, teacher, "PATCH", fmt.Sprintf("/api/users/%d/role", student), map[string]string{"role": "admin"}), 403)
	status(t, request(t, a, adminID, "PATCH", fmt.Sprintf("/api/users/%d/role", adminID), map[string]string{"role": "student"}), 400)
	r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"x","password":"x"}`))
	r.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	a.routes().ServeHTTP(rec, r)
	status(t, rec, 403)
	// A role downgrade removes authoring access even if the user still owns a course.
	a.db.Model(&User{}).Where("id = ?", teacher).Update("role", "student")
	var c Course
	a.db.First(&c)
	status(t, request(t, a, teacher, "POST", fmt.Sprintf("/api/courses/%d/sections", c.ID), map[string]string{"title": "Forbidden"}), 403)
}
func TestLearningAndGradingWorkflow(t *testing.T) {
	a := testApp(t)
	student := userID(t, a, "student")
	teacher := userID(t, a, "teacher")
	var c Course
	a.db.Order("id DESC").First(&c)
	path := fmt.Sprintf("/api/courses/%d", c.ID)
	status(t, request(t, a, student, "GET", path, nil), 403)
	status(t, request(t, a, student, "POST", path+"/enroll", nil), 200)
	w := request(t, a, student, "GET", path, nil)
	status(t, w, 200)
	if strings.Contains(w.Body.String(), "correct_index") {
		t.Fatal("correct answers exposed")
	}
	d := unmarshal[struct {
		Sections    []Section    `json:"sections"`
		Assignments []Assignment `json:"assignments"`
		Quizzes     []Quiz       `json:"quizzes"`
	}](t, w)
	lesson := d.Sections[0].Lessons[0]
	for i := 0; i < 2; i++ {
		status(t, request(t, a, student, "POST", fmt.Sprintf("/api/lessons/%d/complete", lesson.ID), nil), 200)
	}
	var n int64
	a.db.Model(&Completion{}).Where("lesson_id = ? AND user_id = ?", lesson.ID, student).Count(&n)
	if n != 1 {
		t.Fatal("duplicate completions")
	}
	as := d.Assignments[0]
	w = request(t, a, student, "POST", fmt.Sprintf("/api/assignments/%d/submit", as.ID), map[string]string{"content": "Моя практическая работа"})
	status(t, w, 200)
	sub := unmarshal[Submission](t, w)
	status(t, request(t, a, student, "PATCH", fmt.Sprintf("/api/submissions/%d/grade", sub.ID), map[string]any{"score": 99, "feedback": "Fake"}), 403)
	status(t, request(t, a, teacher, "PATCH", fmt.Sprintf("/api/submissions/%d/grade", sub.ID), map[string]any{"score": as.MaxScore + 1, "feedback": "Bad"}), 400)
	status(t, request(t, a, teacher, "PATCH", fmt.Sprintf("/api/submissions/%d/grade", sub.ID), map[string]any{"score": 88, "feedback": "Хорошая работа"}), 200)
	status(t, request(t, a, student, "POST", fmt.Sprintf("/api/assignments/%d/submit", as.ID), map[string]string{"content": "Не перезаписывать оценённую работу"}), 409)
	q := d.Quizzes[0]
	quizPath := fmt.Sprintf("/api/quizzes/%d/attempts", q.ID)
	status(t, request(t, a, student, "POST", quizPath, map[string]any{"answers": []int{99, 0, 1}}), 400)
	for i := 0; i < 3; i++ {
		w = request(t, a, student, "POST", quizPath, map[string]any{"answers": []int{1, 0, 1}})
		status(t, w, 201)
		attempt := unmarshal[Attempt](t, w)
		if attempt.Score != 3 {
			t.Fatal("server scoring failed")
		}
	}
	status(t, request(t, a, student, "POST", quizPath, map[string]any{"answers": []int{1, 0, 1}}), 409)
	w = request(t, a, student, "GET", path+"/grades", nil)
	status(t, w, 200)
	var grade struct {
		Rows []struct {
			User        User         `json:"user"`
			Submissions []Submission `json:"submissions"`
		} `json:"rows"`
	}
	json.Unmarshal(w.Body.Bytes(), &grade)
	if len(grade.Rows) != 1 || grade.Rows[0].User.ID != student || *grade.Rows[0].Submissions[0].Score != 88 {
		t.Fatal("incorrect gradebook or access scope")
	}
}
func TestCohortEnrollmentAndManualPreservation(t *testing.T) {
	a := testApp(t)
	adminID := userID(t, a, "admin")
	teacher := userID(t, a, "teacher")
	student := userID(t, a, "student")
	var c Course
	a.db.Order("id DESC").First(&c)
	base := fmt.Sprintf("/api/courses/%d", c.ID)
	status(t, request(t, a, student, "GET", "/api/cohorts", nil), 403)
	w := request(t, a, adminID, "POST", "/api/cohorts", map[string]string{"name": "Поток 2026", "description": "Глобальная группа"})
	status(t, w, 201)
	group := unmarshal[Cohort](t, w)
	status(t, request(t, a, teacher, "POST", fmt.Sprintf("/api/cohorts/%d/members", group.ID), map[string]uint{"user_id": student}), 403)
	status(t, request(t, a, teacher, "POST", base+"/cohorts", map[string]uint{"cohort_id": group.ID}), 200)
	memberPath := fmt.Sprintf("/api/cohorts/%d/members", group.ID)
	status(t, request(t, a, adminID, "POST", memberPath, map[string]uint{"user_id": student}), 200)
	status(t, request(t, a, student, "GET", base, nil), 200)
	status(t, request(t, a, adminID, "DELETE", memberPath+fmt.Sprintf("/%d", student), nil), 200)
	status(t, request(t, a, student, "GET", base, nil), 403)
	// Re-adding, self-enrolling, then removing the cohort must retain manual enrolment.
	status(t, request(t, a, adminID, "POST", memberPath, map[string]uint{"user_id": student}), 200)
	status(t, request(t, a, student, "POST", base+"/enroll", nil), 200)
	status(t, request(t, a, adminID, "DELETE", memberPath+fmt.Sprintf("/%d", student), nil), 200)
	status(t, request(t, a, student, "GET", base, nil), 200)
	var e Enrollment
	a.db.Where("user_id = ? AND course_id = ?", student, c.ID).First(&e)
	if e.Source != "manual" {
		t.Fatal("manual source lost")
	}
}
func TestBankReuseAndMoodleImport(t *testing.T) {
	a := testApp(t)
	student := userID(t, a, "student")
	teacher := userID(t, a, "teacher")
	var c Course
	a.db.First(&c)
	base := fmt.Sprintf("/api/courses/%d", c.ID)
	status(t, request(t, a, student, "GET", "/api/bank", nil), 403)
	w := request(t, a, teacher, "POST", "/api/bank", map[string]any{"title": "Общий материал", "kind": "page", "body": "Переиспользуемый текст", "category": "Общее", "duration": 10, "max_score": 100, "questions": []any{}})
	status(t, w, 201)
	item := unmarshal[BankItem](t, w)
	status(t, request(t, a, teacher, "POST", base+fmt.Sprintf("/bank/%d", item.ID), map[string]any{"section_id": 0}), 201)
	var count int64
	a.db.Model(&Lesson{}).Where("course_id = ? AND title = ?", c.ID, "Общий материал").Count(&count)
	if count != 1 {
		t.Fatal("bank material not copied")
	}
	var b bytes.Buffer
	mp := multipart.NewWriter(&b)
	file, _ := mp.CreateFormFile("backup", "moodle.mbz")
	file.Write(tarBackup(t, backupFixture()))
	mp.Close()
	token := "import-session"
	a.db.Create(&Session{Token: hash(token), UserID: teacher, ExpiresAt: time.Now().Add(time.Hour)})
	r := httptest.NewRequest("POST", base+"/import/moodle", &b)
	r.Header.Set("Content-Type", mp.FormDataContentType())
	r.AddCookie(&http.Cookie{Name: "campus_session", Value: token})
	rec := httptest.NewRecorder()
	a.routes().ServeHTTP(rec, r)
	status(t, rec, 201)
	report := unmarshal[struct {
		Activities int      `json:"activities"`
		Bank       int      `json:"question_bank"`
		Warnings   []string `json:"warnings"`
	}](t, rec)
	if report.Activities != 4 || report.Bank != 1 {
		t.Fatalf("wrong import report: %+v", report)
	}
	var imported Quiz
	if err := a.db.Preload("Questions").Where("course_id = ? AND title = ?", c.ID, "Контрольный тест").First(&imported).Error; err != nil {
		t.Fatal(err)
	}
	w = request(t, a, student, "POST", fmt.Sprintf("/api/quizzes/%d/attempts", imported.ID), map[string]any{"answers": []int{1}})
	status(t, w, 201)
	attempt := unmarshal[Attempt](t, w)
	if attempt.Score != 1 {
		t.Fatal("imported quiz cannot be scored")
	}
	// Removing the bank source must not remove its independent course copy.
	status(t, request(t, a, teacher, "DELETE", fmt.Sprintf("/api/bank/%d", item.ID), nil), 200)
	a.db.Model(&Lesson{}).Where("course_id = ? AND title = ?", c.ID, "Общий материал").Count(&count)
	if count != 1 {
		t.Fatal("bank deletion removed course copy")
	}
}
