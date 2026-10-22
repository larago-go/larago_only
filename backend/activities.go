package main

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func courseAvailability(c Course) string {
	now := time.Now()
	if c.StartAt != nil && now.Before(*c.StartAt) {
		return "Курс ещё не начался"
	}
	if c.EndAt != nil && now.After(*c.EndAt) {
		return "Период доступа к курсу завершён"
	}
	return ""
}
func (a *app) validSection(w http.ResponseWriter, cid, sid uint) bool {
	if sid == 0 {
		return true
	}
	var n int64
	a.db.Model(&Section{}).Where("id = ? AND course_id = ?", sid, cid).Count(&n)
	if n != 1 {
		fail(w, 400, "Тема не относится к этому курсу")
		return false
	}
	return true
}
func (a *app) topicAccess(w http.ResponseWriter, r *http.Request, cid, sid uint, write bool) (Course, bool) {
	c, ok := a.access(w, r, cid, write)
	if !ok {
		return c, false
	}
	if sid == 0 {
		return c, true
	}
	var s Section
	if a.db.Where("id = ? AND course_id = ?", sid, cid).First(&s).Error != nil {
		fail(w, 404, "Тема не найдена")
		return c, false
	}
	u := current(r)
	if s.Hidden && u.Role != "admin" && !(u.Role == "teacher" && c.TeacherID == u.ID) {
		fail(w, 404, "Элемент недоступен")
		return c, false
	}
	return c, true
}
func (a *app) visibleActivities(r *http.Request, q *gorm.DB, c Course) *gorm.DB {
	u := current(r)
	if u.Role == "admin" || (u.Role == "teacher" && c.TeacherID == u.ID) {
		return q
	}
	return q.Where("section_id = 0 OR section_id IN (?)", a.db.Model(&Section{}).Select("id").Where("course_id = ? AND hidden = ?", c.ID, false))
}
func (a *app) activityRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/lessons/{lesson}/bank", a.auth(a.saveActivityToBank))
	m.HandleFunc("PATCH /api/assignments/{assignment}/topic", a.auth(func(w http.ResponseWriter, r *http.Request) { a.moveTask(w, r, false) }))
	m.HandleFunc("PATCH /api/quizzes/{quiz}/topic", a.auth(func(w http.ResponseWriter, r *http.Request) { a.moveTask(w, r, true) }))
	for p, h := range map[string]http.HandlerFunc{
		"PATCH /api/sections/{section}": a.editSection, "DELETE /api/sections/{section}": a.deleteSection, "PUT /api/courses/{course}/sections/order": a.orderSections,
		"PATCH /api/lessons/{lesson}": a.editLesson, "DELETE /api/lessons/{lesson}": a.deleteLesson,
		"GET /api/lessons/{lesson}/interaction": a.interaction, "POST /api/lessons/{lesson}/vote": a.vote, "POST /api/lessons/{lesson}/posts": a.activityPost,
		"POST /api/lessons/{lesson}/files": a.uploadFile, "DELETE /api/resources/{resource}": a.deleteFile, "GET /api/resources/{resource}/download": a.downloadFile,
	} {
		m.HandleFunc(p, a.auth(h))
	}
}
func (a *app) addSection(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	var in struct {
		Title   string `json:"title"`
		Summary string `json:"summary"`
		Hidden  bool   `json:"hidden"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Title)) < 2 || len(in.Title) > 200 || len(in.Summary) > 10000 {
		fail(w, 400, "Проверьте название и описание темы")
		return
	}
	var max int
	a.db.Model(&Section{}).Where("course_id = ?", c.ID).Select("COALESCE(MAX(position), 0)").Scan(&max)
	s := Section{CourseID: c.ID, Title: in.Title, Summary: in.Summary, Hidden: in.Hidden, Position: max + 1, Lessons: []Lesson{}}
	if a.result(w, a.db.Create(&s).Error) {
		reply(w, 201, s)
	}
}
func (a *app) editSection(w http.ResponseWriter, r *http.Request) {
	var s Section
	if a.db.First(&s, id(r, "section")).Error != nil {
		fail(w, 404, "Тема не найдена")
		return
	}
	if _, ok := a.access(w, r, s.CourseID, true); !ok {
		return
	}
	var in struct {
		Title   string `json:"title"`
		Summary string `json:"summary"`
		Hidden  bool   `json:"hidden"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Title)) < 2 || len(in.Title) > 200 || len(in.Summary) > 10000 {
		fail(w, 400, "Проверьте название и описание темы")
		return
	}
	s.Title = in.Title
	s.Summary = in.Summary
	s.Hidden = in.Hidden
	if a.result(w, a.db.Omit("Lessons").Save(&s).Error) {
		reply(w, 200, s)
	}
}
func (a *app) orderSections(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	var in struct {
		IDs []uint `json:"ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	err := a.db.Transaction(func(tx *gorm.DB) error {
		var locked Course
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, c.ID).Error; err != nil {
			return err
		}
		rows := []Section{}
		if err := tx.Where("course_id = ?", c.ID).Find(&rows).Error; err != nil {
			return err
		}
		if len(in.IDs) != len(rows) {
			return fmt.Errorf("invalid_order")
		}
		valid := map[uint]bool{}
		for _, s := range rows {
			valid[s.ID] = true
		}
		for i, sid := range in.IDs {
			if !valid[sid] {
				return fmt.Errorf("invalid_order")
			}
			delete(valid, sid)
			if err := tx.Model(&Section{}).Where("id = ?", sid).Update("position", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil && err.Error() == "invalid_order" {
		fail(w, 400, "Передайте все темы курса без повторений")
		return
	}
	if a.result(w, err) {
		reply(w, 200, map[string]bool{"ok": true})
	}
}
func (a *app) deleteSection(w http.ResponseWriter, r *http.Request) {
	var s Section
	if a.db.First(&s, id(r, "section")).Error != nil {
		fail(w, 404, "Тема не найдена")
		return
	}
	if _, ok := a.access(w, r, s.CourseID, true); !ok {
		return
	}
	var n, n2, n3 int64
	a.db.Model(&Lesson{}).Where("section_id = ?", s.ID).Count(&n)
	a.db.Model(&Assignment{}).Where("section_id = ?", s.ID).Count(&n2)
	a.db.Model(&Quiz{}).Where("section_id = ?", s.ID).Count(&n3)
	if n+n2+n3 > 0 {
		fail(w, 409, "В теме есть элементы. Переместите или удалите их перед удалением темы")
		return
	}
	if a.result(w, a.db.Delete(&s).Error) {
		reply(w, 200, map[string]bool{"ok": true})
	}
}

type lessonInput struct {
	Title     string       `json:"title"`
	Content   string       `json:"content"`
	Kind      string       `json:"kind"`
	URL       string       `json:"url"`
	Duration  int          `json:"duration"`
	SectionID uint         `json:"section_id"`
	Data      ActivityData `json:"data"`
}

func validateActivity(v lessonInput) string {
	if len(strings.TrimSpace(v.Title)) < 2 || len(v.Title) > 200 || len(v.Content) > 100000 || v.Duration < 1 || v.Duration > 1000 {
		return "Проверьте название, содержание и длительность элемента"
	}
	switch v.Kind {
	case "page", "label", "file", "folder", "forum":
	case "url", "video":
		if !validURL(v.URL) || len(v.URL) > 4000 {
			return "Укажите ссылку HTTP или HTTPS"
		}
	case "book":
		if len(v.Data.Chapters) < 1 || len(v.Data.Chapters) > 100 {
			return "Книга должна содержать от 1 до 100 глав"
		}
		total := 0
		for _, c := range v.Data.Chapters {
			if len(strings.TrimSpace(c.Title)) < 2 || len(c.Title) > 200 || strings.TrimSpace(c.Content) == "" {
				return "У каждой главы должны быть название и текст"
			}
			total += len(c.Content)
		}
		if total > 100000 {
			return "Общий текст книги превышает 100000 символов"
		}
	case "glossary":
		if len(v.Data.Entries) < 1 || len(v.Data.Entries) > 200 {
			return "Добавьте от 1 до 200 терминов"
		}
		for _, e := range v.Data.Entries {
			if len(strings.TrimSpace(e.Term)) < 1 || len(e.Term) > 200 || len(strings.TrimSpace(e.Definition)) < 1 || len(e.Definition) > 10000 {
				return "Проверьте термины и определения"
			}
		}
	case "choice":
		if len(v.Data.Options) < 2 || len(v.Data.Options) > 10 {
			return "Опрос должен содержать от 2 до 10 вариантов"
		}
		seen := map[string]bool{}
		for _, o := range v.Data.Options {
			k := strings.ToLower(strings.TrimSpace(o))
			if k == "" || len(o) > 200 || seen[k] {
				return "Варианты опроса должны быть заполнены и различаться"
			}
			seen[k] = true
		}
	default:
		return "Неизвестный тип элемента"
	}
	return ""
}
func (a *app) addLesson(w http.ResponseWriter, r *http.Request) {
	var s Section
	if a.db.First(&s, id(r, "section")).Error != nil {
		fail(w, 404, "Тема не найдена")
		return
	}
	if _, ok := a.access(w, r, s.CourseID, true); !ok {
		return
	}
	var in lessonInput
	if !decode(w, r, &in) {
		return
	}
	if in.Kind == "" {
		in.Kind = "page"
	}
	if msg := validateActivity(in); msg != "" {
		fail(w, 400, msg)
		return
	}
	var max int
	a.db.Model(&Lesson{}).Where("section_id = ?", s.ID).Select("COALESCE(MAX(position),0)").Scan(&max)
	v := Lesson{CourseID: s.CourseID, SectionID: s.ID, Title: in.Title, Content: in.Content, Kind: in.Kind, URL: in.URL, Duration: in.Duration, Data: in.Data, Position: max + 1}
	if a.result(w, a.db.Create(&v).Error) {
		reply(w, 201, v)
	}
}
func (a *app) editLesson(w http.ResponseWriter, r *http.Request) {
	var l Lesson
	if a.db.First(&l, id(r, "lesson")).Error != nil {
		fail(w, 404, "Элемент не найден")
		return
	}
	if _, ok := a.access(w, r, l.CourseID, true); !ok {
		return
	}
	var in lessonInput
	if !decode(w, r, &in) {
		return
	}
	if in.Kind == "" {
		in.Kind = "page"
	}
	if msg := validateActivity(in); msg != "" {
		fail(w, 400, msg)
		return
	}
	if in.SectionID == 0 {
		in.SectionID = l.SectionID
	}
	if !a.validSection(w, l.CourseID, in.SectionID) {
		return
	}

	err := a.db.Transaction(func(tx *gorm.DB) error {
		var fresh Lesson
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&fresh, l.ID).Error; err != nil {
			return err
		}
		var n int64
		if err := tx.Model(&ChoiceVote{}).Where("lesson_id = ?", fresh.ID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 && (in.Kind != "choice" || !reflect.DeepEqual(in.Data.Options, fresh.Data.Options)) {
			return fmt.Errorf("conflict:Нельзя менять варианты или тип опроса после голосования")
		}
		var files int64
		if err := tx.Model(&Resource{}).Where("lesson_id = ?", fresh.ID).Count(&files).Error; err != nil {
			return err
		}
		if files > 0 && in.Kind != "file" && in.Kind != "folder" {
			return fmt.Errorf("conflict:Сначала удалите прикреплённые файлы")
		}
		if files > 1 && in.Kind == "file" {
			return fmt.Errorf("conflict:Для нескольких файлов используйте элемент «Папка»")
		}
		if fresh.SectionID != in.SectionID {
			var max int
			if err := tx.Model(&Lesson{}).Where("section_id = ?", in.SectionID).Select("COALESCE(MAX(position),0)").Scan(&max).Error; err != nil {
				return err
			}
			fresh.Position = max + 1
		}
		fresh.Title = in.Title
		fresh.Content = in.Content
		fresh.Kind = in.Kind
		fresh.URL = in.URL
		fresh.Duration = in.Duration
		fresh.Data = in.Data
		fresh.SectionID = in.SectionID
		if err := tx.Omit("Files").Save(&fresh).Error; err != nil {
			return err
		}
		l = fresh
		return nil
	})
	if err != nil && strings.HasPrefix(err.Error(), "conflict:") {
		fail(w, 409, strings.TrimPrefix(err.Error(), "conflict:"))
		return
	}
	if a.result(w, err) {
		a.db.Preload("Files").First(&l, l.ID)
		reply(w, 200, l)
	}
}
func (a *app) deleteLesson(w http.ResponseWriter, r *http.Request) {
	var l Lesson
	if a.db.First(&l, id(r, "lesson")).Error != nil {
		fail(w, 404, "Элемент не найден")
		return
	}
	if _, ok := a.access(w, r, l.CourseID, true); !ok {
		return
	}
	files := []Resource{}
	a.db.Where("lesson_id = ?", l.ID).Find(&files)
	err := a.db.Transaction(func(tx *gorm.DB) error {
		for _, v := range []any{&Completion{}, &ChoiceVote{}, &Discussion{}, &Resource{}} {
			if err := tx.Where("lesson_id = ?", l.ID).Delete(v).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&l).Error
	})
	if !a.result(w, err) {
		return
	}
	for _, f := range files {
		removeResource(f)
	}
	reply(w, 200, map[string]bool{"ok": true})
}
func (a *app) activityLesson(w http.ResponseWriter, r *http.Request) (Lesson, bool) {
	var l Lesson
	if a.db.First(&l, id(r, "lesson")).Error != nil {
		fail(w, 404, "Элемент не найден")
		return l, false
	}
	_, ok := a.topicAccess(w, r, l.CourseID, l.SectionID, false)
	return l, ok
}
func (a *app) interaction(w http.ResponseWriter, r *http.Request) {
	l, ok := a.activityLesson(w, r)
	if !ok {
		return
	}
	switch l.Kind {
	case "forum":
		posts := []Discussion{}
		if a.result(w, a.db.Preload("User").Where("lesson_id = ?", l.ID).Order("id DESC").Limit(100).Find(&posts).Error) {
			reply(w, 200, map[string]any{"posts": posts})
		}
	case "choice":
		var vote ChoiceVote
		found := a.db.Where("lesson_id = ? AND user_id = ?", l.ID, current(r).ID).First(&vote).Error == nil
		var mine any
		if found {
			mine = vote.OptionIndex
		}
		counts := make([]int64, len(l.Data.Options))
		u := current(r)
		if found || u.Role != "student" {
			type row struct {
				OptionIndex int
				Count       int64
			}
			rows := []row{}
			a.db.Model(&ChoiceVote{}).Select("option_index, COUNT(*) AS count").Where("lesson_id = ?", l.ID).Group("option_index").Scan(&rows)
			for _, v := range rows {
				if v.OptionIndex >= 0 && v.OptionIndex < len(counts) {
					counts[v.OptionIndex] = v.Count
				}
			}
		}
		reply(w, 200, map[string]any{"vote": mine, "counts": counts})
	default:
		fail(w, 400, "У этого элемента нет интерактивной части")
	}
}
func (a *app) vote(w http.ResponseWriter, r *http.Request) {
	l, ok := a.activityLesson(w, r)
	if !ok {
		return
	}
	if l.Kind != "choice" {
		fail(w, 400, "Этот элемент не является опросом")
		return
	}
	if current(r).Role != "student" {
		fail(w, 403, "В опросе участвуют студенты")
		return
	}
	var in struct {
		OptionIndex *int `json:"option_index"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.OptionIndex == nil || *in.OptionIndex < 0 || *in.OptionIndex >= len(l.Data.Options) {
		fail(w, 400, "Выберите существующий вариант")
		return
	}

	err := a.db.Transaction(func(tx *gorm.DB) error {
		var fresh Lesson
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&fresh, l.ID).Error; err != nil {
			return err
		}
		if fresh.Kind != "choice" || *in.OptionIndex >= len(fresh.Data.Options) {
			return fmt.Errorf("conflict:Опрос изменился. Обновите страницу")
		}
		v := ChoiceVote{LessonID: fresh.ID, UserID: current(r).ID, OptionIndex: *in.OptionIndex}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "lesson_id"}, {Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"option_index"})}).Create(&v).Error
	})
	if err != nil && strings.HasPrefix(err.Error(), "conflict:") {
		fail(w, 409, strings.TrimPrefix(err.Error(), "conflict:"))
		return
	}
	if a.result(w, err) {
		a.interaction(w, r)
	}
}
func (a *app) activityPost(w http.ResponseWriter, r *http.Request) {
	l, ok := a.activityLesson(w, r)
	if !ok {
		return
	}
	if l.Kind != "forum" {
		fail(w, 400, "Этот элемент не является форумом")
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
	v := Discussion{CourseID: l.CourseID, LessonID: l.ID, UserID: current(r).ID, Body: in.Body}
	if a.result(w, a.db.Create(&v).Error) {
		a.db.Preload("User").First(&v, v.ID)
		reply(w, 201, v)
	}
}

func (a *app) moveTask(w http.ResponseWriter, r *http.Request, quiz bool) {
	var cid uint
	var model any
	var key uint
	if quiz {
		var v Quiz
		if a.db.First(&v, id(r, "quiz")).Error != nil {
			fail(w, 404, "Тест не найден")
			return
		}
		cid = v.CourseID
		key = v.ID
		model = &Quiz{}
	} else {
		var v Assignment
		if a.db.First(&v, id(r, "assignment")).Error != nil {
			fail(w, 404, "Задание не найдено")
			return
		}
		cid = v.CourseID
		key = v.ID
		model = &Assignment{}
	}
	if _, ok := a.access(w, r, cid, true); !ok {
		return
	}
	var in struct {
		SectionID uint `json:"section_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !a.validSection(w, cid, in.SectionID) {
		return
	}
	if a.result(w, a.db.Model(model).Where("id = ?", key).Update("section_id", in.SectionID).Error) {
		reply(w, 200, map[string]bool{"ok": true})
	}
}

func (a *app) visibleLessonIDs(r *http.Request, c Course) *gorm.DB {
	q := a.db.Model(&Lesson{}).Select("id").Where("course_id = ?", c.ID)
	return a.visibleActivities(r, q, c)
}
