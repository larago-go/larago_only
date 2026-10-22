package main

import (
	"net/http"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func staff(w http.ResponseWriter, r *http.Request) bool {
	if current(r).Role == "student" {
		fail(w, 403, "Доступно преподавателям и администраторам")
		return false
	}
	return true
}
func admin(w http.ResponseWriter, r *http.Request) bool {
	if current(r).Role != "admin" {
		fail(w, 403, "Доступно только администратору")
		return false
	}
	return true
}
func (a *app) extendedRoutes(m *http.ServeMux) {
	a.activityRoutes(m)
	for p, h := range map[string]http.HandlerFunc{
		"GET /api/cohorts": a.cohorts, "POST /api/cohorts": a.createCohort, "POST /api/cohorts/{cohort}/members": a.addMember, "DELETE /api/cohorts/{cohort}/members/{user}": a.removeMember,
		"GET /api/courses/{course}/cohorts": a.courseCohorts, "POST /api/courses/{course}/cohorts": a.linkCohort, "DELETE /api/courses/{course}/cohorts/{cohort}": a.unlinkCohort,
		"GET /api/bank": a.bank, "POST /api/bank": a.createBank, "DELETE /api/bank/{item}": a.deleteBank, "POST /api/courses/{course}/bank/{item}": a.useBank,
		"POST /api/courses/{course}/import/moodle": a.importMoodle,
	} {
		m.HandleFunc(p, a.auth(h))
	}
}
func (a *app) cohorts(w http.ResponseWriter, r *http.Request) {
	if !staff(w, r) {
		return
	}
	out := []Cohort{}
	if !a.result(w, a.db.Order("id").Find(&out).Error) {
		return
	}
	result := []map[string]any{}
	for _, c := range out {
		var n int64
		a.db.Model(&CohortMember{}).Where("cohort_id = ?", c.ID).Count(&n)
		row := map[string]any{"id": c.ID, "name": c.Name, "description": c.Description, "count": n}
		if current(r).Role == "admin" {
			members := []CohortMember{}
			a.db.Preload("User").Where("cohort_id = ?", c.ID).Order("id").Find(&members)
			row["members"] = members
		}
		result = append(result, row)
	}
	reply(w, 200, result)
}
func (a *app) createCohort(w http.ResponseWriter, r *http.Request) {
	if !admin(w, r) {
		return
	}
	var in struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Name)) < 2 || len(in.Name) > 200 || len(in.Description) > 5000 {
		fail(w, 400, "Проверьте название и описание группы")
		return
	}
	v := Cohort{Name: in.Name, Description: in.Description}
	if a.result(w, a.db.Create(&v).Error) {
		reply(w, 201, v)
	}
}

// All synchronisations for a course share an advisory transaction lock.
func syncCohortEnrollments(tx *gorm.DB, courseID uint) error {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(?, ?)", 72514, int64(courseID)).Error; err != nil {
		return err
	}
	ids := []uint{}
	if err := tx.Table("cohort_members").Select("DISTINCT cohort_members.user_id").Joins("JOIN course_cohorts ON course_cohorts.cohort_id = cohort_members.cohort_id").Where("course_cohorts.course_id = ?", courseID).Pluck("cohort_members.user_id", &ids).Error; err != nil {
		return err
	}
	for _, uid := range ids {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&Enrollment{CourseID: courseID, UserID: uid, Source: "cohort"}).Error; err != nil {
			return err
		}
	}
	q := tx.Where("course_id = ? AND source = ?", courseID, "cohort")
	if len(ids) > 0 {
		q = q.Where("user_id NOT IN ?", ids)
	}
	return q.Delete(&Enrollment{}).Error
}
func syncMemberCourses(tx *gorm.DB, cohortID uint) error {
	links := []CourseCohort{}
	if err := tx.Where("cohort_id = ?", cohortID).Order("course_id").Find(&links).Error; err != nil {
		return err
	}
	for _, l := range links {
		if err := syncCohortEnrollments(tx, l.CourseID); err != nil {
			return err
		}
	}
	return nil
}
func (a *app) addMember(w http.ResponseWriter, r *http.Request) {
	if !admin(w, r) {
		return
	}
	var in struct {
		UserID uint `json:"user_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	var c Cohort
	var u User
	if a.db.First(&c, id(r, "cohort")).Error != nil || a.db.First(&u, in.UserID).Error != nil {
		fail(w, 404, "Группа или пользователь не найдены")
		return
	}
	err := a.db.Transaction(func(tx *gorm.DB) error {
		v := CohortMember{CohortID: c.ID, UserID: u.ID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&v).Error; err != nil {
			return err
		}
		return syncMemberCourses(tx, c.ID)
	})
	if a.result(w, err) {
		reply(w, 200, map[string]bool{"ok": true})
	}
}
func (a *app) removeMember(w http.ResponseWriter, r *http.Request) {
	if !admin(w, r) {
		return
	}
	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("cohort_id = ? AND user_id = ?", id(r, "cohort"), id(r, "user")).Delete(&CohortMember{}).Error; err != nil {
			return err
		}
		return syncMemberCourses(tx, id(r, "cohort"))
	})
	if a.result(w, err) {
		reply(w, 200, map[string]bool{"ok": true})
	}
}
func (a *app) courseCohorts(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	out := []CourseCohort{}
	if a.result(w, a.db.Preload("Cohort").Where("course_id = ?", c.ID).Find(&out).Error) {
		reply(w, 200, out)
	}
}
func (a *app) linkCohort(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	var in struct {
		CohortID uint `json:"cohort_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	var cohort Cohort
	if a.db.First(&cohort, in.CohortID).Error != nil {
		fail(w, 404, "Группа не найдена")
		return
	}
	err := a.db.Transaction(func(tx *gorm.DB) error {
		v := CourseCohort{CourseID: c.ID, CohortID: cohort.ID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&v).Error; err != nil {
			return err
		}
		return syncCohortEnrollments(tx, c.ID)
	})
	if a.result(w, err) {
		reply(w, 200, map[string]bool{"ok": true})
	}
}
func (a *app) unlinkCohort(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("course_id = ? AND cohort_id = ?", c.ID, id(r, "cohort")).Delete(&CourseCohort{}).Error; err != nil {
			return err
		}
		return syncCohortEnrollments(tx, c.ID)
	})
	if a.result(w, err) {
		reply(w, 200, map[string]bool{"ok": true})
	}
}
