package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const maxBackup = 32 << 20
const maxExpanded = 64 << 20
const maxXML = 8 << 20

type xmlNode struct {
	Name     string
	Attrs    map[string]string
	Text     string
	Children []*xmlNode
}

func parseXML(data []byte) (*xmlNode, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	root := &xmlNode{}
	stack := []*xmlNode{root}
	tokens := 0
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		tokens++
		if tokens > 500000 {
			return nil, fmt.Errorf("слишком сложный XML")
		}
		switch v := t.(type) {
		case xml.StartElement:
			if len(stack) > 100 {
				return nil, fmt.Errorf("слишком глубокий XML")
			}
			n := &xmlNode{Name: v.Name.Local, Attrs: map[string]string{}}
			for _, a := range v.Attr {
				n.Attrs[a.Name.Local] = a.Value
			}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, n)
			stack = append(stack, n)
		case xml.CharData:
			stack[len(stack)-1].Text += string(v)
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if len(root.Children) != 1 {
		return nil, fmt.Errorf("неверный XML")
	}
	return root.Children[0], nil
}
func (n *xmlNode) child(name string) *xmlNode {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}
func (n *xmlNode) val(name string) string {
	if c := n.child(name); c != nil {
		return strings.TrimSpace(c.Text)
	}
	return ""
}
func (n *xmlNode) all(name string) []*xmlNode {
	out := []*xmlNode{}
	if n == nil {
		return out
	}
	if n.Name == name {
		out = append(out, n)
	}
	for _, c := range n.Children {
		out = append(out, c.all(name)...)
	}
	return out
}

var stripTags = regexp.MustCompile(`<[^>]*>`)
var scriptTags = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(script|style)>`)
var breaks = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</div>|</li>|</h[1-6]>`)

func plain(s string) string {
	return strings.TrimSpace(html.UnescapeString(stripTags.ReplaceAllString(breaks.ReplaceAllString(scriptTags.ReplaceAllString(s, ""), "\n"), "")))
}
func safeArchiveName(s string) bool {
	if s == "" || strings.Contains(s, "\\") || strings.HasPrefix(s, "/") {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == ".." {
			return false
		}
	}
	return !strings.Contains(s, ":")
}
func wantedXML(name string) bool {
	return name == "moodle_backup.xml" || name == "questions.xml" || name == "course/course.xml" || (strings.HasPrefix(name, "sections/") && path.Base(name) == "section.xml") || (strings.HasPrefix(name, "activities/") && strings.HasSuffix(name, ".xml") && len(strings.Split(name, "/")) == 3)
}

// Archive entries are read in memory, never extracted. Enforce actual expanded size as well as declared sizes.
func readMoodleArchive(data []byte) (map[string][]byte, int, error) {
	files := map[string][]byte{}
	total := int64(0)
	entries := 0
	skipped := 0
	add := func(name string, size int64, r io.Reader) error {
		entries++
		if entries > 5000 {
			return fmt.Errorf("в архиве больше 5000 элементов")
		}
		if !safeArchiveName(name) {
			return fmt.Errorf("небезопасный путь в архиве")
		}
		name = strings.TrimPrefix(path.Clean(name), "./")
		if size < 0 || size > maxExpanded || total+size > maxExpanded {
			return fmt.Errorf("распакованный архив превышает 64 МБ")
		}
		var buf bytes.Buffer
		var w io.Writer = io.Discard
		wanted := wantedXML(name)
		if wanted {
			if size > maxXML {
				return fmt.Errorf("XML-файл превышает 8 МБ")
			}
			w = &buf
		}
		n, err := io.Copy(w, io.LimitReader(r, maxExpanded-total+1))
		total += n
		if err != nil {
			return err
		}
		if total > maxExpanded {
			return fmt.Errorf("распакованный архив превышает 64 МБ")
		}
		if wanted {
			if n > maxXML {
				return fmt.Errorf("XML-файл превышает 8 МБ")
			}
			if _, exists := files[name]; exists {
				return fmt.Errorf("повторяющийся XML-файл")
			}
			files[name] = buf.Bytes()
		} else {
			skipped++
		}
		return nil
	}
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, 0, fmt.Errorf("не удалось открыть ZIP: %w", err)
		}
		for _, f := range z.File {
			if !safeArchiveName(f.Name) || f.Mode()&0x8000000 != 0 {
				return nil, 0, fmt.Errorf("небезопасный элемент ZIP")
			}
			if f.FileInfo().IsDir() {
				continue
			}
			if f.UncompressedSize64 > maxExpanded {
				return nil, 0, fmt.Errorf("слишком большой элемент ZIP")
			}
			r, err := f.Open()
			if err != nil {
				return nil, 0, err
			}
			err = add(f.Name, int64(f.UncompressedSize64), r)
			r.Close()
			if err != nil {
				return nil, 0, err
			}
		}
	} else {
		var reader io.Reader = bytes.NewReader(data)
		if bytes.HasPrefix(data, []byte{0x1f, 0x8b}) {
			g, err := gzip.NewReader(reader)
			if err != nil {
				return nil, 0, err
			}
			defer g.Close()
			reader = g
		}
		tr := tar.NewReader(reader)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, 0, fmt.Errorf("не удалось открыть TAR/GZIP: %w", err)
			}
			if !safeArchiveName(h.Name) {
				return nil, 0, fmt.Errorf("небезопасный путь в TAR")
			}
			if h.Typeflag == tar.TypeDir {
				continue
			}
			if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA {
				return nil, 0, fmt.Errorf("ссылки и специальные файлы не поддерживаются")
			}
			if err = add(h.Name, h.Size, tr); err != nil {
				return nil, 0, err
			}
		}
	}
	if _, ok := files["moodle_backup.xml"]; !ok {
		return nil, 0, fmt.Errorf("moodle_backup.xml не найден: нужен незашифрованный бэкап курса Moodle")
	}
	return files, skipped, nil
}

