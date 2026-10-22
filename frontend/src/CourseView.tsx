import { useEffect, useState } from "react";
import {
  ArrowLeft,
  Plus,
  Check,
  ChevronRight,
  Clock,
  BookOpen,
  FileText,
  ClipboardCheck,
  MessagesSquare,
  GraduationCap,
  Megaphone,
  Send,
  CheckCircle2,
  Lock,
  Users,
  Pencil,
  Trash2,
  ArrowUp,
  ArrowDown,
  EyeOff,
  Library,
} from "lucide-react";
import {
  api,
  type Course,
  type User,
  type Detail,
  type Section,
  activityKinds,
  type Lesson,
  type Quiz,
  type Assignment,
  type Submission,
  type Gradebook,
  date,
  initials,
} from "./types";
import { Cover, Empty, Field, FormModal, Modal, Submit } from "./components";
import { BankView, CourseCohortsModal, MoodleImport } from "./Extended";
import {
  ActivityEditor,
  ActivityViewer,
  ElementPicker,
  TopicEditor,
  icons,
} from "./CourseEditor";
const localDate = (value?: string | null) =>
  value
    ? new Date(
        new Date(value).getTime() - new Date(value).getTimezoneOffset() * 60000,
      )
        .toISOString()
        .slice(0, 16)
    : "";
