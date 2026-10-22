package main

import (
	"gorm.io/gorm"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func validURL(value string) bool {
	u, e := url.Parse(value)
	return e == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}
func validateBank(v BankItem) string {
	if len(strings.TrimSpace(v.Title)) < 2 || len(v.Title) > 200 || len(v.Category) > 100 || len(v.Body) > 100000 {
		return "Проверьте название и размер материала"
	}
	switch v.Kind {
	case "page", "label":
	case "url":
		if !validURL(v.URL) {
			return "Укажите ссылку HTTP или HTTPS"
		}
	case "assignment":
		if v.MaxScore < 1 || v.MaxScore > 1000 {
			return "Укажите максимальный балл от 1 до 1000"
		}
	case "quiz":
		if len(v.Questions) < 1 || len(v.Questions) > 50 {
			return "Добавьте от 1 до 50 вопросов"
		}
		for _, q := range v.Questions {
			if len(q.Prompt) < 2 || len(q.Prompt) > 2000 || len(q.Options) < 2 || len(q.Options) > 8 || q.CorrectIndex < 0 || q.CorrectIndex >= len(q.Options) {
				return "Проверьте вопросы и правильные ответы"
			}
			for _, o := range q.Options {
				if strings.TrimSpace(o) == "" || len(o) > 1000 {
					return "Варианты ответа должны содержать от 1 до 1000 символов"
				}
			}
		}
	default:
		return "Неизвестный тип элемента"
	}
	return ""
}
func (a *app) bank(w http.ResponseWriter, r *http.Request) {
	if !staff(w, r) {
		return
	}
	out := []BankItem{}
	if a.result(w, a.db.Preload("Author").Order("id DESC").Find(&out).Error) {
		reply(w, 200, out)
	}
}
func (a *app) createBank(w http.ResponseWriter, r *http.Request) {
	if !staff(w, r) {
		return
	}
	var in struct {
		Title     string         `json:"title"`
		Category  string         `json:"category"`
		Kind      string         `json:"kind"`
		Body      string         `json:"body"`
		URL       string         `json:"url"`
		Duration  int            `json:"duration"`
		MaxScore  int            `json:"max_score"`
		Questions []BankQuestion `json:"questions"`
	}
	if !decode(w, r, &in) {
		return
	}
	v := BankItem{Title: in.Title, Category: in.Category, Kind: in.Kind, Body: in.Body, URL: in.URL, Duration: in.Duration, MaxScore: in.MaxScore, Questions: in.Questions, AuthorID: current(r).ID}
	if msg := validateBank(v); msg != "" {
		fail(w, 400, msg)
		return
	}
	if a.result(w, a.db.Create(&v).Error) {
		reply(w, 201, v)
	}
}
func (a *app) deleteBank(w http.ResponseWriter, r *http.Request) {
	if !staff(w, r) {
		return
	}
	var v BankItem
	if a.db.First(&v, id(r, "item")).Error != nil {
		fail(w, 404, "Материал не найден")
		return
	}
	if current(r).Role != "admin" && v.AuthorID != current(r).ID {
		fail(w, 403, "Удалить материал может его автор или администратор")
		return
	}
	if a.result(w, a.db.Delete(&v).Error) {
		reply(w, 200, map[string]bool{"ok": true})
	}
}
func copyBank(tx *gorm.DB, v BankItem, courseID, sectionID uint, due time.Time) error {
	switch v.Kind {
	case "assignment":
		if due.IsZero() {
			due = time.Now().AddDate(0, 0, 7)
		}
		return tx.Create(&Assignment{CourseID: courseID, Title: v.Title, Description: v.Body, MaxScore: v.MaxScore, DueAt: due}).Error
	case "quiz":
		q := Quiz{CourseID: courseID, Title: v.Title}
		for _, question := range v.Questions {
			q.Questions = append(q.Questions, Question{Prompt: question.Prompt, Options: question.Options, CorrectIndex: question.CorrectIndex})
		}
		return tx.Create(&q).Error
	default:
		if sectionID == 0 {
			s := Section{CourseID: courseID, Title: "Материалы из банка", Position: 1000}
			if err := tx.Where("course_id = ? AND title = ?", courseID, s.Title).FirstOrCreate(&s).Error; err != nil {
				return err
			}
			sectionID = s.ID
		}
		var n int64
		tx.Model(&Lesson{}).Where("section_id = ?", sectionID).Count(&n)
		duration := v.Duration
		if duration < 1 {
			duration = 10
		}
		return tx.Create(&Lesson{CourseID: courseID, SectionID: sectionID, Title: v.Title, Content: v.Body, Kind: v.Kind, URL: v.URL, Duration: duration, Position: int(n) + 1}).Error
	}
}
func (a *app) useBank(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	var v BankItem
	if a.db.First(&v, id(r, "item")).Error != nil {
		fail(w, 404, "Материал не найден")
		return
	}
	var in struct {
		SectionID uint      `json:"section_id"`
		DueAt     time.Time `json:"due_at"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.SectionID != 0 {
		var n int64
		a.db.Model(&Section{}).Where("id = ? AND course_id = ?", in.SectionID, c.ID).Count(&n)
		if n == 0 {
			fail(w, 400, "Модуль не относится к курсу")
			return
		}
	}
	if a.result(w, a.db.Transaction(func(tx *gorm.DB) error { return copyBank(tx, v, c.ID, in.SectionID, in.DueAt) })) {
		reply(w, 201, map[string]bool{"ok": true})
	}
}