type importedActivity struct {
	Kind, Title, Body, URL, Section string
	MaxScore                        int
	DueAt                           time.Time
	Questions                       []BankQuestion
}
type importPlan struct {
	Sections     []string
	Activities   []importedActivity
	Warnings     []string
	QuestionBank []BankQuestion
	Source       string
}

func backupQuestion(n *xmlNode) (BankQuestion, bool) {
	kind := n.val("qtype")
	if kind != "multichoice" && kind != "truefalse" {
		return BankQuestion{}, false
	}
	if kind == "multichoice" {
		for _, single := range n.all("single") {
			if strings.TrimSpace(single.Text) == "0" {
				return BankQuestion{}, false
			}
		}
	}
	q := BankQuestion{Prompt: plain(n.val("questiontext")), CorrectIndex: -1}
	if q.Prompt == "" {
		q.Prompt = plain(n.val("name"))
	}
	correct := 0
	for _, answer := range n.all("answer") {
		text := plain(answer.val("answertext"))
		if text == "" {
			text = plain(answer.val("text"))
		}
		if text == "" {
			continue
		}
		fraction, _ := strconv.ParseFloat(answer.val("fraction"), 64)
		if fraction >= 99.99 {
			q.CorrectIndex = len(q.Options)
			correct++
		}
		q.Options = append(q.Options, text)
	}
	return q, correct == 1 && len(q.Options) >= 2 && len(q.Options) <= 8 && len(q.Prompt) >= 2
}
func parseMoodle(files map[string][]byte) (importPlan, error) {
	p := importPlan{Warnings: []string{}, Activities: []importedActivity{}, Sections: []string{}, QuestionBank: []BankQuestion{}}
	if _, err := parseXML(files["moodle_backup.xml"]); err != nil {
		return p, fmt.Errorf("неверный moodle_backup.xml")
	}
	roots := map[string]*xmlNode{}
	for name, data := range files {
		n, err := parseXML(data)
		if err != nil {
			return p, fmt.Errorf("неверный XML %s", name)
		}
		roots[name] = n
	}
	if root := roots["course/course.xml"]; root != nil {
		if cs := root.all("course"); len(cs) > 0 {
			p.Source = plain(cs[0].val("fullname"))
		}
	}
	questions := map[string]BankQuestion{}
	bankEntries := map[string]BankQuestion{}
	bankVersions := map[string]BankQuestion{}
	if root := roots["questions.xml"]; root != nil {
		all := root.all("question")
		for _, n := range all {
			if q, ok := backupQuestion(n); ok {
				questions[n.Attrs["id"]] = q
				p.QuestionBank = append(p.QuestionBank, q)
			} else if n.val("qtype") != "" {
				p.Warnings = append(p.Warnings, "Пропущен вопрос типа «"+n.val("qtype")+"»: поддерживаются multichoice с одним ответом и truefalse")
			}
		}
		for _, entry := range root.all("question_bank_entry") {
			version := -1
			var latest *xmlNode
			for _, v := range entry.all("question_version") {
				ver, _ := strconv.Atoi(v.val("version"))
				for _, n := range v.all("question") {
					if q, ok := backupQuestion(n); ok {
						bankVersions[entry.Attrs["id"]+"/"+strconv.Itoa(ver)] = q
					}
				}
				if v.val("status") == "hidden" {
					continue
				}
				if ver >= version {
					latest = nil
					if nodes := v.all("question"); len(nodes) > 0 {
						latest = nodes[0]
					}
					version = ver
				}
			}
			if latest != nil {
				if q, ok := backupQuestion(latest); ok {
					bankEntries[entry.Attrs["id"]] = q
				}
			}
		}
	}
	type sectionSpec struct {
		number    int
		id, title string
	}
	ss := []sectionSpec{}
	for name, n := range roots {
		if strings.HasPrefix(name, "sections/") && path.Base(name) == "section.xml" {
			nodes := n.all("section")
			if len(nodes) == 0 {
				continue
			}
			s := nodes[0]
			num, _ := strconv.Atoi(s.val("number"))
			title := plain(s.val("name"))
			if title == "" || title == "$@NULL@$" {
				if num == 0 {
					title = "Общее"
				} else {
					title = fmt.Sprintf("Модуль %d", num)
				}
			}
			ss = append(ss, sectionSpec{num, s.Attrs["id"], title})
		}
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].number < ss[j].number })
	sectionNames := map[string]string{}
	for _, s := range ss {
		sectionNames[s.id] = s.title
		p.Sections = append(p.Sections, s.title)
	}
	names := []string{}
	for name := range roots {
		if strings.HasPrefix(name, "activities/") {
			base := path.Base(name)
			if base != "module.xml" && base != "roles.xml" && base != "inforef.xml" && base != "filters.xml" && base != "grade_history.xml" && base != "grades.xml" && base != "completion.xml" && base != "comments.xml" && base != "calendar.xml" && base != "competencies.xml" {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	for _, name := range names {
		kind := strings.TrimSuffix(path.Base(name), ".xml")
		root := roots[name]
		nodes := root.all(kind)
		if len(nodes) == 0 {
			continue
		}
		n := nodes[0]
		v := importedActivity{Kind: kind, Title: plain(n.val("name")), Body: plain(n.val("intro")), MaxScore: 100}
		if v.Title == "" {
			v.Title = "Материал Moodle"
		}
		module := roots[path.Dir(name)+"/module.xml"]
		if module != nil {
			mods := module.all("module")
			if len(mods) > 0 {
				v.Section = sectionNames[mods[0].val("sectionid")]
				if v.Section == "" {
					v.Section = sectionNames[mods[0].val("section")]
				}
			}
		}
		if v.Section == "" {
			v.Section = "Импорт из Moodle"
		}
		supported := true
		switch kind {
		case "page":
			v.Body = plain(n.val("content"))
			v.Kind = "page"
		case "label":
			v.Kind = "label"
		case "url":
			v.URL = n.val("externalurl")
			if !validURL(v.URL) {
				p.Warnings = append(p.Warnings, "Пропущена некорректная ссылка: "+v.Title)
				supported = false
			}
		case "book":
			parts := []string{}
			for _, ch := range n.all("chapter") {
				parts = append(parts, plain(ch.val("title"))+"\n\n"+plain(ch.val("content")))
			}
			v.Body = strings.Join(parts, "\n\n")
			v.Kind = "page"
		case "assign":
			v.Kind = "assignment"
			if grade, err := strconv.ParseFloat(n.val("grade"), 64); err == nil && grade >= 1 {
				v.MaxScore = int(grade)
			}
			if ts, err := strconv.ParseInt(n.val("duedate"), 10, 64); err == nil && ts > 0 {
				v.DueAt = time.Unix(ts, 0)
			} else {
				v.DueAt = time.Now().AddDate(0, 0, 7)
				p.Warnings = append(p.Warnings, "Заданию «"+v.Title+"» без срока установлен срок через 7 дней")
			}
		case "quiz":
			refs := []*xmlNode{}
			for _, slot := range n.all("question_instance") {
				refs = append(refs, slot)
			}
			if len(refs) > 0 {
				for _, ref := range refs {
					q, ok := questions[ref.val("questionid")]
					if !ok {
						q, ok = questions[ref.val("question")]
					}
					if !ok {
						supported = false
						break
					}
					v.Questions = append(v.Questions, q)
				}
			} else {
				slots := n.all("question_slot")
				if len(slots) == 0 {
					slots = n.all("slot")
				}
				for _, slot := range slots {
					var q BankQuestion
					ok := false
					ids := slot.all("questionbankentryid")
					if len(ids) > 0 {
						entryID := strings.TrimSpace(ids[0].Text)
						pinned := ""
						if refs := slot.all("question_reference"); len(refs) > 0 {
							pinned = refs[0].val("version")
						}
						if pinned != "" && pinned != "0" && pinned != "$@NULL@$" {
							q, ok = bankVersions[entryID+"/"+pinned]
						} else {
							q, ok = bankEntries[entryID]
						}
					} else {
						q, ok = questions[slot.val("questionid")]
						if !ok {
							q, ok = questions[slot.val("question")]
						}
					}
					if !ok {
						supported = false
						break
					}
					v.Questions = append(v.Questions, q)
				}
			}
			if len(v.Questions) == 0 {
				supported = false
			}
			if !supported {
				p.Warnings = append(p.Warnings, "Тест «"+v.Title+"» пропущен целиком: не удалось сопоставить все вопросы или их типы не поддерживаются")
			}
		default:
			supported = false
			p.Warnings = append(p.Warnings, "Элемент «"+kind+"» («"+v.Title+"») не поддерживается и пропущен")
		}
		if supported {
			bank := BankItem{Title: v.Title, Kind: v.Kind, Body: v.Body, URL: v.URL, MaxScore: v.MaxScore, Questions: v.Questions}
			if msg := validateBank(bank); msg != "" {
				p.Warnings = append(p.Warnings, "Пропущен элемент «"+v.Title+"»: "+msg)
				continue
			}
			p.Activities = append(p.Activities, v)
		}
	}
	if len(p.Activities) == 0 {
		return p, fmt.Errorf("в бэкапе не найдено поддерживаемых элементов; поддерживаются page, book, label, url, assign, quiz с multichoice/truefalse")
	}
	p.Warnings = append(p.Warnings, "HTML преобразован в текст. Вложения, пользователи, оценки, попытки, блоки, ограничения доступа и плагины Moodle не переносятся.")
	return p, nil
}
func (a *app) importMoodle(w http.ResponseWriter, r *http.Request) {
	c, ok := a.access(w, r, id(r, "course"), true)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBackup+1<<20)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		fail(w, 400, "Не удалось прочитать файл. Максимальный размер бэкапа — 32 МБ")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	f, _, err := r.FormFile("backup")
	if err != nil {
		fail(w, 400, "Выберите .mbz файл в поле backup")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxBackup+1))
	if err != nil || len(data) > maxBackup {
		fail(w, 400, "Бэкап превышает 32 МБ")
		return
	}
	files, skipped, err := readMoodleArchive(data)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	plan, err := parseMoodle(files)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	sectionIDs := map[string]uint{}
	counts := map[string]int{}
	bankCount := 0
	uid := current(r).ID
	err = a.db.Transaction(func(tx *gorm.DB) error {
		var position int64
		if err := tx.Model(&Section{}).Where("course_id = ?", c.ID).Count(&position).Error; err != nil {
			return err
		}
		getSection := func(title string) (uint, error) {
			if id, ok := sectionIDs[title]; ok {
				return id, nil
			}
			position++
			s := Section{CourseID: c.ID, Title: title, Position: int(position)}
			if err := tx.Create(&s).Error; err != nil {
				return 0, err
			}
			sectionIDs[title] = s.ID
			return s.ID, nil
		}
		for _, title := range plan.Sections {
			if _, err := getSection(title); err != nil {
				return err
			}
		}
		for _, v := range plan.Activities {
			sid, err := getSection(v.Section)
			if err != nil {
				return err
			}
			bank := BankItem{AuthorID: uid, Title: v.Title, Kind: v.Kind, Body: v.Body, URL: v.URL, Duration: 15, MaxScore: v.MaxScore, Questions: v.Questions, Category: "Moodle: " + plan.Source}
			if len(bank.Category) > 100 {
				bank.Category = "Импорт Moodle"
			}
			if err := copyBank(tx, bank, c.ID, sid, v.DueAt); err != nil {
				return err
			}
			if err := tx.Create(&bank).Error; err != nil {
				return err
			}
			counts[v.Kind]++
		}
		// Reusable question bank entries remain staff-only, including answer keys.
		for i, q := range plan.QuestionBank {
			title := fmt.Sprintf("Вопрос Moodle %d", i+1)
			if len(q.Prompt) < 180 {
				title = q.Prompt
			}
			bank := BankItem{AuthorID: uid, Title: title, Kind: "quiz", Category: "Банк вопросов Moodle", Questions: []BankQuestion{q}}
			if validateBank(bank) == "" {
				if err := tx.Create(&bank).Error; err != nil {
					return err
				}
				bankCount++
			}
		}
		return nil
	})
	if !a.result(w, err) {
		return
	}
	reply(w, 201, map[string]any{"source": plan.Source, "sections": len(sectionIDs), "activities": len(plan.Activities), "counts": counts, "question_bank": bankCount, "warnings": plan.Warnings, "skipped_files": skipped})
}
