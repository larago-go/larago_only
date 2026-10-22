package main

import "time"

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name"`
	Email     string    `json:"email" gorm:"uniqueIndex"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
type Session struct {
	ID        uint   `gorm:"primaryKey"`
	Token     string `gorm:"uniqueIndex"`
	UserID    uint   `gorm:"index"`
	ExpiresAt time.Time
}
type Course struct {
	ShortName             string     `json:"short_name"`
	Format                string     `json:"format" gorm:"default:topics"`
	Language              string     `json:"language" gorm:"default:ru"`
	StartAt               *time.Time `json:"start_at"`
	EndAt                 *time.Time `json:"end_at"`
	DisableSelfEnrollment bool       `json:"disable_self_enrollment" gorm:"default:false"`
	ID                    uint       `json:"id" gorm:"primaryKey"`
	Title                 string     `json:"title"`
	Description           string     `json:"description"`
	Category              string     `json:"category"`
	Color                 string     `json:"color"`
	Published             bool       `json:"published"`
	TeacherID             uint       `json:"teacher_id" gorm:"index"`
	Teacher               User       `json:"teacher" gorm:"foreignKey:TeacherID"`
	CreatedAt             time.Time  `json:"created_at"`
}
type Enrollment struct {
	Source    string    `json:"source" gorm:"default:manual"`
	ID        uint      `json:"id" gorm:"primaryKey"`
	CourseID  uint      `json:"course_id" gorm:"uniqueIndex:enrollment"`
	UserID    uint      `json:"user_id" gorm:"uniqueIndex:enrollment"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
}
type Section struct {
	Summary  string   `json:"summary"`
	Hidden   bool     `json:"hidden" gorm:"default:false"`
	ID       uint     `json:"id" gorm:"primaryKey"`
	CourseID uint     `json:"course_id" gorm:"index"`
	Title    string   `json:"title"`
	Position int      `json:"position"`
	Lessons  []Lesson `json:"lessons"`
}
type Lesson struct {
	Data      ActivityData `json:"data" gorm:"serializer:json"`
	Files     []Resource   `json:"files" gorm:"foreignKey:LessonID"`
	Kind      string       `json:"kind"`
	URL       string       `json:"url"`
	ID        uint         `json:"id" gorm:"primaryKey"`
	SectionID uint         `json:"section_id" gorm:"index;default:0"`
	CourseID  uint         `json:"course_id" gorm:"index"`
	Title     string       `json:"title"`
	Content   string       `json:"content"`
	Duration  int          `json:"duration"`
	Position  int          `json:"position"`
}
type Completion struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"uniqueIndex:completion"`
	LessonID  uint `gorm:"uniqueIndex:completion"`
	CourseID  uint `gorm:"index"`
	CreatedAt time.Time
}
type Assignment struct {
	SectionID   uint      `json:"section_id" gorm:"index;default:0"`
	ID          uint      `json:"id" gorm:"primaryKey"`
	CourseID    uint      `json:"course_id" gorm:"index"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueAt       time.Time `json:"due_at"`
	MaxScore    int       `json:"max_score"`
}
type Submission struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	AssignmentID uint      `json:"assignment_id" gorm:"uniqueIndex:submission"`
	UserID       uint      `json:"user_id" gorm:"uniqueIndex:submission"`
	User         User      `json:"user"`
	Content      string    `json:"content"`
	Score        *int      `json:"score"`
	Feedback     string    `json:"feedback"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
type Quiz struct {
	SectionID uint       `json:"section_id" gorm:"index;default:0"`
	ID        uint       `json:"id" gorm:"primaryKey"`
	CourseID  uint       `json:"course_id" gorm:"index"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
}
type Question struct {
	ID           uint     `json:"id" gorm:"primaryKey"`
	QuizID       uint     `json:"quiz_id" gorm:"index"`
	Prompt       string   `json:"prompt"`
	Options      []string `json:"options" gorm:"serializer:json"`
	CorrectIndex int      `json:"-"`
}
type Attempt struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	QuizID    uint      `json:"quiz_id" gorm:"index"`
	UserID    uint      `json:"user_id" gorm:"index"`
	Score     int       `json:"score"`
	Total     int       `json:"total"`
	Answers   []int     `json:"-" gorm:"serializer:json"`
	CreatedAt time.Time `json:"created_at"`
}
type Announcement struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CourseID  uint      `json:"course_id" gorm:"index"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
type Discussion struct {
	LessonID  uint      `json:"lesson_id" gorm:"index;default:0"`
	ID        uint      `json:"id" gorm:"primaryKey"`
	CourseID  uint      `json:"course_id" gorm:"index"`
	UserID    uint      `json:"user_id"`
	User      User      `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// Cohorts are site-wide groups. CourseCohort synchronises their membership.
type Cohort struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
type CohortMember struct {
	ID       uint `json:"id" gorm:"primaryKey"`
	CohortID uint `json:"cohort_id" gorm:"uniqueIndex:cohort_member"`
	UserID   uint `json:"user_id" gorm:"uniqueIndex:cohort_member"`
	User     User `json:"user"`
}
type CourseCohort struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	CourseID uint   `json:"course_id" gorm:"uniqueIndex:course_cohort"`
	CohortID uint   `json:"cohort_id" gorm:"uniqueIndex:course_cohort"`
	Cohort   Cohort `json:"cohort"`
}
type BankQuestion struct {
	Prompt       string   `json:"prompt"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correct_index"`
}
type BankItem struct {
	Data      ActivityData   `json:"data" gorm:"serializer:json"`
	ID        uint           `json:"id" gorm:"primaryKey"`
	AuthorID  uint           `json:"author_id"`
	Author    User           `json:"author" gorm:"foreignKey:AuthorID"`
	Title     string         `json:"title"`
	Category  string         `json:"category"`
	Kind      string         `json:"kind"`
	Body      string         `json:"body"`
	URL       string         `json:"url"`
	Duration  int            `json:"duration"`
	MaxScore  int            `json:"max_score"`
	Questions []BankQuestion `json:"questions" gorm:"serializer:json"`
	CreatedAt time.Time      `json:"created_at"`
}

// Structured data for interactive course resources; no arbitrary HTML is executed.
type Chapter struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
type GlossaryEntry struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
}
type ActivityData struct {
	Chapters []Chapter       `json:"chapters"`
	Entries  []GlossaryEntry `json:"entries"`
	Options  []string        `json:"options"`
}
type ChoiceVote struct {
	ID          uint `json:"id" gorm:"primaryKey"`
	LessonID    uint `json:"lesson_id" gorm:"uniqueIndex:choice_vote"`
	UserID      uint `json:"-" gorm:"uniqueIndex:choice_vote"`
	OptionIndex int  `json:"option_index"`
}
type Resource struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	LessonID    uint      `json:"lesson_id" gorm:"index;default:0"`
	CourseID    uint      `json:"-" gorm:"index"`
	Name        string    `json:"name"`
	Key         string    `json:"-"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}
