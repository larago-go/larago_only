package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func backupFixture() map[string]string {
	return map[string]string{
		"moodle_backup.xml":              `<?xml version="1.0"?><moodle_backup><information><moodle_version>2024100700</moodle_version></information></moodle_backup>`,
		"course/course.xml":              `<course id="25"><fullname>Тестовый курс Moodle</fullname></course>`,
		"sections/section_7/section.xml": `<section id="7"><number>1</number><name>Основы</name></section>`,
		"activities/page_1/page.xml":     `<activity id="1" moduleid="10" modulename="page"><page id="1"><name>Первая страница</name><content>&lt;p&gt;Содержимое урока&lt;/p&gt;&lt;script&gt;alert(1)&lt;/script&gt;</content></page></activity>`,
		"activities/page_1/module.xml":   `<module id="10"><sectionid>7</sectionid></module>`,
		"activities/assign_2/assign.xml": `<activity><assign id="2"><name>Практическое задание</name><intro>&lt;p&gt;Опишите решение&lt;/p&gt;</intro><grade>50</grade><duedate>1900000000</duedate></assign></activity>`,
		"activities/assign_2/module.xml": `<module><sectionid>7</sectionid></module>`,
		"questions.xml":                  `<question_categories><question_category id="1"><question_bank_entries><question_bank_entry id="20"><question_version id="30"><version>1</version><status>ready</status><question id="42"><name>Основы</name><qtype>multichoice</qtype><questiontext>&lt;p&gt;Сколько будет два плюс два?&lt;/p&gt;</questiontext><answers><answer id="1"><answertext>Три</answertext><fraction>0</fraction></answer><answer id="2"><answertext>Четыре</answertext><fraction>100</fraction></answer></answers><plugin_qtype_multichoice_question><multichoice><single>1</single></multichoice></plugin_qtype_multichoice_question></question></question_version></question_bank_entry></question_bank_entries></question_category></question_categories>`,
		"activities/quiz_3/quiz.xml":     `<activity><quiz id="3"><name>Контрольный тест</name><question_slots><question_slot id="5"><slot>1</slot><question_reference><questionbankentryid>20</questionbankentryid></question_reference></question_slot></question_slots></quiz></activity>`,
		"activities/quiz_3/module.xml":   `<module><sectionid>7</sectionid></module>`,
		"activities/url_4/url.xml":       `<activity><url><name>Документация Go</name><externalurl>https://go.dev/doc/</externalurl><intro>Официальная документация</intro></url></activity>`,
		"activities/forum_5/forum.xml":   `<activity><forum><name>Форум курса</name></forum></activity>`,
	}
}
func tarBackup(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	for name, data := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestMoodleRestorePlan(t *testing.T) {
	files, _, err := readMoodleArchive(tarBackup(t, backupFixture()))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := parseMoodle(files)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Source != "Тестовый курс Moodle" || len(plan.Activities) != 4 || len(plan.QuestionBank) != 1 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	for _, v := range plan.Activities {
		if v.Kind == "page" && strings.Contains(v.Body, "alert") {
			t.Fatal("HTML script survived conversion")
		}
		if v.Kind == "quiz" && (v.Questions[0].CorrectIndex != 1 || v.Questions[0].Options[1] != "Четыре") {
			t.Fatal("wrong answer key")
		}
	}
	if !strings.Contains(strings.Join(plan.Warnings, " "), "forum") {
		t.Fatal("unsupported activity was not reported")
	}
}
func TestMoodleLegacyQuestions(t *testing.T) {
	f := backupFixture()
	f["questions.xml"] = `<question_categories><question_category><questions><question id="42"><qtype>truefalse</qtype><questiontext>Go — язык программирования?</questiontext><answers><answer><answertext>Верно</answertext><fraction>100</fraction></answer><answer><answertext>Неверно</answertext><fraction>0</fraction></answer></answers></question></questions></question_category></question_categories>`
	f["activities/quiz_3/quiz.xml"] = `<activity><quiz><name>Старый формат</name><question_instances><question_instance><questionid>42</questionid></question_instance></question_instances></quiz></activity>`
	files, _, err := readMoodleArchive(tarBackup(t, f))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := parseMoodle(files)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range plan.Activities {
		if a.Kind == "quiz" {
			found = true
			if a.Questions[0].CorrectIndex != 0 {
				t.Fatal("wrong truefalse key")
			}
		}
	}
	if !found {
		t.Fatal("legacy quiz not restored")
	}
}
func TestMoodleUnsupportedQuestionDoesNotSilentlyChangeQuiz(t *testing.T) {
	f := backupFixture()
	f["questions.xml"] = strings.ReplaceAll(f["questions.xml"], "multichoice", "essay")
	files, _, _ := readMoodleArchive(tarBackup(t, f))
	plan, err := parseMoodle(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range plan.Activities {
		if a.Kind == "quiz" {
			t.Fatal("partial quiz must not be restored")
		}
	}
	if !strings.Contains(strings.Join(plan.Warnings, " "), "пропущен целиком") {
		t.Fatal("missing compatibility warning")
	}
}
func TestMoodleRejectsUnsafeAndInvalidArchives(t *testing.T) {
	for _, name := range []string{"../escape.xml", "/absolute.xml", `bad\file.xml`} {
		t.Run(name, func(t *testing.T) {
			f := backupFixture()
			f[name] = "x"
			if _, _, err := readMoodleArchive(tarBackup(t, f)); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
	if _, _, err := readMoodleArchive([]byte("not a backup")); err == nil {
		t.Fatal("invalid archive accepted")
	}
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	tw.WriteHeader(&tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"})
	tw.Close()
	if _, _, err := readMoodleArchive(b.Bytes()); err == nil {
		t.Fatal("symlink accepted")
	}
	b.Reset()
	tw = tar.NewWriter(&b)
	tw.WriteHeader(&tar.Header{Name: "large.xml", Mode: 0600, Size: maxExpanded + 1})
	if _, _, err := readMoodleArchive(b.Bytes()); err == nil {
		t.Fatal("oversized entry accepted")
	}
}
func TestMoodleZipAndAnswerKeyRedaction(t *testing.T) {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for name, data := range backupFixture() {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprint(w, data)
	}
	z.Close()
	files, _, err := readMoodleArchive(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = parseMoodle(files); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(Question{Prompt: "Test", Options: []string{"A", "B"}, CorrectIndex: 1})
	if bytes.Contains(raw, []byte("correct")) {
		t.Fatal("answer key leaked into student payload")
	}
}

func TestMoodleNeverFallsBackToOldSupportedQuestionVersion(t *testing.T) {
	f := backupFixture()
	newer := `<question_version id="31"><version>2</version><status>ready</status><question id="43"><qtype>essay</qtype><questiontext>Напишите эссе</questiontext></question></question_version>`
	f["questions.xml"] = strings.Replace(f["questions.xml"], "</question_bank_entry>", newer+"</question_bank_entry>", 1)
	files, _, err := readMoodleArchive(tarBackup(t, f))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := parseMoodle(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range plan.Activities {
		if a.Kind == "quiz" {
			t.Fatal("unsupported current question silently replaced by older version")
		}
	}
}

func TestMoodlePinnedQuestionVersion(t *testing.T) {
	f := backupFixture()
	newer := `<question_version id="31"><version>2</version><status>ready</status><question id="43"><qtype>truefalse</qtype><questiontext>Новая версия вопроса?</questiontext><answers><answer><answertext>Да</answertext><fraction>100</fraction></answer><answer><answertext>Нет</answertext><fraction>0</fraction></answer></answers></question></question_version>`
	f["questions.xml"] = strings.Replace(f["questions.xml"], "</question_bank_entry>", newer+"</question_bank_entry>", 1)
	f["activities/quiz_3/quiz.xml"] = strings.Replace(f["activities/quiz_3/quiz.xml"], "</questionbankentryid>", "</questionbankentryid><version>1</version>", 1)
	files, _, err := readMoodleArchive(tarBackup(t, f))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := parseMoodle(files)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range plan.Activities {
		if a.Kind == "quiz" {
			found = true
			if a.Questions[0].CorrectIndex != 1 || a.Questions[0].Prompt != "Сколько будет два плюс два?" {
				t.Fatal("pinned question version was not respected")
			}
		}
	}
	if !found {
		t.Fatal("pinned quiz was omitted")
	}
}
