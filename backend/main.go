package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type app struct {
	db      *gorm.DB
	secure  bool
	loginMu sync.Mutex
	logins  map[string]loginLimit
}
type loginLimit struct {
	Count int
	Reset time.Time
}
type userKey struct{}

func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	reply(w, status, map[string]string{"error": message})
}
func decode(w http.ResponseWriter, r *http.Request, dest any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(dest); err != nil {
		fail(w, 400, "Проверьте формат и размер данных")
		return false
	}
	return true
}
func current(r *http.Request) User { return r.Context().Value(userKey{}).(User) }
func id(r *http.Request, name string) uint {
	n, _ := strconv.ParseUint(r.PathValue(name), 10, 32)
	return uint(n)
}
func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func (a *app) result(w http.ResponseWriter, err error) bool {
	if err != nil {
		log.Printf("database operation: %v", err)
		fail(w, 500, "Не удалось сохранить данные")
		return false
	}
	return true
}
func (a *app) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("campus_session")
		if err != nil {
			fail(w, 401, "Войдите в аккаунт")
			return
		}
		var s Session
		if a.db.Where("token = ? AND expires_at > ?", hash(c.Value), time.Now()).First(&s).Error != nil {
			fail(w, 401, "Сессия истекла")
			return
		}
		var u User
		if a.db.First(&u, s.UserID).Error != nil {
			fail(w, 401, "Пользователь не найден")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	}
}
func (a *app) session(w http.ResponseWriter, u User) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		fail(w, 500, "Не удалось создать сессию")
		return
	}
	token := hex.EncodeToString(b)
	if !a.result(w, a.db.Create(&Session{Token: hash(token), UserID: u.ID, ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}).Error) {
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "campus_session", Value: token, Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode, MaxAge: 7 * 24 * 3600})
	reply(w, 200, u)
}
func (a *app) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	key := strings.ToLower(strings.TrimSpace(in.Email))
	a.loginMu.Lock()
	lim := a.logins[key]
	if time.Now().After(lim.Reset) {
		lim = loginLimit{Reset: time.Now().Add(10 * time.Minute)}
	}
	lim.Count++
	a.logins[key] = lim
	if len(a.logins) > 10000 {
		for k, v := range a.logins {
			if time.Now().After(v.Reset) {
				delete(a.logins, k)
			}
		}
	}
	a.loginMu.Unlock()
	if lim.Count > 20 {
		fail(w, 429, "Слишком много попыток. Повторите через 10 минут")
		return
	}
	var u User
	err := a.db.Where("email = ?", key).First(&u).Error
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.Password)) != nil {
		fail(w, 401, "Неверная почта или пароль")
		return
	}
	a.session(w, u)
}
func (a *app) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if len(in.Name) < 2 || len(in.Name) > 100 || !strings.Contains(in.Email, "@") || len(in.Email) > 254 || len(in.Password) < 8 || len(in.Password) > 72 {
		fail(w, 400, "Укажите имя, корректную почту и пароль от 8 до 72 символов")
		return
	}
	h, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if !a.result(w, err) {
		return
	}
	u := User{Name: in.Name, Email: in.Email, Password: string(h), Role: "student"}
	if err := a.db.Create(&u).Error; err != nil {
		fail(w, 409, "Не удалось создать аккаунт: возможно, почта уже занята")
		return
	}
	a.session(w, u)
}
func (a *app) access(w http.ResponseWriter, r *http.Request, courseID uint, write bool) (Course, bool) {
	var c Course
	if a.db.Preload("Teacher").First(&c, courseID).Error != nil {
		fail(w, 404, "Курс не найден")
		return c, false
	}
	u := current(r)
	if u.Role == "admin" || (u.Role == "teacher" && c.TeacherID == u.ID) {
		return c, true
	}
	if write {
		fail(w, 403, "Только преподаватель этого курса может изменять материалы")
		return c, false
	}
	var n int64
	a.db.Model(&Enrollment{}).Where("course_id = ? AND user_id = ?", c.ID, u.ID).Count(&n)
	if n == 0 || !c.Published {
		fail(w, 403, "Запишитесь на опубликованный курс")
		return c, false
	}
	return c, true
}
func (a *app) courses(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	var courses []Course
	q := a.db.Preload("Teacher").Order("id ASC")
	if u.Role != "admin" {
		q = q.Where("published = ? OR teacher_id = ?", true, u.ID)
	}
	if !a.result(w, q.Find(&courses).Error) {
		return
	}
	type item struct {
		Course
		Enrolled bool  `json:"enrolled"`
		Progress int64 `json:"progress"`
		Lessons  int64 `json:"lessons"`
		Students int64 `json:"students"`
	}
	out := []item{}
	for _, c := range courses {
		var e, n, total, students int64
		a.db.Model(&Enrollment{}).Where("course_id = ? AND user_id = ?", c.ID, u.ID).Count(&e)
		a.db.Model(&Completion{}).Where("course_id = ? AND user_id = ?", c.ID, u.ID).Count(&n)
		a.db.Model(&Lesson{}).Where("course_id = ?", c.ID).Count(&total)
		a.db.Model(&Enrollment{}).Where("course_id = ?", c.ID).Count(&students)
		out = append(out, item{c, e > 0, n, total, students})
	}
	reply(w, 200, out)
}
func validCourse(c Course) bool {
	return len(strings.TrimSpace(c.Title)) >= 3 && len(c.Title) <= 200 && len(c.Description) <= 10000 && len(c.Category) <= 100 && (c.Color == "violet" || c.Color == "blue" || c.Color == "green" || c.Color == "orange")
}
func (a *app) saveCourse(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if u.Role == "student" {
		fail(w, 403, "Недостаточно прав")
		return
	}
	var in struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Color       string `json:"color"`
		Published   bool   `json:"published"`
	}
	if !decode(w, r, &in) {
		return
	}
	c := Course{Title: in.Title, Description: in.Description, Category: in.Category, Color: in.Color, Published: in.Published, TeacherID: u.ID}
	if !validCourse(c) {
		fail(w, 400, "Проверьте название, описание и цвет курса")
		return
	}
	if r.Method == "PATCH" {
		old, ok := a.access(w, r, id(r, "course"), true)
		if !ok {
			return
		}
		c.ID = old.ID
		c.TeacherID = old.TeacherID
		c.CreatedAt = old.CreatedAt
	}
	if !a.result(w, a.db.Save(&c).Error) {
		return
	}
	a.db.Preload("Teacher").First(&c, c.ID)
	reply(w, 200, c)
}
func (a *app) enroll(w http.ResponseWriter, r *http.Request) {
	var c Course
	if a.db.Where("published = ?", true).First(&c, id(r, "course")).Error != nil {
		fail(w, 404, "Курс не найден")
		return
	}
	u := current(r)
	e := Enrollment{CourseID: c.ID, UserID: u.ID}
	if !a.result(w, a.db.Where("course_id = ? AND user_id = ?", c.ID, u.ID).FirstOrCreate(&e).Error) {
		return
	}
	if !a.result(w, a.db.Model(&e).Update("source", "manual").Error) {
		return
	}
	e.Source = "manual"
	reply(w, 200, e)
}
func (a *app) detail(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), false)
	if !ok {
		return
	}
	u := current(r)
	sections := []Section{}
	a.db.Preload("Lessons", func(db *gorm.DB) *gorm.DB { return db.Order("position, id") }).Where("course_id = ?", c.ID).Order("position, id").Find(&sections)
	assignments := []Assignment{}
	a.db.Where("course_id = ?", c.ID).Order("due_at, id").Find(&assignments)
	quizzes := []Quiz{}
	a.db.Preload("Questions", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).Where("course_id = ?", c.ID).Find(&quizzes)
	completed := []uint{}
	a.db.Model(&Completion{}).Where("course_id = ? AND user_id = ?", c.ID, u.ID).Pluck("lesson_id", &completed)
	announcements := []Announcement{}
	a.db.Where("course_id = ?", c.ID).Order("created_at DESC").Find(&announcements)
	posts := []Discussion{}
	a.db.Preload("User").Where("course_id = ?", c.ID).Order("created_at DESC").Limit(100).Find(&posts)
	submissions := []Submission{}
	q := a.db.Preload("User").Joins("JOIN assignments ON assignments.id = submissions.assignment_id").Where("assignments.course_id = ?", c.ID)
	if u.Role != "admin" && c.TeacherID != u.ID {
		q = q.Where("submissions.user_id = ?", u.ID)
	}
	q.Order("submissions.updated_at DESC").Find(&submissions)
	attempts := []Attempt{}
	aq := a.db.Joins("JOIN quizzes ON quizzes.id = attempts.quiz_id").Where("quizzes.course_id = ?", c.ID)
	if u.Role != "admin" && c.TeacherID != u.ID {
		aq = aq.Where("attempts.user_id = ?", u.ID)
	}
	aq.Order("attempts.id DESC").Find(&attempts)
	reply(w, 200, map[string]any{"course": c, "sections": sections, "assignments": assignments, "quizzes": quizzes, "completed": completed, "announcements": announcements, "discussions": posts, "submissions": submissions, "attempts": attempts})
}
func (a *app) addSection(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	var in struct {
		Title string `json:"title"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Title)) < 2 || len(in.Title) > 200 {
		fail(w, 400, "Укажите название модуля")
		return
	}
	var n int64
	a.db.Model(&Section{}).Where("course_id = ?", c.ID).Count(&n)
	s := Section{CourseID: c.ID, Title: in.Title, Position: int(n) + 1, Lessons: []Lesson{}}
	if a.result(w, a.db.Create(&s).Error) {
		reply(w, 201, s)
	}
}
func (a *app) addLesson(w http.ResponseWriter, r *http.Request) {
	var s Section
	if a.db.First(&s, id(r, "section")).Error != nil {
		fail(w, 404, "Модуль не найден")
		return
	}
	if _, ok := a.access(w, r, s.CourseID, true); !ok {
		return
	}
	var in struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Duration int    `json:"duration"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Title)) < 2 || len(in.Title) > 200 || len(in.Content) > 100000 || in.Duration < 1 || in.Duration > 1000 {
		fail(w, 400, "Проверьте название, текст и длительность урока")
		return
	}
	var n int64
	a.db.Model(&Lesson{}).Where("section_id = ?", s.ID).Count(&n)
	l := Lesson{SectionID: s.ID, CourseID: s.CourseID, Title: in.Title, Content: in.Content, Duration: in.Duration, Position: int(n) + 1}
	if a.result(w, a.db.Create(&l).Error) {
		reply(w, 201, l)
	}
}
func (a *app) complete(w http.ResponseWriter, r *http.Request) {
	var l Lesson
	if a.db.First(&l, id(r, "lesson")).Error != nil {
		fail(w, 404, "Урок не найден")
		return
	}
	if _, ok := a.access(w, r, l.CourseID, false); !ok {
		return
	}
	u := current(r)
	c := Completion{LessonID: l.ID, CourseID: l.CourseID, UserID: u.ID}
	if !a.result(w, a.db.Where("lesson_id = ? AND user_id = ?", l.ID, u.ID).FirstOrCreate(&c).Error) {
		return
	}
	reply(w, 200, map[string]bool{"completed": true})
}
func (a *app) addAssignment(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	var in struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		DueAt       time.Time `json:"due_at"`
		MaxScore    int       `json:"max_score"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Title)) < 2 || len(in.Title) > 200 || len(in.Description) > 20000 || in.MaxScore < 1 || in.MaxScore > 1000 || in.DueAt.IsZero() {
		fail(w, 400, "Проверьте название, срок и максимальный балл")
		return
	}
	v := Assignment{CourseID: c.ID, Title: in.Title, Description: in.Description, DueAt: in.DueAt, MaxScore: in.MaxScore}
	if a.result(w, a.db.Create(&v).Error) {
		reply(w, 201, v)
	}
}
func (a *app) submit(w http.ResponseWriter, r *http.Request) {
	var as Assignment
	if a.db.First(&as, id(r, "assignment")).Error != nil {
		fail(w, 404, "Задание не найдено")
		return
	}
	if _, ok := a.access(w, r, as.CourseID, false); !ok {
		return
	}
	u := current(r)
	if u.Role != "student" {
		fail(w, 403, "Отправка доступна студентам")
		return
	}
	var in struct {
		Content string `json:"content"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Content)) < 3 || len(in.Content) > 50000 {
		fail(w, 400, "Ответ должен содержать от 3 до 50000 символов")
		return
	}
	var s Submission
	err := a.db.Where("assignment_id = ? AND user_id = ?", as.ID, u.ID).First(&s).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.result(w, err)
		return
	}
	if s.Score != nil {
		fail(w, 409, "Работа уже оценена")
		return
	}
	s.AssignmentID = as.ID
	s.UserID = u.ID
	s.Content = in.Content
	if !a.result(w, a.db.Save(&s).Error) {
		return
	}
	reply(w, 200, s)
}
func (a *app) grade(w http.ResponseWriter, r *http.Request) {
	var s Submission
	if a.db.First(&s, id(r, "submission")).Error != nil {
		fail(w, 404, "Работа не найдена")
		return
	}
	var as Assignment
	if !a.result(w, a.db.First(&as, s.AssignmentID).Error) {
		return
	}
	if _, ok := a.access(w, r, as.CourseID, true); !ok {
		return
	}
	var in struct {
		Score    *int   `json:"score"`
		Feedback string `json:"feedback"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Score == nil || *in.Score < 0 || *in.Score > as.MaxScore || len(in.Feedback) > 10000 {
		fail(w, 400, "Баллы вне допустимого диапазона")
		return
	}
	s.Score = in.Score
	s.Feedback = in.Feedback
	if a.result(w, a.db.Save(&s).Error) {
		reply(w, 200, s)
	}
}
func (a *app) addQuiz(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	var in struct {
		Title     string `json:"title"`
		Questions []struct {
			Prompt       string   `json:"prompt"`
			Options      []string `json:"options"`
			CorrectIndex int      `json:"correct_index"`
		} `json:"questions"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Title)) < 2 || len(in.Title) > 200 || len(in.Questions) < 1 || len(in.Questions) > 50 {
		fail(w, 400, "Добавьте название и от 1 до 50 вопросов")
		return
	}
	q := Quiz{Title: in.Title, CourseID: c.ID}
	for _, v := range in.Questions {
		if len(v.Prompt) < 2 || len(v.Prompt) > 2000 || len(v.Options) < 2 || len(v.Options) > 8 || v.CorrectIndex < 0 || v.CorrectIndex >= len(v.Options) {
			fail(w, 400, "Проверьте вопросы и правильные ответы")
			return
		}
		for _, o := range v.Options {
			if strings.TrimSpace(o) == "" || len(o) > 1000 {
				fail(w, 400, "Варианты ответа не должны быть пустыми")
				return
			}
		}
		q.Questions = append(q.Questions, Question{Prompt: v.Prompt, Options: v.Options, CorrectIndex: v.CorrectIndex})
	}
	if a.result(w, a.db.Create(&q).Error) {
		reply(w, 201, q)
	}
}
func (a *app) attempt(w http.ResponseWriter, r *http.Request) {
	var q Quiz
	if a.db.Preload("Questions", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).First(&q, id(r, "quiz")).Error != nil {
		fail(w, 404, "Тест не найден")
		return
	}
	if _, ok := a.access(w, r, q.CourseID, false); !ok {
		return
	}
	u := current(r)
	if u.Role != "student" {
		fail(w, 403, "Прохождение доступно студентам")
		return
	}
	var in struct {
		Answers []int `json:"answers"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Answers) != len(q.Questions) {
		fail(w, 400, "Ответьте на все вопросы")
		return
	}
	score := 0
	for i, question := range q.Questions {
		if in.Answers[i] < 0 || in.Answers[i] >= len(question.Options) {
			fail(w, 400, "Недопустимый вариант ответа")
			return
		}
		if question.CorrectIndex == in.Answers[i] {
			score++
		}
	}
	v := Attempt{QuizID: q.ID, UserID: u.ID, Score: score, Total: len(q.Questions), Answers: in.Answers}
	err := a.db.Transaction(func(tx *gorm.DB) error { // Serialise attempts by user to enforce the limit under concurrency.
		var user User
		if err := tx.Raw("SELECT * FROM users WHERE id = ? FOR UPDATE", u.ID).Scan(&user).Error; err != nil {
			return err
		}
		var n int64
		tx.Model(&Attempt{}).Where("quiz_id = ? AND user_id = ?", q.ID, u.ID).Count(&n)
		if n >= 3 {
			return fmt.Errorf("attempt_limit")
		}
		return tx.Create(&v).Error
	})
	if err != nil && err.Error() == "attempt_limit" {
		fail(w, 409, "Использованы все три попытки")
		return
	}
	if a.result(w, err) {
		reply(w, 201, v)
	}
}
func (a *app) announcement(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	var in struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Title) < 2 || len(in.Title) > 200 || len(in.Body) < 2 || len(in.Body) > 20000 {
		fail(w, 400, "Укажите заголовок и текст объявления")
		return
	}
	v := Announcement{CourseID: c.ID, Title: in.Title, Body: in.Body}
	if a.result(w, a.db.Create(&v).Error) {
		reply(w, 201, v)
	}
}
func (a *app) discuss(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), false)
	if !ok {
		return
	}
	var in struct {
		Body string `json:"body"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Body)) < 2 || len(in.Body) > 10000 {
		fail(w, 400, "Сообщение должно содержать от 2 до 10000 символов")
		return
	}
	v := Discussion{CourseID: c.ID, UserID: current(r).ID, Body: in.Body}
	if !a.result(w, a.db.Create(&v).Error) {
		return
	}
	a.db.Preload("User").First(&v, v.ID)
	reply(w, 201, v)
}
func (a *app) gradebook(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), false)
	if !ok {
		return
	}
	u := current(r)
	enrollments := []Enrollment{}
	q := a.db.Preload("User").Where("course_id = ?", c.ID)
	if u.Role != "admin" && c.TeacherID != u.ID {
		q = q.Where("user_id = ?", u.ID)
	}
	q.Order("id").Find(&enrollments)
	as := []Assignment{}
	a.db.Where("course_id = ?", c.ID).Find(&as)
	quizzes := []Quiz{}
	a.db.Where("course_id = ?", c.ID).Find(&quizzes)
	var lessons int64
	a.db.Model(&Lesson{}).Where("course_id = ?", c.ID).Count(&lessons)
	rows := []map[string]any{}
	for _, e := range enrollments {
		var complete int64
		a.db.Model(&Completion{}).Where("course_id = ? AND user_id = ?", c.ID, e.UserID).Count(&complete)
		subs := []Submission{}
		a.db.Where("user_id = ? AND assignment_id IN (?)", e.UserID, a.db.Model(&Assignment{}).Select("id").Where("course_id = ?", c.ID)).Find(&subs)
		attempts := []Attempt{}
		a.db.Where("user_id = ? AND quiz_id IN (?)", e.UserID, a.db.Model(&Quiz{}).Select("id").Where("course_id = ?", c.ID)).Find(&attempts)
		rows = append(rows, map[string]any{"user": e.User, "completed": complete, "lessons": lessons, "submissions": subs, "attempts": attempts})
	}
	reply(w, 200, map[string]any{"rows": rows, "assignments": as, "quizzes": quizzes})
}
func (a *app) users(w http.ResponseWriter, r *http.Request) {
	if current(r).Role != "admin" {
		fail(w, 403, "Недостаточно прав")
		return
	}
	out := []User{}
	if a.result(w, a.db.Order("id").Limit(500).Find(&out).Error) {
		reply(w, 200, out)
	}
}
func (a *app) role(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if u.Role != "admin" {
		fail(w, 403, "Недостаточно прав")
		return
	}
	var in struct {
		Role string `json:"role"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Role != "admin" && in.Role != "teacher" && in.Role != "student" {
		fail(w, 400, "Неизвестная роль")
		return
	}
	if id(r, "user") == u.ID {
		fail(w, 400, "Нельзя менять собственную роль")
		return
	}
	var target User
	if a.db.First(&target, id(r, "user")).Error != nil {
		fail(w, 404, "Пользователь не найден")
		return
	}
	if a.result(w, a.db.Model(&target).Update("role", in.Role).Error) {
		reply(w, 200, target)
	}
}
func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		sql, err := a.db.DB()
		if err != nil || sql.PingContext(r.Context()) != nil {
			fail(w, 503, "База данных недоступна")
			return
		}
		reply(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/auth/login", a.login)
	mux.HandleFunc("POST /api/auth/register", a.register)
	mux.HandleFunc("GET /api/auth/me", a.auth(func(w http.ResponseWriter, r *http.Request) { reply(w, 200, current(r)) }))
	mux.HandleFunc("POST /api/auth/logout", a.auth(func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Cookie("campus_session")
		if !a.result(w, a.db.Where("token = ?", hash(c.Value)).Delete(&Session{}).Error) {
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "campus_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode})
		reply(w, 200, map[string]bool{"ok": true})
	}))
	for pattern, handler := range map[string]http.HandlerFunc{"GET /api/courses": a.courses, "POST /api/courses": a.saveCourse, "PATCH /api/courses/{course}": a.saveCourse, "GET /api/courses/{course}": a.detail, "POST /api/courses/{course}/enroll": a.enroll, "POST /api/courses/{course}/sections": a.addSection, "POST /api/sections/{section}/lessons": a.addLesson, "POST /api/lessons/{lesson}/complete": a.complete, "POST /api/courses/{course}/assignments": a.addAssignment, "POST /api/assignments/{assignment}/submit": a.submit, "PATCH /api/submissions/{submission}/grade": a.grade, "POST /api/courses/{course}/quizzes": a.addQuiz, "POST /api/quizzes/{quiz}/attempts": a.attempt, "POST /api/courses/{course}/announcements": a.announcement, "POST /api/courses/{course}/discussions": a.discuss, "GET /api/courses/{course}/grades": a.gradebook, "GET /api/users": a.users, "PATCH /api/users/{user}/role": a.role} {
		mux.HandleFunc(pattern, a.auth(handler))
	}
	a.extendedRoutes(mux)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "Маршрут не найден") })
	mux.Handle("/", http.FileServer(http.Dir(env("STATIC_DIR", "../frontend/dist"))))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			if origin := r.Header.Get("Origin"); origin != "" {
				o, err := url.Parse(origin)
				if err != nil || o.Host != r.Host {
					fail(w, 403, "Запрос с другого сайта запрещён")
					return
				}
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				fail(w, 403, "Запрос с другого сайта запрещён")
				return
			}
		}
		defer func() {
			if err := recover(); err != nil {
				log.Printf("request panic: %v", err)
				fail(w, 500, "Внутренняя ошибка сервера")
			}
		}()
		mux.ServeHTTP(w, r)
	})
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Error)})
	if err != nil {
		log.Fatal("PostgreSQL connection failed")
	}
	sql, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	sql.SetMaxOpenConns(10)
	sql.SetMaxIdleConns(5)
	sql.SetConnMaxLifetime(time.Hour)
	if err = db.AutoMigrate(&User{}, &Session{}, &Course{}, &Enrollment{}, &Section{}, &Lesson{}, &Completion{}, &Assignment{}, &Submission{}, &Quiz{}, &Question{}, &Attempt{}, &Announcement{}, &Discussion{}, &Cohort{}, &CohortMember{}, &CourseCohort{}, &BankItem{}); err != nil {
		log.Fatal(err)
	}
	if os.Getenv("DEMO_SEED") == "true" {
		if err = seed(db); err != nil {
			log.Fatal(err)
		}
	}
	a := &app{db: db, secure: os.Getenv("COOKIE_SECURE") == "true", logins: make(map[string]loginLimit)}
	server := &http.Server{Addr: env("LISTEN_ADDR", "127.0.0.1:8080"), Handler: a.routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		log.Printf("Campus listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	sql.Close()
}
