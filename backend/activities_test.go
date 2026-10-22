package main

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func firstCourse(t *testing.T, a *app) (Course, Section) {
	t.Helper()
	var c Course
	var s Section
	if err := a.db.First(&c).Error; err != nil {
		t.Fatal(err)
	}
	if err := a.db.Where("course_id = ?", c.ID).First(&s).Error; err != nil {
		t.Fatal(err)
	}
	return c, s
}
func newActivity(t *testing.T, a *app, teacher, sid uint, kind string, data ActivityData) Lesson {
	t.Helper()
	w := request(t, a, teacher, "POST", fmt.Sprintf("/api/sections/%d/lessons", sid), map[string]any{"title": "Элемент " + kind, "kind": kind, "duration": 10, "content": "Учебное содержание", "data": data, "url": "https://example.org/video.mp4"})
	status(t, w, 201)
	return unmarshal[Lesson](t, w)
}
func TestCourseSettingsAndScheduledAccess(t *testing.T) {
	a := testApp(t)
	teacher := userID(t, a, "teacher")
	student := userID(t, a, "student")
	c, _ := firstCourse(t, a)
	path := fmt.Sprintf("/api/courses/%d", c.ID)
	settings := map[string]any{"title": c.Title, "description": c.Description, "category": c.Category, "color": c.Color, "published": true, "short_name": "UX-101", "format": "weeks", "language": "ru", "disable_self_enrollment": true, "start_at": nil, "end_at": nil}
	w := request(t, a, teacher, "PATCH", path, settings)
	status(t, w, 200)
	saved := unmarshal[Course](t, w)
	if saved.ShortName != "UX-101" || saved.Format != "weeks" || !saved.DisableSelfEnrollment {
		t.Fatal("settings not saved")
	}
	status(t, request(t, a, student, "POST", path+"/enroll", nil), 403)
	status(t, request(t, a, student, "GET", path, nil), 200) // Existing cohort/manual enrolments are retained.
	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	settings["start_at"] = start
	settings["end_at"] = end
	status(t, request(t, a, teacher, "PATCH", path, settings), 200)
	status(t, request(t, a, student, "GET", path, nil), 403)
	status(t, request(t, a, teacher, "GET", path, nil), 200)
	settings["end_at"] = start.Add(-time.Hour)
	status(t, request(t, a, teacher, "PATCH", path, settings), 400)
	settings["start_at"] = nil
	settings["end_at"] = nil
	settings["disable_self_enrollment"] = false
	w = request(t, a, teacher, "PATCH", path, settings)
	status(t, w, 200)
	if unmarshal[Course](t, w).DisableSelfEnrollment {
		t.Fatal("cannot reenable self enrolment")
	}
}
func TestTopicOrderingVisibilityAndMoving(t *testing.T) {
	a := testApp(t)
	teacher := userID(t, a, "teacher")
	student := userID(t, a, "student")
	c, s := firstCourse(t, a)
	w := request(t, a, teacher, "POST", fmt.Sprintf("/api/courses/%d/sections", c.ID), map[string]any{"title": "Скрытая тема", "summary": "Описание темы", "hidden": true})
	status(t, w, 201)
	hidden := unmarshal[Section](t, w)
	l := newActivity(t, a, teacher, hidden.ID, "page", ActivityData{})
	status(t, request(t, a, student, "POST", fmt.Sprintf("/api/lessons/%d/complete", l.ID), nil), 404)
	w = request(t, a, student, "GET", fmt.Sprintf("/api/courses/%d", c.ID), nil)
	status(t, w, 200)
	if strings.Contains(w.Body.String(), "Скрытая тема") || strings.Contains(w.Body.String(), "Описание темы") {
		t.Fatal("hidden topic leaked")
	}
	status(t, request(t, a, teacher, "DELETE", fmt.Sprintf("/api/sections/%d", hidden.ID), nil), 409)
	sections := []Section{}
	a.db.Where("course_id = ?", c.ID).Order("position").Find(&sections)
	ids := []uint{}
	for i := len(sections) - 1; i >= 0; i-- {
		ids = append(ids, sections[i].ID)
	}
	order := fmt.Sprintf("/api/courses/%d/sections/order", c.ID)
	status(t, request(t, a, student, "PUT", order, map[string]any{"ids": ids}), 403)
	status(t, request(t, a, teacher, "PUT", order, map[string]any{"ids": []uint{s.ID, s.ID}}), 400)
	status(t, request(t, a, teacher, "PUT", order, map[string]any{"ids": ids}), 200)
	var first Section
	a.db.Where("course_id = ?", c.ID).Order("position").First(&first)
	if first.ID != hidden.ID {
		t.Fatal("topics not reordered")
	}
	status(t, request(t, a, teacher, "PATCH", fmt.Sprintf("/api/lessons/%d", l.ID), map[string]any{"title": l.Title, "kind": "page", "content": l.Content, "duration": l.Duration, "section_id": s.ID, "data": ActivityData{}}), 200)
	status(t, request(t, a, student, "POST", fmt.Sprintf("/api/lessons/%d/complete", l.ID), nil), 200)
	status(t, request(t, a, teacher, "DELETE", fmt.Sprintf("/api/sections/%d", hidden.ID), nil), 200)
	// Other course instructors do not gain authoring rights.
	other := User{Name: "Другой преподаватель", Email: "other-teacher@test.local", Role: "teacher"}
	a.db.Create(&other)
	status(t, request(t, a, other.ID, "PATCH", fmt.Sprintf("/api/sections/%d", s.ID), map[string]any{"title": "Изменение", "hidden": false}), 403)
}
func TestInteractiveResourcesAndStructuredBankCopy(t *testing.T) {
	a := testApp(t)
	teacher := userID(t, a, "teacher")
	student := userID(t, a, "student")
	c, s := firstCourse(t, a)
	book := newActivity(t, a, teacher, s.ID, "book", ActivityData{Chapters: []Chapter{{Title: "Первая глава", Content: "Полный текст"}, {Title: "Вторая глава", Content: "Продолжение"}}})
	w := request(t, a, teacher, "POST", fmt.Sprintf("/api/lessons/%d/bank", book.ID), nil)
	status(t, w, 201)
	bank := unmarshal[BankItem](t, w)
	if len(bank.Data.Chapters) != 2 {
		t.Fatal("book structure lost in bank")
	}
	status(t, request(t, a, teacher, "POST", fmt.Sprintf("/api/courses/%d/bank/%d", c.ID, bank.ID), map[string]any{"section_id": s.ID}), 201)
	var copied Lesson
	a.db.Where("title = ? AND id <> ?", book.Title, book.ID).First(&copied)
	if len(copied.Data.Chapters) != 2 {
		t.Fatal("book structure lost in course copy")
	}
	newActivity(t, a, teacher, s.ID, "glossary", ActivityData{Entries: []GlossaryEntry{{Term: "API", Definition: "Программный интерфейс"}}})
	poll := newActivity(t, a, teacher, s.ID, "choice", ActivityData{Options: []string{"Утром", "Вечером"}})
	base := fmt.Sprintf("/api/lessons/%d", poll.ID)
	status(t, request(t, a, student, "POST", base+"/vote", map[string]any{"option_index": 8}), 400)
	for _, option := range []int{0, 1} {
		status(t, request(t, a, student, "POST", base+"/vote", map[string]any{"option_index": option}), 200)
	}
	var votes int64
	a.db.Model(&ChoiceVote{}).Where("lesson_id = ?", poll.ID).Count(&votes)
	if votes != 1 {
		t.Fatal("multiple votes per user")
	}
	w = request(t, a, teacher, "GET", base+"/interaction", nil)
	status(t, w, 200)
	var result struct {
		Counts []int64 `json:"counts"`
	}
	result = unmarshal[struct {
		Counts []int64 `json:"counts"`
	}](t, w)
	if result.Counts[1] != 1 || result.Counts[0] != 0 {
		t.Fatal("incorrect poll tally")
	}
	status(t, request(t, a, teacher, "PATCH", base, map[string]any{"title": poll.Title, "content": "Edited", "duration": 10, "kind": "choice", "data": ActivityData{Options: []string{"Другое", "Вечером"}}}), 409)
	forum := newActivity(t, a, teacher, s.ID, "forum", ActivityData{})
	status(t, request(t, a, student, "POST", fmt.Sprintf("/api/lessons/%d/posts", forum.ID), map[string]string{"body": "Обсуждение в отдельном форуме"}), 201)
	w = request(t, a, student, "GET", fmt.Sprintf("/api/lessons/%d/interaction", forum.ID), nil)
	status(t, w, 200)
	if !strings.Contains(w.Body.String(), "Обсуждение в отдельном форуме") {
		t.Fatal("forum message missing")
	}
	w = request(t, a, student, "GET", fmt.Sprintf("/api/courses/%d", c.ID), nil)
	d := unmarshal[DetailForTest](t, w)
	for _, p := range d.Discussions {
		if p.LessonID == forum.ID {
			t.Fatal("activity forum mixed with course discussion")
		}
	}
}

