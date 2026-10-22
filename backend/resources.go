package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxResource = 20 << 20

func uploadDir() string { return env("UPLOAD_DIR", "../var/uploads") }
func removeResource(v Resource) {
	if err := os.Remove(filepath.Join(uploadDir(), v.Key)); err != nil && !os.IsNotExist(err) {
		log.Printf("resource cleanup failed: %v", err)
	}
}
func (a *app) uploadFile(w http.ResponseWriter, r *http.Request) {
	var l Lesson
	if a.db.First(&l, id(r, "lesson")).Error != nil {
		fail(w, 404, "Элемент не найден")
		return
	}
	if _, ok := a.access(w, r, l.CourseID, true); !ok {
		return
	}
	if l.Kind != "file" && l.Kind != "folder" {
		fail(w, 400, "Файлы добавляются только в элементы «Файл» и «Папка»")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxResource+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		fail(w, 400, "Не удалось прочитать файл. Максимум 20 МБ")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	f, header, err := r.FormFile("file")
	if err != nil {
		fail(w, 400, "Выберите файл")
		return
	}
	defer f.Close()
	name := filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
	if len(name) > 240 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		fail(w, 400, "Некорректное имя файла")
		return
	}
	ext := strings.ToLower(filepath.Ext(name))
	allowed := map[string]bool{".pdf": true, ".txt": true, ".md": true, ".csv": true, ".docx": true, ".xlsx": true, ".pptx": true, ".odt": true, ".ods": true, ".odp": true, ".zip": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".mp3": true, ".mp4": true, ".webm": true}
	if !allowed[ext] {
		fail(w, 400, "Формат файла не разрешён. Используйте документ, архив, изображение, аудио или видео")
		return
	}
	if err := os.MkdirAll(uploadDir(), 0700); err != nil {
		a.result(w, err)
		return
	}
	key := make([]byte, 24)
	if _, err = rand.Read(key); err != nil {
		a.result(w, err)
		return
	}
	storage := hex.EncodeToString(key)
	dest, err := os.OpenFile(filepath.Join(uploadDir(), storage), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		a.result(w, err)
		return
	}
	n, err := io.Copy(dest, io.LimitReader(f, maxResource+1))
	closeErr := dest.Close()
	if err != nil || closeErr != nil || n > maxResource || n == 0 {
		os.Remove(filepath.Join(uploadDir(), storage))
		fail(w, 400, "Файл пустой, повреждён или превышает 20 МБ")
		return
	}
	v := Resource{LessonID: l.ID, CourseID: l.CourseID, Name: name, Key: storage, Size: n, ContentType: mime.TypeByExtension(ext)}
	if v.ContentType == "" {
		v.ContentType = "application/octet-stream"
	}
	err = a.db.Transaction(func(tx *gorm.DB) error {
		var lock Lesson
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lock, l.ID).Error; err != nil {
			return err
		}
		var count int64
		tx.Model(&Resource{}).Where("lesson_id = ?", l.ID).Count(&count)
		if (lock.Kind != "file" && lock.Kind != "folder") || (lock.Kind == "file" && count >= 1) || count >= 20 {
			return fmt.Errorf("file_limit")
		}
		return tx.Create(&v).Error
	})
	if err != nil {
		removeResource(v)
		if err.Error() == "file_limit" {
			fail(w, 409, "Файл уже добавлен или в папке достигнут лимит 20 файлов")
		} else {
			a.result(w, err)
		}
		return
	}
	reply(w, 201, v)
}
func (a *app) downloadFile(w http.ResponseWriter, r *http.Request) {
	var v Resource
	if a.db.First(&v, id(r, "resource")).Error != nil {
		fail(w, 404, "Файл не найден")
		return
	}
	var l Lesson
	if a.db.First(&l, v.LessonID).Error != nil {
		fail(w, 404, "Элемент не найден")
		return
	}
	if _, ok := a.topicAccess(w, r, l.CourseID, l.SectionID, false); !ok {
		return
	}
	f, err := os.Open(filepath.Join(uploadDir(), v.Key))
	if err != nil {
		fail(w, 404, "Файл отсутствует в хранилище")
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		fail(w, 500, "Не удалось прочитать файл")
		return
	}
	w.Header().Set("Content-Type", v.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": v.Name}))
	http.ServeContent(w, r, v.Name, stat.ModTime(), f)
}
func (a *app) deleteFile(w http.ResponseWriter, r *http.Request) {
	var v Resource
	if a.db.First(&v, id(r, "resource")).Error != nil {
		fail(w, 404, "Файл не найден")
		return
	}
	if _, ok := a.access(w, r, v.CourseID, true); !ok {
		return
	}
	if a.result(w, a.db.Delete(&v).Error) {
		removeResource(v)
		reply(w, 200, map[string]bool{"ok": true})
	}
}
