import { useEffect, useState } from "react";
import {
  Plus,
  Users,
  Trash2,
  Link,
  Upload,
  Library,
  FileText,
  ClipboardCheck,
  ExternalLink,
  CheckCircle2,
  AlertTriangle,
  Search,
  ChevronDown,
  BookOpen,
  LoaderCircle,
} from "lucide-react";
import {
  api,
  type User,
  type Course,
  type Cohort,
  type CourseCohort,
  type BankItem,
  type BankQuestion,
  type ImportReport,
  type Section,
  kinds,
  activityKinds,
  initials,
} from "./types";
import { Empty, Field, FormModal, Modal, Submit } from "./components";
const val = (f: FormData, k: string) => String(f.get(k) || "");
export function CohortsView({
  user,
  notify,
}: {
  user: User;
  notify: (s: string) => void;
}) {
  const [groups, setGroups] = useState<Cohort[]>([]),
    [users, setUsers] = useState<User[]>([]),
    [create, setCreate] = useState(false),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [selected, setSelected] = useState(0);
  const load = async () => {
    setGroups(await api<Cohort[]>("/cohorts"));
    if (user.role === "admin") setUsers(await api<User[]>("/users"));
  };
  useEffect(() => {
    void load().catch((e) => setError(e.message));
  }, []);
  const mutate = async (fn: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await fn();
      await load();
      notify(
        "Состав группы обновлён. Запись на связанные курсы синхронизирована.",
      );
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const group = groups.find((g) => g.id === selected);
  return (
    <>
      <div className="page-heading">
        <div>
          <span className="eyebrow">УЧИТЬСЯ ВМЕСТЕ</span>
          <h1>Глобальные группы</h1>
          <p>Объединяйте пользователей и записывайте целые группы на курсы.</p>
        </div>
        {user.role === "admin" && (
          <button className="btn primary" onClick={() => setCreate(true)}>
            <Plus size={17} /> Создать группу
          </button>
        )}
      </div>
      {error && <p className="error mb-5">{error}</p>}
      <div className="notice mb-6">
        <Users size={20} />
        <p>
          Группы действуют на всей платформе. Привяжите группу к курсу в его
          настройках — новые участники будут записываться автоматически. Ручная
          запись сохраняется после удаления из группы.
        </p>
      </div>
      <div className="course-grid">
        {groups.map((g) => (
          <button
            className="panel group-card"
            key={g.id}
            onClick={() => setSelected(g.id)}
          >
            <span className="activity-icon violet">
              <Users size={24} />
            </span>
            <h2>{g.name}</h2>
            <p className="muted text-sm">
              {g.description || "Глобальная группа пользователей"}
            </p>
            <span className="badge mt-5">Участников: {g.count}</span>
            {user.role === "admin" && (
              <span className="text-button mt-5">Управление участниками →</span>
            )}
          </button>
        ))}
      </div>
      {!groups.length && (
        <Empty title="Создайте первую группу">
          Например, «Дизайн · поток 2026» или «Новые сотрудники».
        </Empty>
      )}
      {create && (
        <FormModal
          title="Новая глобальная группа"
          close={() => setCreate(false)}
          submit={async (f) => {
            await api("/cohorts", "POST", {
              name: val(f, "name"),
              description: val(f, "description"),
            });
            await load();
            notify("Группа создана");
          }}
        >
          <Field label="Название">
            <input required minLength={2} maxLength={200} name="name" />
          </Field>
          <Field label="Описание">
            <textarea name="description" rows={3} maxLength={5000} />
          </Field>
        </FormModal>
      )}
      {group && (
        <Modal title={group.name} close={() => setSelected(0)}>
          <p className="muted mb-5">{group.description}</p>
          {user.role === "admin" ? (
            <>
              <form
                className="flex gap-3 items-end mb-6"
                onSubmit={(e) => {
                  e.preventDefault();
                  const f = new FormData(e.currentTarget);
                  void mutate(() =>
                    api("/cohorts/" + group.id + "/members", "POST", {
                      user_id: Number(f.get("user")),
                    }),
                  );
                }}
              >
                <div className="flex-1">
                  <Field label="Добавить участника">
                    <select name="user" required defaultValue="">
                      <option value="" disabled>
                        Выберите пользователя
                      </option>
                      {users
                        .filter(
                          (u) =>
                            !group.members?.some((m) => m.user_id === u.id),
                        )
                        .map((u) => (
                          <option value={u.id} key={u.id}>
                            {u.name} · {u.email}
                          </option>
                        ))}
                    </select>
                  </Field>
                </div>
                <Submit busy={busy}>
                  <Plus size={16} /> Добавить
                </Submit>
              </form>
              {group.members?.map((m) => (
                <div className="review-row" key={m.id}>
                  <span className="avatar">{initials(m.user.name)}</span>
                  <div className="flex-1">
                    <b>{m.user.name}</b>
                    <p className="muted text-xs mt-1">{m.user.email}</p>
                  </div>
                  <button
                    className="icon-btn"
                    aria-label={"Удалить из группы " + m.user.name}
                    disabled={busy}
                    onClick={() =>
                      void mutate(() =>
                        api(
                          "/cohorts/" + group.id + "/members/" + m.user_id,
                          "DELETE",
                        ),
                      )
                    }
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              ))}
              {!group.members?.length && (
                <Empty title="В группе пока нет участников" />
              )}
            </>
          ) : (
            <p className="notice">
              Участников: {group.count}. Составом глобальных групп управляет
              администратор.
            </p>
          )}
        </Modal>
      )}
    </>
  );
}
export function CourseCohortsModal({
  course,
  close,
  refresh,
}: {
  course: Course;
  close: () => void;
  refresh: () => Promise<void>;
}) {
  const [groups, setGroups] = useState<Cohort[]>([]),
    [links, setLinks] = useState<CourseCohort[]>([]),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const load = async () => {
    const [g, l] = await Promise.all([
      api<Cohort[]>("/cohorts"),
      api<CourseCohort[]>("/courses/" + course.id + "/cohorts"),
    ]);
    setGroups(g);
    setLinks(l);
  };
  useEffect(() => {
    void load().catch((e) => setError(e.message));
  }, [course.id]);
  const change = async (fn: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await fn();
      await load();
      await refresh();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title="Запись через глобальные группы" close={close}>
      <p className="muted text-sm mb-5">
        Участники привязанных групп автоматически получают доступ к курсу. При
        изменении группы запись синхронизируется.
      </p>
      {error && <p className="error mb-4">{error}</p>}
      <form
        className="flex gap-3 items-end mb-5"
        onSubmit={(e) => {
          e.preventDefault();
          const f = new FormData(e.currentTarget);
          void change(() =>
            api("/courses/" + course.id + "/cohorts", "POST", {
              cohort_id: Number(f.get("cohort")),
            }),
          );
        }}
      >
        <div className="flex-1">
          <Field label="Глобальная группа">
            <select name="cohort" required defaultValue="">
              <option value="" disabled>
                Выберите группу
              </option>
              {groups
                .filter((g) => !links.some((l) => l.cohort_id === g.id))
                .map((g) => (
                  <option value={g.id} key={g.id}>
                    {g.name} · {g.count} участников
                  </option>
                ))}
            </select>
          </Field>
        </div>
        <Submit busy={busy}>
          <Link size={16} /> Записать
        </Submit>
      </form>
      {links.map((l) => (
        <div className="review-row" key={l.id}>
          <Users size={19} />
          <div className="flex-1">
            <b>{l.cohort.name}</b>
            <p className="muted text-xs mt-1">Синхронизация включена</p>
          </div>
          <button
            className="icon-btn"
            aria-label="Отвязать группу"
            disabled={busy}
            onClick={() =>
              void change(() =>
                api(
                  "/courses/" + course.id + "/cohorts/" + l.cohort_id,
                  "DELETE",
                ),
              )
            }
          >
            <Trash2 size={16} />
          </button>
        </div>
      ))}
      {!links.length && <Empty title="К курсу пока не привязаны группы" />}
    </Modal>
  );
}
export function BankView({
  user,
  courses,
  notify,
  refresh,
  chooseCourse,
  close,
}: {
  user: User;
  courses: Course[];
  notify: (s: string) => void;
  refresh: () => Promise<void>;
  chooseCourse?: number;
  close?: () => void;
}) {
  const [items, setItems] = useState<BankItem[]>([]),
    [filter, setFilter] = useState("all"),
    [search, setSearch] = useState(""),
    [create, setCreate] = useState(false),
    [use, setUse] = useState<BankItem>(),
    [preview, setPreview] = useState<BankItem>(),
    [error, setError] = useState(""),
    [remove, setRemove] = useState<BankItem>(),
    [busy, setBusy] = useState(false);
  const load = async () => setItems(await api<BankItem[]>("/bank"));
  useEffect(() => {
    void load().catch((e) => setError(e.message));
  }, []);
  const list = items.filter(
    (i) =>
      (filter === "all" || i.kind === filter) &&
      (i.title + " " + i.category).toLowerCase().includes(search.toLowerCase()),
  );
  const content = (
    <>
      <div className="flex flex-wrap gap-3 justify-between mb-6">
        <div>
          <h2>
            {chooseCourse
              ? "Добавить элемент из банка"
              : "Банк контента и тестов"}
          </h2>
          <p className="muted text-sm mt-2">
            Общие материалы, задания и тесты. Изменения копии в курсе не
            затрагивают оригинал.
          </p>
        </div>
        <button className="btn primary" onClick={() => setCreate(true)}>
          <Plus size={16} /> Новый элемент
        </button>
      </div>
      {error && <p className="error mb-5">{error}</p>}
      <div className="flex flex-wrap justify-between gap-4 mb-5">
        <div className="category-tabs">
          {[
            { id: "all", title: "Все элементы" },
            ...Object.entries(activityKinds)
              .filter(([id]) => id !== "file" && id !== "folder")
              .map(([id, title]) => ({ id, title })),
          ].map((k) => (
            <button
              key={k.id}
              className={filter === k.id ? "active" : ""}
              onClick={() => setFilter(k.id)}
            >
              {k.title}
            </button>
          ))}
        </div>
        <label className="flex gap-2 items-center">
          <Search size={17} className="muted" />
          <input
            aria-label="Поиск по банку"
            placeholder="Найти материал"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </label>
      </div>
      <div className={chooseCourse ? "space-y-3" : "course-grid"}>
        {list.map((i) => (
          <article className="panel bank-card" key={i.id}>
            <div className="flex justify-between items-start">
              <span
                className={
                  "activity-icon " +
                  (i.kind === "quiz"
                    ? "violet"
                    : i.kind === "assignment"
                      ? "orange"
                      : "blue")
                }
              >
                {i.kind === "quiz" ? (
                  <ClipboardCheck size={23} />
                ) : i.kind === "url" ? (
                  <ExternalLink size={23} />
                ) : (
                  <FileText size={23} />
                )}
              </span>
              <span className="badge">
                {activityKinds[i.kind] || kinds[i.kind]}
              </span>
            </div>
            <h3 className="mt-5">{i.title}</h3>
            <p className="muted text-xs mt-2">
              {i.category || "Без категории"} · {i.author?.name || "Автор"}
            </p>
            {i.kind === "quiz" && (
              <p className="muted text-xs mt-2">
                Вопросов: {i.questions.length}
              </p>
            )}
            <div className="flex flex-wrap gap-2 mt-5">
              <button className="btn" onClick={() => setPreview(i)}>
                Просмотр
              </button>
              <button className="btn primary" onClick={() => setUse(i)}>
                <Plus size={14} /> В курс
              </button>
              {(user.role === "admin" || i.author_id === user.id) && (
                <button
                  className="icon-btn"
                  aria-label="Удалить из банка"
                  onClick={() => setRemove(i)}
                >
                  <Trash2 size={16} />
                </button>
              )}
            </div>
          </article>
        ))}
      </div>
      {!list.length && (
        <Empty title="В банке пока нет подходящих элементов">
          Создайте материал или импортируйте поддерживаемые элементы из бэкапа
          Moodle.
        </Empty>
      )}
    </>
  );
  return (
    <>
      {chooseCourse && close ? (
        <Modal title="Банк контента" close={close}>
          {content}
        </Modal>
      ) : (
        <>
          <div className="page-heading">
            <div>
              <span className="eyebrow">
                СОЗДАВАЙТЕ ОДНАЖДЫ · ИСПОЛЬЗУЙТЕ СНОВА
              </span>
              <h1>Банк контента</h1>
            </div>
          </div>
          {content}
        </>
      )}
      {create && (
        <BankEditor
          close={() => setCreate(false)}
          saved={async () => {
            await load();
            notify("Элемент добавлен в банк");
          }}
        />
      )}
      {use && (
        <BankCopy
          item={use}
          courses={courses.filter(
            (c) => user.role === "admin" || c.teacher_id === user.id,
          )}
          defaultCourse={chooseCourse}
          close={() => setUse(undefined)}
          saved={async () => {
            await refresh();
            notify("Элемент скопирован в курс");
            close?.();
          }}
        />
      )}
      {preview && (
        <Modal title={preview.title} close={() => setPreview(undefined)}>
          <span className="badge mb-5">
            {activityKinds[preview.kind] || kinds[preview.kind]}
          </span>
          <article className="lesson-content">{preview.body}</article>
          {preview.url && (
            <a
              className="text-button mt-4"
              href={preview.url}
              target="_blank"
              rel="noreferrer"
            >
              Открыть ссылку <ExternalLink size={15} />
            </a>
          )}
          {preview.data?.chapters?.map((c, i) => (
            <section className="submitted-work mt-4" key={i}>
              <h3>{c.title}</h3>
              <p className="mt-3 whitespace-pre-wrap">{c.content}</p>
            </section>
          ))}
          {preview.data?.entries?.map((v, i) => (
            <section className="submitted-work mt-4" key={i}>
              <h3>{v.term}</h3>
              <p className="mt-3 whitespace-pre-wrap">{v.definition}</p>
            </section>
          ))}
          {preview.data?.options?.map((o, i) => (
            <p className="quiz-option mt-3" key={i}>
              {o}
            </p>
          ))}
          {preview.kind === "quiz" &&
            preview.questions.map((q, i) => (
              <div key={i} className="quiz-edit mt-4">
                <h3>
                  {i + 1}. {q.prompt}
                </h3>
                {q.options.map((o, j) => (
                  <p
                    className={
                      "mt-2 text-sm " +
                      (q.correct_index === j
                        ? "text-green-600 font-semibold"
                        : "muted")
                    }
                    key={j}
                  >
                    {q.correct_index === j ? "✓ " : ""}
                    {o}
                  </p>
                ))}
              </div>
            ))}
        </Modal>
      )}
      {remove && (
        <Modal title="Удалить из банка?" close={() => setRemove(undefined)}>
          <p className="muted mb-6">
            «{remove.title}» будет удалён из общего банка. Копии в курсах
            сохранятся.
          </p>
          <button
            className="btn primary"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              try {
                await api("/bank/" + remove.id, "DELETE");
                setRemove(undefined);
                await load();
                notify("Элемент удалён");
              } catch (e) {
                notify((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            Удалить элемент
          </button>
        </Modal>
      )}
    </>
  );
}
function BankEditor({
  close,
  saved,
}: {
  close: () => void;
  saved: () => Promise<void>;
}) {
  const [kind, setKind] = useState("page"),
    [questions, setQuestions] = useState<BankQuestion[]>([
      { prompt: "", options: ["", "", ""], correct_index: 0 },
    ]);
  return (
    <FormModal
      title="Новый элемент банка"
      close={close}
      submit={async (f) => {
        await api("/bank", "POST", {
          title: val(f, "title"),
          category: val(f, "category"),
          kind,
          body: val(f, "body"),
          url: val(f, "url"),
          duration: Number(f.get("duration") || 15),
          max_score: Number(f.get("max_score") || 100),
          questions: kind === "quiz" ? questions : [],
        });
        await saved();
      }}
    >
      <div className="grid grid-cols-2 gap-4">
        <Field label="Тип элемента">
          <select value={kind} onChange={(e) => setKind(e.target.value)}>
            {Object.entries(kinds).map(([k, v]) => (
              <option value={k} key={k}>
                {v}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Категория">
          <input
            name="category"
            maxLength={100}
            placeholder="Например, Разработка"
          />
        </Field>
      </div>
      <Field label="Название">
        <input name="title" required minLength={2} maxLength={200} />
      </Field>
      {kind !== "quiz" && (
        <Field label={kind === "assignment" ? "Условия задания" : "Содержание"}>
          <textarea name="body" rows={6} maxLength={100000} />
        </Field>
      )}
      {kind === "url" && (
        <Field label="Адрес ссылки">
          <input name="url" type="url" required placeholder="https://…" />
        </Field>
      )}
      {kind === "assignment" && (
        <Field label="Максимальный балл">
          <input
            name="max_score"
            required
            type="number"
            min={1}
            max={1000}
            defaultValue={100}
          />
        </Field>
      )}
      {kind === "quiz" && (
        <>
          <p className="notice">
            Один правильный ответ на вопрос. Один вопрос можно сохранить как
            отдельный элемент банка.
          </p>
          {questions.map((q, i) => (
            <div key={i} className="quiz-edit">
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
                Отметьте правильный вариант
              </span>
              {q.options.map((o, k) => (
                <label className="flex gap-3 items-center mt-2" key={k}>
                  <input
                    type="radio"
                    name={"correct" + i}
                    aria-label={"Правильный вариант " + (k + 1)}
                    checked={q.correct_index === k}
                    onChange={() =>
                      setQuestions((v) =>
                        v.map((x, j) =>
                          j === i ? { ...x, correct_index: k } : x,
                        ),
                      )
                    }
                  />
                  <input
                    aria-label={"Вариант ответа " + (k + 1)}
                    required
                    maxLength={1000}
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
                  onClick={() =>
                    setQuestions((v) => v.filter((_, j) => j !== i))
                  }
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
            <Plus size={16} /> Добавить вопрос
          </button>
        </>
      )}
    </FormModal>
  );
}
function BankCopy({
  item,
  courses,
  defaultCourse,
  close,
  saved,
}: {
  item: BankItem;
  courses: Course[];
  defaultCourse?: number;
  close: () => void;
  saved: () => Promise<void>;
}) {
  const [courseId, setCourse] = useState(defaultCourse || courses[0]?.id || 0),
    [sections, setSections] = useState<Section[]>([]);
  useEffect(() => {
    if (courseId)
      api<{ sections: Section[] }>("/courses/" + courseId)
        .then((d) => setSections(d.sections))
        .catch(() => setSections([]));
  }, [courseId]);
  return (
    <FormModal
      title={"Добавить в курс: " + item.title}
      close={close}
      submit={async (f) => {
        await api("/courses/" + courseId + "/bank/" + item.id, "POST", {
          section_id: Number(f.get("section") || 0),
          ...(item.kind === "assignment" && f.get("due")
            ? { due_at: new Date(val(f, "due")).toISOString() }
            : {}),
        });
        await saved();
      }}
    >
      <Field label="Курс">
        <select
          required
          value={courseId || ""}
          onChange={(e) => setCourse(Number(e.target.value))}
        >
          <option value="" disabled>
            Выберите курс
          </option>
          {courses.map((c) => (
            <option key={c.id} value={c.id}>
              {c.title}
            </option>
          ))}
        </select>
      </Field>
      {item.kind !== "assignment" && item.kind !== "quiz" && (
        <Field label="Модуль">
          <select name="section" key={courseId}>
            <option value={0}>Создать модуль «Материалы из банка»</option>
            {sections.map((s) => (
              <option key={s.id} value={s.id}>
                {s.title}
              </option>
            ))}
          </select>
        </Field>
      )}
      {item.kind === "assignment" && (
        <Field label="Срок сдачи">
          <input type="datetime-local" name="due" required />
        </Field>
      )}
      <p className="muted text-sm">Будет создана независимая копия элемента.</p>
    </FormModal>
  );
}
export function MoodleImport({
  course,
  close,
  saved,
}: {
  course: Course;
  close: () => void;
  saved: () => Promise<void>;
}) {
  const [file, setFile] = useState<File>(),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [report, setReport] = useState<ImportReport>();
  return (
    <Modal
      title="Импорт бэкапа Moodle"
      close={() => {
        if (!busy) close();
      }}
    >
      {report ? (
        <>
          <div className="notice success mb-5">
            <CheckCircle2 size={24} />
            <div>
              <b>Импорт завершён</b>
              <p className="mt-1">
                {report.source || "Курс Moodle"} → {course.title}
              </p>
            </div>
          </div>
          <div className="import-stats">
            <span>
              <b>{report.sections}</b> модулей
            </span>
            <span>
              <b>{report.activities}</b> элементов
            </span>
            <span>
              <b>{report.question_bank}</b> вопросов в банке
            </span>
          </div>
          <div className="flex gap-2 flex-wrap my-5">
            {Object.entries(report.counts).map(([k, n]) => (
              <span className="badge" key={k}>
                {kinds[k] || k}: {n}
              </span>
            ))}
          </div>
          <h3 className="flex gap-2 items-center mb-3">
            <AlertTriangle size={17} /> Отчёт о совместимости
          </h3>
          <ul className="import-warnings">
            {report.warnings.map((w, i) => (
              <li key={i}>{w}</li>
            ))}
          </ul>
          <button className="btn primary mt-6" onClick={close}>
            Вернуться к курсу
          </button>
        </>
      ) : (
        <form
          className="space-y-5"
          onSubmit={async (e) => {
            e.preventDefault();
            if (!file) return;
            setBusy(true);
            setError("");
            try {
              const body = new FormData();
              body.set("backup", file);
              const response = await fetch(
                "/api/courses/" + course.id + "/import/moodle",
                { method: "POST", body, credentials: "same-origin" },
              );
              const result = await response.json();
              if (!response.ok)
                throw new Error(result.error || "Ошибка импорта");
              setReport(result);
              await saved();
            } catch (err) {
              setError((err as Error).message);
            } finally {
              setBusy(false);
            }
          }}
        >
          <p className="muted text-sm leading-relaxed">
            Добавьте материалы из резервной копии Moodle в «{course.title}».
            Существующее содержимое курса сохраняется. Повторный импорт создаст
            новые копии.
          </p>
          <div className="notice">
            <FileText size={22} />
            <p>
              Поддерживаются страницы, книги, текстовые блоки, ссылки, задания и
              тесты с одним правильным ответом (multichoice / truefalse). HTML
              преобразуется в текст. Файлы, пользователи, оценки и плагины не
              переносятся.
            </p>
          </div>
          <Field label="Резервная копия .mbz (TAR/GZIP или ZIP), до 32 МБ">
            <input
              type="file"
              required
              accept=".mbz,.zip,.gz,.tgz,.tar"
              disabled={busy}
              onChange={(e) => setFile(e.target.files?.[0])}
            />
          </Field>
          <p className="muted text-xs">
            Максимум 64 МБ после распаковки. Зашифрованные архивы и Zstandard не
            поддерживаются. Проверяйте отчёт: совместимость зависит от структуры
            бэкапа.
          </p>
          {error && (
            <p className="error" role="alert">
              {error}
            </p>
          )}
          <button
            type="submit"
            className="btn primary"
            disabled={busy || !file || file.size > 32 * 1024 * 1024}
          >
            {busy ? (
              <LoaderCircle size={17} className="animate-spin" />
            ) : (
              <Upload size={17} />
            )}{" "}
            {busy ? "Импортируем…" : "Импортировать в курс"}
          </button>
        </form>
      )}
    </Modal>
  );
}