type DetailForTest struct {
	Discussions []Discussion `json:"discussions"`
}

func fileRequest(t *testing.T, a *app, user, lid uint, name, content string) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	mp := multipart.NewWriter(&b)
	part, err := mp.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte(content))
	mp.Close()
	token := fmt.Sprintf("file-test-%d", user)
	a.db.Where("token = ?", hash(token)).FirstOrCreate(&Session{Token: hash(token), UserID: user, ExpiresAt: time.Now().Add(time.Hour)})
	r := httptest.NewRequest("POST", fmt.Sprintf("/api/lessons/%d/files", lid), &b)
	r.Header.Set("Content-Type", mp.FormDataContentType())
	r.AddCookie(&http.Cookie{Name: "campus_session", Value: token})
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, r)
	return w
}
func TestCourseFileStorageAndAccess(t *testing.T) {
	a := testApp(t)
	t.Setenv("UPLOAD_DIR", t.TempDir())
	teacher := userID(t, a, "teacher")
	student := userID(t, a, "student")
	_, s := firstCourse(t, a)
	l := newActivity(t, a, teacher, s.ID, "file", ActivityData{})
	status(t, fileRequest(t, a, student, l.ID, "test.txt", "Forbidden"), 403)
	status(t, fileRequest(t, a, teacher, l.ID, "test.html", "<script>bad</script>"), 400)
	w := fileRequest(t, a, teacher, l.ID, "Материал.txt", "Учебное содержимое")
	status(t, w, 201)
	resource := unmarshal[Resource](t, w)
	if strings.Contains(w.Body.String(), "Key") {
		t.Fatal("storage key leaked")
	}
	status(t, fileRequest(t, a, teacher, l.ID, "second.txt", "Duplicate"), 409)
	path := fmt.Sprintf("/api/resources/%d/download", resource.ID)
	status(t, request(t, a, 0, "GET", path, nil), 401)
	w = request(t, a, student, "GET", path, nil)
	status(t, w, 200)
	if w.Body.String() != "Учебное содержимое" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal("incorrect file download")
	}
	status(t, request(t, a, teacher, "PATCH", fmt.Sprintf("/api/sections/%d", s.ID), map[string]any{"title": s.Title, "summary": "Скрыто", "hidden": true}), 200)
	status(t, request(t, a, student, "GET", path, nil), 404)
	status(t, request(t, a, teacher, "DELETE", fmt.Sprintf("/api/lessons/%d", l.ID), nil), 200)
	status(t, request(t, a, teacher, "GET", path, nil), 404)
	files, err := os.ReadDir(uploadDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatal("deleted or rejected files left in storage")
	}
}
func TestTaskTopicAssociationRespectsVisibility(t *testing.T) {
	a := testApp(t)
	teacher := userID(t, a, "teacher")
	student := userID(t, a, "student")
	c, s := firstCourse(t, a)
	var assignment Assignment
	var quiz Quiz
	a.db.Where("course_id = ?", c.ID).First(&assignment)
	a.db.Where("course_id = ?", c.ID).First(&quiz)
	status(t, request(t, a, teacher, "PATCH", fmt.Sprintf("/api/assignments/%d/topic", assignment.ID), map[string]any{"section_id": s.ID}), 200)
	status(t, request(t, a, teacher, "PATCH", fmt.Sprintf("/api/quizzes/%d/topic", quiz.ID), map[string]any{"section_id": s.ID}), 200)
	status(t, request(t, a, teacher, "PATCH", fmt.Sprintf("/api/sections/%d", s.ID), map[string]any{"title": s.Title, "hidden": true}), 200)
	status(t, request(t, a, student, "POST", fmt.Sprintf("/api/assignments/%d/submit", assignment.ID), map[string]string{"content": "Не должно пройти"}), 404)
	status(t, request(t, a, student, "POST", fmt.Sprintf("/api/quizzes/%d/attempts", quiz.ID), map[string]any{"answers": []int{1, 0, 1}}), 404)
	w := request(t, a, student, "GET", fmt.Sprintf("/api/courses/%d", c.ID), nil)
	var detail struct {
		Assignments []Assignment `json:"assignments"`
		Quizzes     []Quiz       `json:"quizzes"`
	}
	jsonValue := unmarshal[struct {
		Assignments []Assignment `json:"assignments"`
		Quizzes     []Quiz       `json:"quizzes"`
	}](t, w)
	detail = jsonValue
	if len(detail.Assignments) != 0 || len(detail.Quizzes) != 0 {
		t.Fatal("hidden topic tasks exposed")
	}
	status(t, request(t, a, teacher, "PATCH", fmt.Sprintf("/api/assignments/%d/topic", assignment.ID), map[string]any{"section_id": 0}), 200)
	status(t, request(t, a, student, "POST", fmt.Sprintf("/api/assignments/%d/submit", assignment.ID), map[string]string{"content": "Снова доступно"}), 200)
}

