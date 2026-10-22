package main

import "gorm.io/gorm"

func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&User{}, &Session{}, &Course{}, &Enrollment{}, &Section{}, &Lesson{}, &Completion{}, &Assignment{}, &Submission{}, &Quiz{}, &Question{}, &Attempt{}, &Announcement{}, &Discussion{}, &Cohort{}, &CohortMember{}, &CourseCohort{}, &BankItem{}, &ChoiceVote{}, &Resource{}); err != nil {
		return err
	}
	return backfillTopicDefaults(db)
}

// Older courses had no topic visibility or task placement columns. Preserve
// their availability when upgrading, including partially applied migrations.
func backfillTopicDefaults(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, query := range []string{
			"UPDATE sections SET hidden = false WHERE hidden IS NULL",
			"UPDATE assignments SET section_id = 0 WHERE section_id IS NULL",
			"UPDATE quizzes SET section_id = 0 WHERE section_id IS NULL",
			"UPDATE discussions SET lesson_id = 0 WHERE lesson_id IS NULL",
			"UPDATE courses SET disable_self_enrollment = false WHERE disable_self_enrollment IS NULL",
		} {
			if err := tx.Exec(query).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