const value = (f: FormData, k: string) => String(f.get(k) || "");
export function CourseForm({
  course,
  close,
  saved,
}: {
  course?: Course;
  close: () => void;
  saved: () => Promise<void>;
}) {
  return (
    <FormModal
      title={course ? "Настройки курса" : "Новый курс"}
      close={close}
      submit={async (f) => {
        await api(
          "/courses" + (course ? "/" + course.id : ""),
          course ? "PATCH" : "POST",
          {
            title: value(f, "title"),
            description: value(f, "description"),
            category: value(f, "category"),
            color: value(f, "color"),
            published: f.get("published") === "on",
            short_name: value(f, "short_name"),
            format: value(f, "format"),
            language: value(f, "language"),
            disable_self_enrollment: f.get("self_enrollment") !== "on",
            start_at: f.get("start_at")
              ? new Date(value(f, "start_at")).toISOString()
              : null,
            end_at: f.get("end_at")
              ? new Date(value(f, "end_at")).toISOString()
              : null,
          },
        );
        await saved();
      }}
    >
      <Field label="Название">
        <input
          name="title"
          required
          minLength={3}
          maxLength={200}
          defaultValue={course?.title}
        />
      </Field>
      <Field label="Описание">
        <textarea
          name="description"
          rows={4}
          maxLength={10000}
          defaultValue={course?.description}
        />
      </Field>
      <div className="grid grid-cols-2 gap-4">
        <Field label="Категория">
          <input
            name="category"
            defaultValue={course?.category || "Разработка"}
          />
        </Field>
        <Field label="Обложка">
          <select name="color" defaultValue={course?.color || "violet"}>
            <option value="violet">Лавандовая</option>
            <option value="blue">Голубая</option>
            <option value="green">Мятная</option>
            <option value="orange">Персиковая</option>
          </select>
        </Field>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Field label="Короткое название">
          <input
            name="short_name"
            maxLength={80}
            defaultValue={course?.short_name}
            placeholder="Например, GO-101"
          />
        </Field>
        <Field label="Формат курса">
          <select name="format" defaultValue={course?.format || "topics"}>
            <option value="topics">Темы</option>
            <option value="weeks">Недели</option>
          </select>
        </Field>
        <Field label="Начало доступа">
          <input
            type="datetime-local"
            name="start_at"
            defaultValue={localDate(course?.start_at)}
          />
        </Field>
        <Field label="Окончание доступа">
          <input
            type="datetime-local"
            name="end_at"
            defaultValue={localDate(course?.end_at)}
          />
        </Field>
        <Field label="Язык материалов">
          <select name="language" defaultValue={course?.language || "ru"}>
            <option value="ru">Русский</option>
            <option value="en">English</option>
          </select>
        </Field>
      </div>
      <p className="muted text-xs">
        Язык обозначает язык материалов; интерфейс платформы остаётся
        русскоязычным. Вне периода доступа студенты не смогут открывать курс.
        Преподаватель сохраняет доступ.
      </p>
      <label className="flex gap-3 items-center text-sm">
        <input
          type="checkbox"
          name="self_enrollment"
          defaultChecked={!course?.disable_self_enrollment}
        />{" "}
        Разрешить самостоятельную запись
      </label>
      <label className="flex gap-3 items-center text-sm">
        <input
          type="checkbox"
          name="published"
          defaultChecked={course?.published}
        />{" "}
        Опубликовать — показывать курс в каталоге
      </label>
    </FormModal>
  );
}
export function GradeTable({ courseId }: { courseId: number }) {
  const [data, setData] = useState<Gradebook>(),
    [error, setError] = useState("");
  useEffect(() => {
    api<Gradebook>("/courses/" + courseId + "/grades")
      .then(setData)
      .catch((e) => setError(e.message));
  }, [courseId]);
  if (error) return <p className="error">{error}</p>;
  if (!data) return <p className="muted p-6">Загружаем журнал…</p>;
  if (!data.rows.length) return <Empty title="В журнале пока нет студентов" />;
  return (
    <div className="table-scroll">
      <table>
        <thead>
          <tr>
            <th>Студент</th>
            <th>Прогресс</th>
            {data.assignments.map((a) => (
              <th key={a.id}>
                {a.title}
                <small>из {a.max_score} баллов</small>
              </th>
            ))}
            {data.quizzes.map((q) => (
              <th key={q.id}>
                {q.title}
                <small>лучшая попытка</small>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.rows.map((row) => (
            <tr key={row.user.id}>
              <td>
                <div className="flex items-center gap-3">
                  <span className="avatar">{initials(row.user.name)}</span>
                  <div>
                    <b>{row.user.name}</b>
                    <small>{row.user.email}</small>
                  </div>
                </div>
              </td>
              <td>
                {row.completed} / {row.lessons}
              </td>
              {data.assignments.map((a) => {
                const s = row.submissions.find((s) => s.assignment_id === a.id);
                return (
                  <td key={a.id}>
                    {s ? (
                      s.score === null ? (
                        <span className="badge orange">На проверке</span>
                      ) : (
                        <b>{s.score}</b>
                      )
                    ) : (
                      "—"
                    )}
                  </td>
                );
              })}
              {data.quizzes.map((q) => {
                const attempts = row.attempts.filter((a) => a.quiz_id === q.id);
                const score = attempts.length
                  ? Math.max(
                      ...attempts.map((a) =>
                        Math.round((a.score / a.total) * 100),
                      ),
                    )
                  : null;
                return <td key={q.id}>{score === null ? "—" : score + "%"}</td>;
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
export default function CourseView({
  id,
  user,
  courses,
  back,
  refresh,
  notify,
}: {
  id: number;
  user: User;
  courses: Course[];
  back: () => void;
  refresh: () => Promise<void>;
  notify: (s: string) => void;
}) {
  const [data, setData] = useState<Detail>(),
    [error, setError] = useState(""),
    [tab, setTab] = useState("content"),
    [lesson, setLesson] = useState<Lesson>(),
    [quiz, setQuiz] = useState<Quiz>(),
    [assignment, setAssignment] = useState<Assignment>(),
    [modal, setModal] = useState(""),
    [section, setSection] = useState(0),
    [topic, setTopic] = useState<Section>(),
    [editing, setEditing] = useState<Lesson>(),
    [kind, setKind] = useState("page"),
    [review, setReview] = useState<Submission>(),
    [message, setMessage] = useState(""),
    [busy, setBusy] = useState(false);
  const course = courses.find((c) => c.id === id);
  const canEdit =
    !!course &&
    (user.role === "admin" ||
      (user.role === "teacher" && course.teacher_id === user.id));
  const canRead = canEdit || course?.enrolled;
  const load = async () => {
    setData(await api<Detail>("/courses/" + id));
    setError("");
  };
  useEffect(() => {
    setData(undefined);
    setError("");
    if (canRead) void load().catch((e) => setError(e.message));
  }, [id, canRead]);
  const saved = async () => {
    await Promise.all([load(), refresh()]);
    notify("Изменения сохранены");
  };
  const perform = async (fn: () => Promise<unknown>, success: string) => {
    setBusy(true);
    try {
      await fn();
      await saved();
      notify(success);
    } catch (e) {
      notify((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  if (!course)
    return (
      <Empty title="Курс не найден">
        <button className="btn mt-4" onClick={back}>
          Вернуться к курсам
        </button>
      </Empty>
    );
  const total = data?.sections.reduce((n, s) => n + s.lessons.length, 0) || 0,
    progress = total
      ? Math.round(((data?.completed.length || 0) / total) * 100)
      : 0;
  return (
    <div className="space-y-6">
      <button className="back-link" onClick={back}>
        <ArrowLeft size={16} /> Все курсы
      </button>
      <div className="course-hero">
        <div className="flex-1 p-7 sm:p-9">
          <div className="flex gap-3 mb-4">
            <span className="badge">{course.category}</span>
            {!course.published && (
              <span className="badge orange">Черновик</span>
            )}
          </div>
          <h1>{course.title}</h1>
          <p className="text-slate-500 mt-4 max-w-xl leading-relaxed">
            {course.description}
          </p>
          <div className="flex flex-wrap gap-5 mt-6 text-sm text-slate-500">
            <span className="flex gap-2 items-center">
              <span className="avatar tiny">
                {initials(course.teacher.name)}
              </span>
              {course.teacher.name}
            </span>
            <span className="flex gap-2 items-center">
              <BookOpen size={16} />
              {course.lessons} уроков
            </span>
            <span className="flex gap-2 items-center">
              <Users size={16} />
              {course.students} студентов
            </span>
          </div>
        </div>
        <Cover course={course} />
      </div>
      {(course.short_name || course.start_at || course.end_at) && (
        <div className="flex flex-wrap gap-3 text-xs muted">
          <span>{course.short_name}</span>
          {course.start_at && (
            <span>Начало доступа: {date(course.start_at)}</span>
          )}
          {course.end_at && (
            <span>Окончание доступа: {date(course.end_at)}</span>
          )}
        </div>
      )}
      {!canRead ? (
        <div className="panel flex flex-wrap items-center justify-between gap-5 p-7">
          <div>
            <h3>
              {course.disable_self_enrollment
                ? "Запись через преподавателя"
                : "Начните учиться уже сегодня"}
            </h3>
            <p className="muted mt-2">
              {course.disable_self_enrollment
                ? "Для записи обратитесь к преподавателю: доступ можно назначить через глобальную группу."
                : "Запишитесь, чтобы открыть уроки, задания и обсуждение."}
            </p>
          </div>
          <button
            className="btn primary"
            disabled={
              busy ||
              course.disable_self_enrollment ||
              (!!course.start_at && new Date(course.start_at) > new Date()) ||
              (!!course.end_at && new Date(course.end_at) < new Date())
            }
            onClick={() =>
              void perform(
                () => api("/courses/" + id + "/enroll", "POST"),
                "Вы записаны на курс",
              )
            }
          >
            Записаться на курс <ChevronRight size={17} />
          </button>
        </div>
      ) : (
        <>
          <div className="flex flex-wrap justify-between items-center gap-4">
            <div className="tabs">
              {[
                { id: "content", text: "Программа", icon: BookOpen },
                { id: "tasks", text: "Задания и тесты", icon: ClipboardCheck },
                { id: "discussion", text: "Обсуждение", icon: MessagesSquare },
                { id: "grades", text: "Журнал оценок", icon: GraduationCap },
              ].map((t) => (
                <button
                  className={tab === t.id ? "active" : ""}
                  key={t.id}
                  onClick={() => setTab(t.id)}
                >
                  <t.icon size={17} />
                  {t.text}
                </button>
              ))}
            </div>
            {canEdit && (
              <div className="flex flex-wrap gap-2">
                <button className="btn" onClick={() => setModal("bank")}>
                  Банк контента
                </button>
                <button className="btn" onClick={() => setModal("cohorts")}>
                  Группы
                </button>
                <button className="btn" onClick={() => setModal("import")}>
                  Импорт Moodle
                </button>
                <button className="btn" onClick={() => setModal("course")}>
                  Настройки
                </button>
              </div>
            )}
          </div>
          {error && (
            <div role="alert" className="error">
              {error}
              <button
                className="ml-4 underline"
                onClick={() => void load().catch((e) => setError(e.message))}
              >
                Повторить
              </button>
            </div>
          )}
          {!data && !error && (
            <p className="muted p-6">Загружаем материалы курса…</p>
          )}
          {data && tab === "content" && (
            <div className="course-columns">
              <div className="space-y-5">
                <div className="section-heading">
                  <h2>
                    Программа курса{" "}
                    <span className="count">{data.sections.length}</span>
                  </h2>
                  {canEdit && (
                    <button
                      className="text-button"
                      onClick={() => {
                        setTopic(undefined);
                        setModal("section");
                      }}
                    >
                      <Plus size={16} /> Добавить тему
                    </button>
                  )}
                </div>
                {data.sections.map((s, i) => (
                  <section
                    className={
                      "panel module " + (s.hidden ? "hidden-topic" : "")
                    }
                    key={s.id}
                  >
                    <div className="module-head">
                      <span className="module-num">
                        {String(i + 1).padStart(2, "0")}
                      </span>
                      <div className="flex-1">
                        <h3>
                          {s.title}{" "}
                          {s.hidden && (
                            <span className="badge">
                              <EyeOff size={12} /> Скрытая тема
                            </span>
                          )}
                        </h3>
                        <p className="muted text-xs mt-1">
                          {course.format === "weeks" ? "Неделя" : "Тема"}{" "}
                          {i + 1} ·{" "}
                          {s.lessons.length +
                            data.assignments.filter(
                              (a) => a.section_id === s.id,
                            ).length +
                            data.quizzes.filter((q) => q.section_id === s.id)
                              .length}{" "}
                          элементов
                          {course.format === "weeks" && course.start_at
                            ? " · " +
                              date(
                                new Date(
                                  new Date(course.start_at).getTime() +
                                    i * 7 * 86400000,
                                ).toISOString(),
                              )
                            : ""}
                        </p>
                      </div>
                      {canEdit && (
                        <div className="topic-tools">
                          <button
                            className="icon-btn"
                            disabled={busy || i === 0}
                            aria-label={"Поднять тему " + s.title}
                            onClick={() =>
                              void perform(
                                () =>
                                  api(
                                    "/courses/" + id + "/sections/order",
                                    "PUT",
                                    {
                                      ids: data.sections
                                        .map((v) => v.id)
                                        .map((v, j) =>
                                          j === i - 1
                                            ? s.id
                                            : j === i
                                              ? data.sections[i - 1].id
                                              : v,
                                        ),
                                    },
                                  ),
                                "Порядок тем обновлён",
                              )
                            }
                          >
                            <ArrowUp size={15} />
                          </button>
                          <button
                            className="icon-btn"
                            disabled={busy || i === data.sections.length - 1}
                            aria-label={"Опустить тему " + s.title}
                            onClick={() =>
                              void perform(
                                () =>
                                  api(
                                    "/courses/" + id + "/sections/order",
                                    "PUT",
                                    {
                                      ids: data.sections
                                        .map((v) => v.id)
                                        .map((v, j) =>
                                          j === i + 1
                                            ? s.id
                                            : j === i
                                              ? data.sections[i + 1].id
                                              : v,
                                        ),
                                    },
                                  ),
                                "Порядок тем обновлён",
                              )
                            }
                          >
                            <ArrowDown size={15} />
                          </button>
                          <button
                            className="icon-btn"
                            aria-label={"Настроить тему " + s.title}
                            onClick={() => {
                              setTopic(s);
                              setModal("section");
                            }}
                          >
                            <Pencil size={15} />
                          </button>
                          <button
                            className="icon-btn"
                            aria-label={"Удалить тему " + s.title}
                            disabled={busy}
                            onClick={() => {
                              if (
                                window.confirm(
                                  "Удалить тему «" +
                                    s.title +
                                    "»? Удаление возможно только для пустой темы.",
                                )
                              )
                                void perform(
                                  () => api("/sections/" + s.id, "DELETE"),
                                  "Тема удалена",
                                );
                            }}
                          >
                            <Trash2 size={15} />
                          </button>
                        </div>
                      )}
                    </div>
                    {s.summary && <p className="topic-summary">{s.summary}</p>}
                    {s.lessons.map((l) => {
                      const Icon = icons[l.kind || "page"] || FileText;
                      return (
                        <div className="element-row" key={l.id}>
                          <button
                            className="lesson-row"
                            onClick={() => setLesson(l)}
                          >
                            <span
                              className={
                                "lesson-check " +
                                (data.completed.includes(l.id) ? "done" : "")
                              }
                            >
                              {data.completed.includes(l.id) ? (
                                <Check size={15} />
                              ) : (
                                <Icon size={15} />
                              )}
                            </span>
                            <span className="flex-1 text-left">
                              <b className="font-medium">{l.title}</b>
                              <small>
                                {activityKinds[l.kind || "page"] || "Страница"}
                                {l.kind === "folder"
                                  ? " · " + (l.files?.length || 0) + " файлов"
                                  : ""}
                              </small>
                              {l.kind === "label" && (
                                <span className="inline-label">
                                  {l.content}
                                </span>
                              )}
                            </span>
                            <ChevronRight
                              size={15}
                              className="text-slate-400"
                            />
                          </button>
                          {canEdit && (
                            <div className="element-tools">
                              {l.kind !== "file" && l.kind !== "folder" && (
                                <button
                                  className="icon-btn"
                                  aria-label={"Сохранить в банк " + l.title}
                                  disabled={busy}
                                  onClick={() =>
                                    void perform(
                                      () =>
                                        api(
                                          "/lessons/" + l.id + "/bank",
                                          "POST",
                                        ),
                                      "Элемент сохранён в банк контента",
                                    )
                                  }
                                >
                                  <Library size={14} />
                                </button>
                              )}
                              <button
                                className="icon-btn"
                                aria-label={"Редактировать элемент " + l.title}
                                onClick={() => {
                                  setEditing(l);
                                  setKind(l.kind || "page");
                                  setSection(s.id);
                                  setModal("activity");
                                }}
                              >
                                <Pencil size={14} />
                              </button>
                              <button
                                className="icon-btn"
                                aria-label={"Удалить элемент " + l.title}
                                disabled={busy}
                                onClick={() => {
                                  if (
                                    window.confirm(
                                      "Удалить элемент «" +
                                        l.title +
                                        "», его файлы и прогресс прохождения?",
                                    )
                                  )
                                    void perform(
                                      () => api("/lessons/" + l.id, "DELETE"),
                                      "Элемент удалён",
                                    );
                                }}
                              >
                                <Trash2 size={14} />
                              </button>
                            </div>
                          )}
                        </div>
                      );
                    })}
                    {data.assignments
                      .filter((a) => a.section_id === s.id)
                      .map((a) => (
                        <button
                          key={"a" + a.id}
                          className="lesson-row"
                          onClick={() => {
                            setTab("tasks");
                            if (!canEdit) setAssignment(a);
                          }}
                        >
                          <span className="activity-icon orange">
                            <FileText size={17} />
                          </span>
                          <span className="flex-1 text-left">
                            {a.title}
                            <small>Задание · до {a.max_score} баллов</small>
                          </span>
                          <ChevronRight size={15} />
                        </button>
                      ))}
                    {data.quizzes
                      .filter((q) => q.section_id === s.id)
                      .map((q) => (
                        <button
                          key={"q" + q.id}
                          className="lesson-row"
                          onClick={() => {
                            setTab("tasks");
                            if (!canEdit) setQuiz(q);
                          }}
                        >
                          <span className="activity-icon violet">
                            <ClipboardCheck size={17} />
                          </span>
                          <span className="flex-1 text-left">
                            {q.title}
                            <small>Тест · {q.questions.length} вопросов</small>
                          </span>
                          <ChevronRight size={15} />
                        </button>
                      ))}
                    {canEdit && (
                      <button
                        className="add-element"
                        onClick={() => {
                          setSection(s.id);
                          setEditing(undefined);
                          setModal("picker");
                        }}
                      >
                        <Plus size={16} /> Добавить элемент или ресурс
                      </button>
                    )}
                  </section>
                ))}
                {!data.sections.length && (
                  <Empty title="Программа ещё готовится" />
                )}
              </div>
              <aside className="space-y-5">
                <div className="panel p-6">
                  <span className="eyebrow">ВАШ ПУТЬ</span>
                  <div className="flex justify-between items-end mt-4 mb-3">
                    <h2 className="text-3xl">{progress}%</h2>
                    <span className="muted text-sm">
                      {data.completed.length} из {total}
                    </span>
                  </div>
                  <div className="progress">
                    <i style={{ width: progress + "%" }} />
                  </div>
                  <p className="muted text-xs mt-4 leading-relaxed">
                    Отмечайте пройденные уроки, чтобы видеть свой прогресс.
                  </p>
                </div>
                <div className="panel p-6">
                  <div className="section-heading mb-4">
                    <h3 className="flex gap-2 items-center">
                      <Megaphone size={18} /> Объявления
                    </h3>
                    {canEdit && (
                      <button
                        className="icon-btn"
                        aria-label="Новое объявление"
                        onClick={() => setModal("announcement")}
                      >
                        <Plus size={17} />
                      </button>
                    )}
                  </div>
                  {data.announcements.map((a) => (
                    <div className="announcement" key={a.id}>
                      <span className="eyebrow">{date(a.created_at)}</span>
                      <h4 className="font-semibold mt-2">{a.title}</h4>
                      <p className="muted text-sm mt-2 whitespace-pre-wrap leading-relaxed">
                        {a.body}
                      </p>
                    </div>
                  ))}
                  {!data.announcements.length && (
                    <p className="muted text-sm">Новых объявлений пока нет.</p>
                  )}
                </div>
              </aside>
            </div>
          )}
          {data && tab === "tasks" && (
            <div className="space-y-6">
              <div className="section-heading">
                <h2>Практические задания</h2>
                {canEdit && (
                  <button
                    className="btn"
                    onClick={() => {
                      setSection(0);
                      setModal("assignment");
                    }}
                  >
                    <Plus size={16} /> Задание
                  </button>
                )}
              </div>
              {data.assignments.map((as) => {
                const sub = data.submissions.find(
                  (s) => s.assignment_id === as.id && s.user_id === user.id,
                );
                return (
                  <div className="panel p-6" key={as.id}>
                    <div className="flex flex-wrap justify-between gap-4">
                      <div className="flex gap-4">
                        <span className="activity-icon orange">
                          <FileText size={22} />
                        </span>
                        <div>
                          <h3>{as.title}</h3>
                          <p className="muted text-sm mt-2">
                            Срок: {date(as.due_at)} · до {as.max_score} баллов
                          </p>
                        </div>
                      </div>
                      {!canEdit && (
                        <button
                          className="btn primary"
                          onClick={() => setAssignment(as)}
                        >
                          {sub
                            ? sub.score === null
                              ? "Изменить ответ"
                              : "Посмотреть результат"
                            : "Сдать работу"}
                          <ChevronRight size={16} />
                        </button>
                      )}
                    </div>
                    <p className="muted text-sm mt-4 whitespace-pre-wrap">
                      {as.description}
                    </p>
                    {sub && (
                      <div className="mt-4">
                        <span
                          className={
                            "badge " + (sub.score === null ? "orange" : "green")
                          }
                        >
                          {sub.score === null
                            ? "На проверке"
                            : `${sub.score} / ${as.max_score} баллов`}
                        </span>
                        {sub.feedback && (
                          <p className="text-sm mt-3">
                            Комментарий преподавателя: {sub.feedback}
                          </p>
                        )}
                      </div>
                    )}
                    {canEdit && (
                      <div className="mt-5 border-t border-slate-100 pt-4">
                        <TopicPlacement
                          kind="assignments"
                          itemId={as.id}
                          sectionId={as.section_id}
                          sections={data.sections}
                          saved={saved}
                          notify={notify}
                        />
                        <span className="eyebrow">РАБОТЫ СТУДЕНТОВ</span>
                        {data.submissions
                          .filter((s) => s.assignment_id === as.id)
                          .map((s) => (
                            <button
                              className="review-row"
                              key={s.id}
                              onClick={() => setReview(s)}
                            >
                              <span className="avatar tiny">
                                {initials(s.user.name)}
                              </span>
                              <span className="flex-1 text-left">
                                {s.user.name}
                              </span>
                              <span className="badge">
                                {s.score === null
                                  ? "Проверить"
                                  : `${s.score} / ${as.max_score}`}
                              </span>
                              <ChevronRight size={16} />
                            </button>
                          ))}
                        {!data.submissions.some(
                          (s) => s.assignment_id === as.id,
                        ) && (
                          <p className="muted text-sm mt-3">
                            Пока никто не сдал работу.
                          </p>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
              {!data.assignments.length && <Empty title="Заданий пока нет" />}
              <div className="section-heading">
                <h2>Проверьте себя</h2>
                {canEdit && (
                  <button
                    className="btn"
                    onClick={() => {
                      setSection(0);
                      setModal("quiz");
                    }}
                  >
                    <Plus size={16} /> Тест
                  </button>
                )}
              </div>
              {data.quizzes.map((q) => {
                const attempts = data.attempts.filter(
                  (a) => a.quiz_id === q.id && a.user_id === user.id,
                );
                return (
                  <div
                    className="panel p-6 flex flex-wrap items-center gap-4"
                    key={q.id}
                  >
                    <span className="activity-icon violet">
                      <ClipboardCheck size={23} />
                    </span>
                    <div className="flex-1">
                      <h3>{q.title}</h3>
                      <p className="muted text-sm mt-2">
                        {q.questions.length} вопроса · до 3 попыток
                        {attempts.length > 0 &&
                          ` · лучший результат ${Math.max(...attempts.map((a) => Math.round((a.score / a.total) * 100)))}%`}
                      </p>
                    </div>
                    {canEdit && (
                      <TopicPlacement
                        kind="quizzes"
                        itemId={q.id}
                        sectionId={q.section_id}
                        sections={data.sections}
                        saved={saved}
                        notify={notify}
                      />
                    )}
                    {!canEdit && (
                      <button
                        className="btn primary"
                        disabled={attempts.length >= 3}
                        onClick={() => setQuiz(q)}
                      >
                        {attempts.length >= 3
                          ? "Попытки закончились"
                          : attempts.length
                            ? "Пройти ещё раз"
                            : "Начать тест"}
                        <ChevronRight size={16} />
                      </button>
                    )}
                  </div>
                );
              })}
            </div>
          )}
          {data && tab === "discussion" && (
            <div className="panel p-6 max-w-4xl">
              <h2 className="mb-2">Учиться интереснее вместе</h2>
              <p className="muted mb-6">
                Задавайте вопросы и делитесь находками с участниками курса.
              </p>
              <form
                className="flex gap-3 items-end mb-7"
                onSubmit={(e) => {
                  e.preventDefault();
                  void perform(async () => {
                    await api("/courses/" + id + "/discussions", "POST", {
                      body: message,
                    });
                    setMessage("");
                  }, "Сообщение опубликовано");
                }}
              >
                <label className="flex-1">
                  <span className="sr-only">Ваше сообщение</span>
                  <textarea
                    required
                    minLength={2}
                    maxLength={10000}
                    rows={3}
                    value={message}
                    onChange={(e) => setMessage(e.target.value)}
                    placeholder="О чём хотите спросить?"
                  />
                </label>
                <button
                  type="submit"
                  className="btn primary"
                  disabled={busy}
                  aria-label="Отправить сообщение"
                >
                  <Send size={18} />
                </button>
              </form>
              {data.discussions.map((d) => (
                <div className="discussion-post" key={d.id}>
                  <span className="avatar">{initials(d.user.name)}</span>
                  <div>
                    <div className="flex gap-3 items-center">
                      <b className="text-sm">{d.user.name}</b>
                      <span className="eyebrow">{date(d.created_at)}</span>
                    </div>
                    <p className="text-sm text-slate-600 mt-2 whitespace-pre-wrap leading-relaxed">
                      {d.body}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
          {data && tab === "grades" && (
            <div className="panel">
              <div className="p-6 border-b border-slate-100">
                <h2>Журнал оценок</h2>
                <p className="muted text-sm mt-2">
                  Баллы за задания, лучшие результаты тестов и прогресс по
                  урокам.
                </p>
              </div>
              <GradeTable courseId={id} />
            </div>
          )}
        </>
      )}
      {lesson && data && (
        <ActivityViewer
          lesson={lesson}
          user={user}
          completed={data.completed.includes(lesson.id)}
          close={() => setLesson(undefined)}
          complete={async () => {
            await api("/lessons/" + lesson.id + "/complete", "POST");
            await saved();
          }}
        />
      )}
      {assignment && data && (
        <AssignmentModal
          assignment={assignment}
          submission={data.submissions.find(
            (s) => s.assignment_id === assignment.id && s.user_id === user.id,
          )}
          close={() => setAssignment(undefined)}
          saved={saved}
        />
      )}
      {quiz && (
        <QuizModal quiz={quiz} close={() => setQuiz(undefined)} saved={saved} />
      )}
      {review && data && (
        <FormModal
          title={"Проверка работы · " + review.user.name}
          close={() => setReview(undefined)}
          submit={async (f) => {
            await api("/submissions/" + review.id + "/grade", "PATCH", {
              score: Number(f.get("score")),
              feedback: value(f, "feedback"),
            });
            await saved();
          }}
        >
          <div className="submitted-work">{review.content}</div>
          <Field
            label={
              "Балл (максимум " +
              data.assignments.find((a) => a.id === review.assignment_id)
                ?.max_score +
              ")"
            }
          >
            <input
              type="number"
              name="score"
              min={0}
              max={
                data.assignments.find((a) => a.id === review.assignment_id)
                  ?.max_score
              }
              required
              defaultValue={review.score ?? ""}
            />
          </Field>
          <Field label="Комментарий студенту">
            <textarea name="feedback" rows={3} defaultValue={review.feedback} />
          </Field>
        </FormModal>
      )}
      {modal === "bank" && (
        <BankView
          user={user}
          courses={courses}
          notify={notify}
          refresh={saved}
          chooseCourse={id}
          close={() => setModal("")}
        />
      )}
      {modal === "cohorts" && (
        <CourseCohortsModal
          course={course}
          close={() => setModal("")}
          refresh={saved}
        />
      )}
      {modal === "import" && (
        <MoodleImport
          course={course}
          close={() => setModal("")}
          saved={saved}
        />
      )}
      {modal === "course" && (
        <CourseForm course={course} close={() => setModal("")} saved={saved} />
      )}
      {modal === "section" && (
        <TopicEditor
          topic={topic}
          course={course}
          close={() => setModal("")}
          saved={saved}
        />
      )}
      {modal === "picker" && (
        <ElementPicker
          close={() => setModal("")}
          select={(k) => {
            setKind(k);
            setModal(k === "assignment" || k === "quiz" ? k : "activity");
          }}
        />
      )}
      {modal === "activity" && data && (
        <ActivityEditor
          kind={kind}
          lesson={editing}
          sectionId={section}
          sections={data.sections}
          close={() => setModal("")}
          saved={saved}
        />
      )}
      {modal === "assignment" && (
        <FormModal
          title="Новое задание"
          close={() => setModal("")}
          submit={async (f) => {
            await api("/courses/" + id + "/assignments", "POST", {
              title: value(f, "title"),
              description: value(f, "description"),
              due_at: new Date(value(f, "due_at")).toISOString(),
              max_score: Number(f.get("max_score")),
              section_id: Number(f.get("section_id") || 0),
            });
            await saved();
          }}
        >
          <Field label="Название">
            <input required minLength={2} maxLength={200} name="title" />
          </Field>
          <Field label="Тема курса">
            <select name="section_id" defaultValue={section}>
              <option value={0}>Вне тем</option>
              {data?.sections.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.title}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Условия задания">
            <textarea name="description" rows={5} required />
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Срок сдачи">
              <input required type="datetime-local" name="due_at" />
            </Field>
            <Field label="Максимальный балл">
              <input
                required
                type="number"
                min={1}
                max={1000}
                defaultValue={100}
                name="max_score"
              />
            </Field>
          </div>
        </FormModal>
      )}
      {modal === "announcement" && (
        <FormModal
          title="Объявление"
          close={() => setModal("")}
          submit={async (f) => {
            await api("/courses/" + id + "/announcements", "POST", {
              title: value(f, "title"),
              body: value(f, "body"),
            });
            await saved();
          }}
        >
          <Field label="Заголовок">
            <input required minLength={2} maxLength={200} name="title" />
          </Field>
          <Field label="Текст">
            <textarea
              required
              minLength={2}
              maxLength={20000}
              name="body"
              rows={5}
            />
          </Field>
        </FormModal>
      )}
      {modal === "quiz" && (
        <QuizEditor
          courseId={id}
          sections={data?.sections || []}
          sectionId={section}
          close={() => setModal("")}
          saved={saved}
        />
      )}
    </div>
  );
}
function AssignmentModal({
  assignment,
  submission,
  close,
  saved,
}: {
  assignment: Assignment;
  submission?: Submission;
  close: () => void;
  saved: () => Promise<void>;
}) {
  return submission?.score != null ? (
    <Modal title={assignment.title} close={close}>
      <span className="badge green">
        {submission.score} / {assignment.max_score} баллов
      </span>
      <p className="mt-4 whitespace-pre-wrap">
        {submission.feedback || "Работа проверена"}
      </p>
      <div className="submitted-work mt-5">{submission.content}</div>
    </Modal>
  ) : (
    <FormModal
      title={assignment.title}
      close={close}
      submit={async (f) => {
        await api("/assignments/" + assignment.id + "/submit", "POST", {
          content: value(f, "content"),
        });
        await saved();
      }}
    >
      <p className="muted text-sm whitespace-pre-wrap">
        {assignment.description}
      </p>
      <p className="badge orange">
        Срок сдачи: {date(assignment.due_at)}
        {new Date(assignment.due_at) < new Date()
          ? " · работа будет сдана с опозданием"
          : ""}
      </p>
      <Field label="Ваш ответ или ссылка на работу">
        <textarea
          name="content"
          required
          minLength={3}
          maxLength={50000}
          rows={8}
          defaultValue={submission?.content}
        />
      </Field>
    </FormModal>
  );
}
function QuizModal({
  quiz,
  close,
  saved,
}: {
  quiz: Quiz;
  close: () => void;
  saved: () => Promise<void>;
}) {
  const [answers, setAnswers] = useState<number[]>(
      Array(quiz.questions.length).fill(-1),
    ),
    [score, setScore] = useState<{ score: number; total: number }>(),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  return (
    <Modal title={quiz.title} close={close}>
      {score ? (
        <div className="quiz-result">
          <span className="result-circle">
            <GraduationCap size={40} />
          </span>
          <h2>{Math.round((score.score / score.total) * 100)}%</h2>
          <p>
            Правильных ответов: {score.score} из {score.total}
          </p>
          <button className="btn primary mt-6" onClick={close}>
            Вернуться к курсу
          </button>
        </div>
      ) : (
        <form
          className="space-y-7"
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            try {
              setScore(
                await api("/quizzes/" + quiz.id + "/attempts", "POST", {
                  answers,
                }),
              );
              await saved();
            } catch (err) {
              setError((err as Error).message);
            } finally {
              setBusy(false);
            }
          }}
        >
          <p className="muted text-sm">
            Выберите один ответ на каждый вопрос. Доступно 3 попытки.
          </p>
          {quiz.questions.map((q, i) => (
            <fieldset key={q.id}>
              <legend className="font-semibold text-sm mb-3">
                {i + 1}. {q.prompt}
              </legend>
              <div className="space-y-2">
                {q.options.map((o, j) => (
                  <label
                    className={
                      "quiz-option " + (answers[i] === j ? "selected" : "")
                    }
                    key={j}
                  >
                    <input
                      type="radio"
                      name={"q" + q.id}
                      required
                      checked={answers[i] === j}
                      onChange={() =>
                        setAnswers((a) => a.map((v, k) => (k === i ? j : v)))
                      }
                    />
                    {o}
                  </label>
                ))}
              </div>
            </fieldset>
          ))}
          {error && <p className="error">{error}</p>}
          <Submit busy={busy}>
            Завершить тест <Check size={16} />
          </Submit>
        </form>
      )}
    </Modal>
  );
}
function QuizEditor({
  courseId,
  sections,
  sectionId,
  close,
  saved,
}: {
  courseId: number;
  sections: Section[];
  sectionId: number;
  close: () => void;
  saved: () => Promise<void>;
}) {
  const [questions, setQuestions] = useState([
    { prompt: "", options: ["", "", ""], correct_index: 0 },
  ]);
  return (
    <FormModal
      title="Новый тест"
      close={close}
      submit={async (f) => {
        await api("/courses/" + courseId + "/quizzes", "POST", {
          title: value(f, "title"),
          questions,
          section_id: Number(f.get("section_id") || 0),
        });
        await saved();
      }}
    >
      <Field label="Название теста">
        <input required minLength={2} maxLength={200} name="title" />
      </Field>
      <Field label="Тема курса">
        <select name="section_id" defaultValue={sectionId}>
          <option value={0}>Вне тем</option>
          {sections.map((s) => (
            <option key={s.id} value={s.id}>
              {s.title}
            </option>
          ))}
        </select>
      </Field>
      {questions.map((q, i) => (
        <div className="quiz-edit" key={i}>
          <Field label={"Вопрос " + (i + 1)}>
            <input
              required
              minLength={2}
              maxLength={2000}
              value={q.prompt}
              onChange={(e) =>
                setQuestions((v) =>
                  v.map((x, j) =>
                    j === i ? { ...x, prompt: e.target.value } : x,
                  ),
                )
              }
            />
          </Field>
          <span className="muted text-xs block my-3">
            Отметьте правильный ответ
          </span>
          {q.options.map((o, k) => (
            <label className="flex gap-3 items-center mt-2" key={k}>
              <input
                aria-label={"Правильный ответ " + (k + 1)}
                type="radio"
                name={"correct" + i}
                checked={q.correct_index === k}
                onChange={() =>
                  setQuestions((v) =>
                    v.map((x, j) => (j === i ? { ...x, correct_index: k } : x)),
                  )
                }
              />
              <input
                aria-label={"Вариант " + (k + 1)}
                required
                maxLength={1000}
                placeholder={"Вариант " + (k + 1)}
                value={o}
                onChange={(e) =>
                  setQuestions((v) =>
                    v.map((x, j) =>
                      j === i
                        ? {
                            ...x,
                            options: x.options.map((op, n) =>
                              n === k ? e.target.value : op,
                            ),
                          }
                        : x,
                    ),
                  )
                }
              />
            </label>
          ))}
          {questions.length > 1 && (
            <button
              type="button"
              className="text-button mt-3"
              onClick={() => setQuestions((v) => v.filter((_, j) => j !== i))}
            >
              Удалить вопрос
            </button>
          )}
        </div>
      ))}
      <button
        type="button"
        className="btn"
        disabled={questions.length >= 50}
        onClick={() =>
          setQuestions((v) => [
            ...v,
            { prompt: "", options: ["", "", ""], correct_index: 0 },
          ])
        }
      >
        <Plus size={16} /> Ещё вопрос
      </button>
    </FormModal>
  );
}

function TopicPlacement({
  kind,
  itemId,
  sectionId,
  sections,
  saved,
  notify,
}: {
  kind: string;
  itemId: number;
  sectionId: number;
  sections: Section[];
  saved: () => Promise<void>;
  notify: (s: string) => void;
}) {
  const [busy, setBusy] = useState(false);
  return (
    <div className="mb-4 max-w-xs">
      <Field label="Разместить в теме">
        <select
          value={sectionId || 0}
          disabled={busy}
          onChange={(e) => {
            setBusy(true);
            void api("/" + kind + "/" + itemId + "/topic", "PATCH", {
              section_id: Number(e.target.value),
            })
              .then(saved)
              .catch((e) => notify(e.message))
              .finally(() => setBusy(false));
          }}
        >
          <option value={0}>Вне тем</option>
          {sections.map((s) => (
            <option key={s.id} value={s.id}>
              {s.title}
            </option>
          ))}
        </select>
      </Field>
    </div>
  );
}