func TestTopicMigrationPreservesExistingCourseContent(t *testing.T) {
	a := testApp(t)
	c, s := firstCourse(t, a)
	student := userID(t, a, "student")
	var as Assignment
	var quiz Quiz
	a.db.Where("course_id = ?", c.ID).First(&as)
	a.db.Where("course_id = ?", c.ID).First(&quiz)
	// Simulate rows from the previous release and an interrupted schema update.
	for _, query := range []string{"UPDATE sections SET hidden = NULL WHERE course_id = ?", "UPDATE assignments SET section_id = NULL WHERE course_id = ?", "UPDATE quizzes SET section_id = NULL WHERE course_id = ?", "UPDATE discussions SET lesson_id = NULL WHERE course_id = ?"} {
		if err := a.db.Exec(query, c.ID).Error; err != nil {
			t.Fatal(err)
		}
	}
	var other Section
	a.db.Where("course_id = ? AND id <> ?", c.ID, s.ID).First(&other)
	a.db.Model(&other).Update("hidden", true)
	if err := backfillTopicDefaults(a.db); err != nil {
		t.Fatal(err)
	}
	w := request(t, a, student, "GET", fmt.Sprintf("/api/courses/%d", c.ID), nil)
	status(t, w, 200)
	detail := unmarshal[struct {
		Sections    []Section    `json:"sections"`
		Assignments []Assignment `json:"assignments"`
		Quizzes     []Quiz       `json:"quizzes"`
		Discussions []Discussion `json:"discussions"`
	}](t, w)
	if len(detail.Sections) != 2 || len(detail.Assignments) != 1 || len(detail.Quizzes) != 1 || len(detail.Discussions) != 1 {
		t.Fatal("upgrade hid existing course content")
	}
	for _, section := range detail.Sections {
		if section.ID == other.ID {
			t.Fatal("backfill overwrote explicitly hidden topic")
		}
	}
}
