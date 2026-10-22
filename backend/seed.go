package main

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"time"
)

// Only seeds an empty database when DEMO_SEED=true. Never resets existing data.
func seed(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&User{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		password, err := bcrypt.GenerateFromPassword([]byte("Learn2026!"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		users := []User{{Name: "Александра Морозова", Email: "student@campus.local", Role: "student", Password: string(password)}, {Name: "Анна Смирнова", Email: "teacher@campus.local", Role: "teacher", Password: string(password)}, {Name: "Администратор", Email: "admin@campus.local", Role: "admin", Password: string(password)}}
		for i := range users {
			if err := tx.Create(&users[i]).Error; err != nil {
				return err
			}
		}
		specs := []struct {
			title, description, category, color string
			modules                             []string
		}{
			{"Проектирование цифровых продуктов", "От исследования пользователей до интерактивного прототипа. Научитесь создавать удобные интерфейсы и обосновывать дизайн-решения.", "Дизайн", "violet", []string{"Знакомство с продуктовым дизайном", "Исследования и пользовательский опыт", "Визуальная система и прототип"}},
			{"Go: от основ к веб-приложениям", "Создайте надёжный API на Go: типы, обработка ошибок, HTTP, работа с PostgreSQL и архитектура приложения.", "Разработка", "blue", []string{"Первые шаги в Go", "HTTP и работа с данными", "Проектирование API"}},
			{"Аналитика данных на практике", "Превращайте данные в решения: формулируйте гипотезы, исследуйте метрики и создавайте понятные отчёты.", "Аналитика", "green", []string{"Мышление аналитика", "Метрики и эксперименты", "История в данных"}},
			{"Управление командой и проектами", "Планируйте работу, выстраивайте коммуникацию и помогайте команде достигать общих целей.", "Менеджмент", "orange", []string{"Команда и общая цель", "Планирование и приоритеты", "Обратная связь"}},
		}
		for i, spec := range specs {
			c := Course{Title: spec.title, Description: spec.description, Category: spec.category, Color: spec.color, Published: true, TeacherID: users[1].ID}
			if err := tx.Create(&c).Error; err != nil {
				return err
			}
			if i < 3 {
				if err := tx.Create(&Enrollment{CourseID: c.ID, UserID: users[0].ID}).Error; err != nil {
					return err
				}
			}
			for j, title := range spec.modules {
				s := Section{CourseID: c.ID, Title: title, Position: j + 1}
				if err := tx.Create(&s).Error; err != nil {
					return err
				}
				for k, t := range []string{"Введение и цели модуля", "Ключевые принципы и примеры", "Практика: применяем знания"} {
					l := Lesson{CourseID: c.ID, SectionID: s.ID, Title: t, Duration: 10 + k*5, Position: k + 1, Content: "Добро пожаловать в модуль «" + title + "».\n\nЦель урока\nРазобраться в основных понятиях и научиться применять их на собственном проекте. Прежде чем приступить к практике, сформулируйте, какую проблему вы хотите решить и для кого.\n\nОт теории к практике\nНачните с небольшого примера. Запишите исходные предположения, выберите способ проверки и сравните результат с ожиданиями. Хорошее решение опирается на наблюдения, а не только на интуицию.\n\nЗадание для самостоятельной работы\n1. Выберите знакомый вам продукт или рабочий процесс.\n2. Опишите одну проблему и критерий успешного решения.\n3. Предложите два варианта улучшения и объясните выбор.\n\nСохраните заметки: они пригодятся для итогового задания. Если возникнут вопросы, обсудите их с преподавателем на вкладке «Обсуждение»."}
					if err := tx.Create(&l).Error; err != nil {
						return err
					}
					if i == 0 && j == 0 {
						if err := tx.Create(&Completion{CourseID: c.ID, LessonID: l.ID, UserID: users[0].ID}).Error; err != nil {
							return err
						}
					}
				}
			}
			as := Assignment{CourseID: c.ID, Title: "Практическая работа: от идеи к решению", Description: "Выберите задачу по теме курса. Опишите проблему, аудиторию, предложенное решение и способ проверки результата. Добавьте текст работы или ссылку на документ с открытым доступом.", MaxScore: 100, DueAt: time.Now().AddDate(0, 0, 5+i)}
			if err := tx.Create(&as).Error; err != nil {
				return err
			}
			q := Quiz{CourseID: c.ID, Title: "Проверка знаний: основы", Questions: []Question{{Prompt: "С чего стоит начинать работу над новой задачей?", Options: []string{"С выбора инструментов", "С определения проблемы и цели", "С оформления презентации"}, CorrectIndex: 1}, {Prompt: "Как проверить, что решение работает?", Options: []string{"Определить критерий успеха и провести проверку", "Опираться только на собственное мнение", "Повторить решение конкурентов"}, CorrectIndex: 0}, {Prompt: "Что помогает закрепить знания?", Options: []string{"Только чтение", "Практика и обратная связь", "Заучивание терминов без примеров"}, CorrectIndex: 1}}}
			if err := tx.Create(&q).Error; err != nil {
				return err
			}
			if err := tx.Create(&Announcement{CourseID: c.ID, Title: "Рады видеть вас на курсе!", Body: "Материалы первого модуля уже доступны. Проходите уроки в своём темпе, а вопросы задавайте в обсуждении. На этой неделе познакомимся с основными понятиями и начнём практическую работу."}).Error; err != nil {
				return err
			}
			if err := tx.Create(&Discussion{CourseID: c.ID, UserID: users[1].ID, Body: "Давайте познакомимся! Расскажите, с каким опытом вы пришли и чему хотите научиться."}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
