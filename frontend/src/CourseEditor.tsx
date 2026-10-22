import { useEffect, useState } from "react";
import {
  AlignLeft,
  BookOpen,
  Check,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  ClipboardCheck,
  Download,
  ExternalLink,
  FileText,
  FolderOpen,
  HelpCircle,
  Link,
  MessageSquare,
  Plus,
  Send,
  Trash2,
  Upload,
  Video,
  Vote,
  Library,
  LoaderCircle,
} from "lucide-react";
import {
  api,
  type Course,
  type Section,
  type Lesson,
  type ActivityData,
  type Resource,
  type User,
  type Discussion,
  activityKinds,
  initials,
  date,
} from "./types";
import { Empty, Field, FormModal, Modal, Submit } from "./components";
const text = (f: FormData, k: string) => String(f.get(k) || "");
export const icons: Record<string, typeof FileText> = {
  page: FileText,
  label: AlignLeft,
  url: Link,
  video: Video,
  book: BookOpen,
  file: Download,
  folder: FolderOpen,
  forum: MessageSquare,
  glossary: Library,
  choice: Vote,
  assignment: FileText,
  quiz: ClipboardCheck,
};
export function ElementPicker({
  close,
  select,
}: {
  close: () => void;
  select: (kind: string) => void;
}) {
  const descriptions: Record<string, string> = {
    page: "Текстовый материал для изучения",
    label: "Описание или пояснение прямо в теме",
    url: "Ссылка на внешний ресурс",
    video: "Видео по ссылке или видеоплеер",
    book: "Материал с главами и навигацией",
    file: "Документ для скачивания",
    folder: "Несколько учебных файлов",
    forum: "Обсуждение отдельной темы",
    glossary: "Словарь терминов и определений",
    choice: "Один вопрос с голосованием",
    assignment: "Работа для проверки преподавателем",
    quiz: "Проверка знаний с оценкой",
  };
  return (
    <Modal title="Добавить элемент курса" close={close}>
      <p className="muted text-sm mb-5">
        Выберите ресурс или интерактивное задание для этой темы.
      </p>
      <div className="element-picker">
        {Object.entries(activityKinds).map(([kind, label]) => {
          const Icon = icons[kind];
          return (
            <button
              key={kind}
              className="element-option"
              onClick={() => select(kind)}
            >
              <span
                className={
                  "activity-icon " +
                  (["quiz", "choice"].includes(kind)
                    ? "violet"
                    : ["assignment", "forum"].includes(kind)
                      ? "orange"
                      : "blue")
                }
              >
                <Icon size={23} />
              </span>
              <span>
                <b>{label}</b>
                <small>{descriptions[kind]}</small>
              </span>
            </button>
          );
        })}
      </div>
    </Modal>
  );
}
export function TopicEditor({
  topic,
  course,
  close,
  saved,
}: {
  topic?: Section;
  course: Course;
  close: () => void;
  saved: () => Promise<void>;
}) {
  return (
    <FormModal
      title={topic ? "Настройки темы" : "Новая тема курса"}
      close={close}
      submit={async (f) => {
        await api(
          topic
            ? "/sections/" + topic.id
            : "/courses/" + course.id + "/sections",
          topic ? "PATCH" : "POST",
          {
            title: text(f, "title"),
            summary: text(f, "summary"),
            hidden: f.get("hidden") === "on",
          },
        );
        await saved();
      }}
    >
      <Field label="Название темы">
        <input
          name="title"
          required
          minLength={2}
          maxLength={200}
          defaultValue={topic?.title}
        />
      </Field>
      <Field label="Описание темы">
        <textarea
          name="summary"
          rows={5}
          maxLength={10000}
          defaultValue={topic?.summary}
          placeholder="Что изучим в этой теме, цели и рекомендации"
        />
      </Field>
      <label className="flex items-center gap-3 text-sm">
        <input type="checkbox" name="hidden" defaultChecked={topic?.hidden} />{" "}
        Скрыть тему и её элементы от студентов
      </label>
      <p className="muted text-xs">
        Скрытая тема остаётся доступна преподавателю. Её элементы также нельзя
        открыть студенту по прямой ссылке.
      </p>
    </FormModal>
  );
}
export function ActivityEditor({
  kind,
  lesson,
  sectionId,
  sections,
  close,
  saved,
}: {
  kind: string;
  lesson?: Lesson;
  sectionId: number;
  sections: Section[];
  close: () => void;
  saved: () => Promise<void>;
}) {
  const [current, setCurrent] = useState(lesson),
    [data, setData] = useState<ActivityData>(lesson?.data || {}),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const chapters = data.chapters?.length
      ? data.chapters
      : [{ title: "", content: "" }],
    entries = data.entries?.length
      ? data.entries
      : [{ term: "", definition: "" }],
    options = data.options?.length ? data.options : ["", ""];
  const files = current?.files || [];
  const upload = async (list: FileList | null) => {
    if (!list || !current) return;
    setBusy(true);
    setError("");
    try {
      for (const file of Array.from(list)) {
        const body = new FormData();
        body.set("file", file);
        const response = await fetch("/api/lessons/" + current.id + "/files", {
          method: "POST",
          body,
          credentials: "same-origin",
        });
        const v = await response.json();
        if (!response.ok)
          throw new Error(v.error || "Не удалось загрузить файл");
        setCurrent((l) => (l ? { ...l, files: [...(l.files || []), v] } : l));
      }
      await saved();
    } catch (e) {
      setError((e as Error).message);
      await saved();
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal
      title={
        (lesson ? "Редактировать: " : "Новый элемент: ") +
        (activityKinds[kind] || "Страница")
      }
      close={() => {
        if (!busy) close();
      }}
    >
      <form
        className="space-y-4"
        onSubmit={async (e) => {
          e.preventDefault();
          const f = new FormData(e.currentTarget);
          setBusy(true);
          setError("");
          try {
            const result = await api<Lesson>(
              current
                ? "/lessons/" + current.id
                : "/sections/" + sectionId + "/lessons",
              current ? "PATCH" : "POST",
              {
                title: text(f, "title"),
                content: text(f, "content"),
                kind,
                url: text(f, "url"),
                duration: Number(f.get("duration") || 10),
                section_id: Number(f.get("section_id") || sectionId),
                data: {
                  chapters: kind === "book" ? chapters : [],
                  entries: kind === "glossary" ? entries : [],
                  options: kind === "choice" ? options : [],
                },
              },
            );
            setCurrent(result);
            await saved();
            if (kind !== "file" && kind !== "folder") close();
          } catch (e) {
            setError((e as Error).message);
          } finally {
            setBusy(false);
          }
        }}
      >
        <Field label="Название элемента">
          <input
            name="title"
            required
            minLength={2}
            maxLength={200}
            defaultValue={lesson?.title}
          />
        </Field>
        <Field
          label={
            kind === "choice"
              ? "Вопрос и пояснение"
              : kind === "forum"
                ? "Тема обсуждения и правила"
                : "Описание / текст материала"
          }
        >
          <textarea
            name="content"
            rows={kind === "page" ? 9 : 3}
            maxLength={100000}
            defaultValue={lesson?.content}
          />
        </Field>
        {(kind === "url" || kind === "video") && (
          <Field
            label={
              kind === "video"
                ? "Ссылка на YouTube, Vimeo или видеофайл"
                : "Адрес ресурса"
            }
          >
            <input
              name="url"
              type="url"
              required
              maxLength={4000}
              defaultValue={lesson?.url}
              placeholder="https://…"
            />
          </Field>
        )}
        {kind === "book" && (
          <div className="space-y-4">
            {chapters.map((c, i) => (
              <div key={i} className="quiz-edit space-y-3">
                <Field label={"Название главы " + (i + 1)}>
                  <input
                    required
                    minLength={2}
                    maxLength={200}
                    value={c.title}
                    onChange={(e) =>
                      setData((d) => ({
                        ...d,
                        chapters: chapters.map((v, j) =>
                          i === j ? { ...v, title: e.target.value } : v,
                        ),
                      }))
                    }
                  />
                </Field>
                <Field label="Текст главы">
                  <textarea
                    required
                    rows={6}
                    value={c.content}
                    onChange={(e) =>
                      setData((d) => ({
                        ...d,
                        chapters: chapters.map((v, j) =>
                          i === j ? { ...v, content: e.target.value } : v,
                        ),
                      }))
                    }
                  />
                </Field>
                {chapters.length > 1 && (
                  <button
                    type="button"
                    className="text-button"
                    onClick={() =>
                      setData((d) => ({
                        ...d,
                        chapters: chapters.filter((_, j) => j !== i),
                      }))
                    }
                  >
                    Удалить главу
                  </button>
                )}
              </div>
            ))}
            <button
              type="button"
              className="btn"
              disabled={chapters.length >= 100}
              onClick={() =>
                setData((d) => ({
                  ...d,
                  chapters: [...chapters, { title: "", content: "" }],
                }))
              }
            >
              <Plus size={16} /> Добавить главу
            </button>
          </div>
        )}
        {kind === "glossary" && (
          <div className="space-y-4">
            {entries.map((v, i) => (
              <div className="quiz-edit space-y-3" key={i}>
                <Field label={"Термин " + (i + 1)}>
                  <input
                    required
                    maxLength={200}
                    value={v.term}
                    onChange={(e) =>
                      setData((d) => ({
                        ...d,
                        entries: entries.map((x, j) =>
                          j === i ? { ...x, term: e.target.value } : x,
                        ),
                      }))
                    }
                  />
                </Field>
                <Field label="Определение">
                  <textarea
                    required
                    rows={3}
                    maxLength={10000}
                    value={v.definition}
                    onChange={(e) =>
                      setData((d) => ({
                        ...d,
                        entries: entries.map((x, j) =>
                          j === i ? { ...x, definition: e.target.value } : x,
                        ),
                      }))
                    }
                  />
                </Field>
                {entries.length > 1 && (
                  <button
                    type="button"
                    className="text-button"
                    onClick={() =>
                      setData((d) => ({
                        ...d,
                        entries: entries.filter((_, j) => j !== i),
                      }))
                    }
                  >
                    Удалить термин
                  </button>
                )}
              </div>
            ))}
            <button
              type="button"
              className="btn"
              disabled={entries.length >= 200}
              onClick={() =>
                setData((d) => ({
                  ...d,
                  entries: [...entries, { term: "", definition: "" }],
                }))
              }
            >
              <Plus size={16} /> Добавить термин
            </button>
          </div>
        )}
        {kind === "choice" && (
          <div className="space-y-3">
            <p className="notice">
              Студент выбирает один ответ. Результаты доступны ему после
              голосования; преподаватель видит их сразу. После первого голоса
              варианты менять нельзя.
            </p>
            {options.map((v, i) => (
              <div className="flex gap-2" key={i}>
                <Field label={"Вариант " + (i + 1)}>
                  <input
                    required
                    maxLength={200}
                    value={v}
                    onChange={(e) =>
                      setData((d) => ({
                        ...d,
                        options: options.map((x, j) =>
                          j === i ? e.target.value : x,
                        ),
                      }))
                    }
                  />
                </Field>
                {options.length > 2 && (
                  <button
                    type="button"
                    className="icon-btn"
                    aria-label="Удалить вариант"
                    onClick={() =>
                      setData((d) => ({
                        ...d,
                        options: options.filter((_, j) => j !== i),
                      }))
                    }
                  >
                    <Trash2 size={16} />
                  </button>
                )}
              </div>
            ))}
            <button
              type="button"
              className="btn"
              disabled={options.length >= 10}
              onClick={() =>
                setData((d) => ({ ...d, options: [...options, ""] }))
              }
            >
              <Plus size={16} /> Добавить вариант
            </button>
          </div>
        )}
        <div className="grid grid-cols-2 gap-4">
          {lesson && (
            <Field label="Тема курса">
              <select name="section_id" defaultValue={lesson.section_id}>
                {sections.map((s) => (
                  <option value={s.id} key={s.id}>
                    {s.title}
                  </option>
                ))}
              </select>
            </Field>
          )}
          <Field label="Время изучения, минут">
            <input
              name="duration"
              type="number"
              min={1}
              max={1000}
              defaultValue={lesson?.duration || 10}
              required
            />
          </Field>
        </div>
        {error && (
          <p className="error" role="alert">
            {error}
          </p>
        )}
        <div className="flex flex-wrap justify-end gap-3 pt-3">
          <button type="button" className="btn" onClick={close} disabled={busy}>
            Закрыть
          </button>
          <Submit busy={busy}>
            {!current && (kind === "file" || kind === "folder")
              ? "Сохранить и добавить файлы"
              : "Сохранить"}
          </Submit>
        </div>
      </form>
      {(kind === "file" || kind === "folder") && current && (
        <div className="file-manager">
          <h3 className="mb-4">
            {kind === "file" ? "Прикреплённый файл" : "Файлы папки"}
          </h3>
          {files.map((f) => (
            <div className="file-row" key={f.id}>
              <FileText size={18} />
              <a
                href={"/api/resources/" + f.id + "/download"}
                className="flex-1"
              >
                {f.name}
                <small>{(f.size / 1024).toFixed(1)} КБ</small>
              </a>
              <button
                className="icon-btn"
                aria-label={"Удалить файл " + f.name}
                disabled={busy}
                onClick={async () => {
                  if (!window.confirm("Удалить файл «" + f.name + "»?")) return;
                  setBusy(true);
                  try {
                    await api("/resources/" + f.id, "DELETE");
                    setCurrent((l) =>
                      l
                        ? {
                            ...l,
                            files: (l.files || []).filter((v) => v.id !== f.id),
                          }
                        : l,
                    );
                    await saved();
                  } catch (e) {
                    setError((e as Error).message);
                  } finally {
                    setBusy(false);
                  }
                }}
              >
                <Trash2 size={17} />
              </button>
            </div>
          ))}
          {((kind === "folder" && files.length < 20) || !files.length) && (
            <Field label="Загрузить файлы (до 20 МБ каждый)">
              <input
                type="file"
                multiple={kind === "folder"}
                disabled={busy}
                onChange={(e) => {
                  void upload(e.target.files);
                  e.target.value = "";
                }}
                accept=".pdf,.txt,.md,.csv,.docx,.xlsx,.pptx,.odt,.ods,.odp,.zip,.png,.jpg,.jpeg,.gif,.webp,.mp3,.mp4,.webm"
              />
            </Field>
          )}
          <p className="muted text-xs mt-3">
            Документы, архивы, изображения, аудио и видео. В папке — до 20
            файлов. Скачивание доступно только участникам курса.
          </p>
          {busy && (
            <p className="text-button mt-3">
              <LoaderCircle size={16} className="animate-spin" /> Обрабатываем
              файлы…
            </p>
          )}
        </div>
      )}
    </Modal>
  );
}
export function ActivityViewer({
  lesson,
  user,
  completed,
  close,
  complete,
}: {
  lesson: Lesson;
  user: User;
  completed: boolean;
  close: () => void;
  complete: () => Promise<void>;
}) {
  const kind = lesson.kind || "page";
  const [chapter, setChapter] = useState(0),
    [search, setSearch] = useState(""),
    [interaction, setInteraction] = useState<{
      vote?: number | null;
      counts?: number[];
      posts?: Discussion[];
    }>({}),
    [option, setOption] = useState(-1),
    [body, setBody] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const load = () =>
    api<typeof interaction>("/lessons/" + lesson.id + "/interaction").then(
      (v) => {
        setInteraction(v);
        if (v.vote != null) setOption(v.vote);
      },
    );
  useEffect(() => {
    if (kind === "forum" || kind === "choice")
      void load().catch((e) => setError(e.message));
  }, [lesson.id, kind]);
  const act = async (fn: () => Promise<unknown>) => {
    setBusy(true);
    setError("");
    try {
      await fn();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const chapters = lesson.data?.chapters || [],
    entries = lesson.data?.entries || [],
    options = lesson.data?.options || [],
    votes = (interaction.counts || []).reduce((sum, v) => sum + v, 0);
  return (
    <Modal title={lesson.title} close={close}>
      <span className="badge mb-5">{activityKinds[kind] || "Страница"}</span>
      {lesson.content && (
        <article className="lesson-content mb-5">{lesson.content}</article>
      )}
      {(kind === "url" || kind === "video") &&
        lesson.url &&
        /^https?:\/\//.test(lesson.url) && (
          <>
            {kind === "video" && <VideoPlayer url={lesson.url} />}
            <a
              className="btn mt-3"
              href={lesson.url}
              target="_blank"
              rel="noreferrer"
            >
              Открыть ресурс <ExternalLink size={16} />
            </a>
          </>
        )}
      {kind === "book" && chapters.length > 0 && (
        <div className="book-view">
          <div className="chapter-tabs">
            {chapters.map((c, i) => (
              <button
                className={chapter === i ? "active" : ""}
                key={i}
                onClick={() => setChapter(i)}
              >
                {i + 1}. {c.title}
              </button>
            ))}
          </div>
          <h3 className="my-5">{chapters[chapter]?.title}</h3>
          <article className="lesson-content">
            {chapters[chapter]?.content}
          </article>
          <div className="flex justify-between gap-3 mt-6">
            <button
              className="btn"
              disabled={chapter === 0}
              onClick={() => setChapter((i) => i - 1)}
            >
              <ChevronLeft size={16} /> Назад
            </button>
            <span className="muted text-sm self-center">
              {chapter + 1} / {chapters.length}
            </span>
            <button
              className="btn"
              disabled={chapter === chapters.length - 1}
              onClick={() => setChapter((i) => i + 1)}
            >
              Далее <ChevronRight size={16} />
            </button>
          </div>
        </div>
      )}
      {(kind === "file" || kind === "folder") && (
        <div>
          {(lesson.files || []).map((f) => (
            <a
              className="file-row"
              href={"/api/resources/" + f.id + "/download"}
              key={f.id}
            >
              <FileText size={22} />
              <span className="flex-1">
                {f.name}
                <small>{(f.size / 1024).toFixed(1)} КБ</small>
              </span>
              <Download size={18} />
            </a>
          ))}
          {!lesson.files?.length && <Empty title="Файлы ещё не добавлены" />}
        </div>
      )}
      {kind === "glossary" && (
        <div>
          <Field label="Найти термин">
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Введите термин или часть определения"
            />
          </Field>
          <dl className="glossary-list">
            {entries
              .filter((v) =>
                (v.term + " " + v.definition)
                  .toLowerCase()
                  .includes(search.toLowerCase()),
              )
              .sort((a, b) => a.term.localeCompare(b.term, "ru"))
              .map((v, i) => (
                <div key={i}>
                  <dt>{v.term}</dt>
                  <dd>{v.definition}</dd>
                </div>
              ))}
          </dl>
        </div>
      )}
      {kind === "choice" && (
        <form
          className="space-y-3"
          onSubmit={(e) => {
            e.preventDefault();
            void act(async () => {
              await api("/lessons/" + lesson.id + "/vote", "POST", {
                option_index: option,
              });
              await load();
            });
          }}
        >
          {options.map((o, i) => (
            <div key={i}>
              <label
                className={"quiz-option " + (option === i ? "selected" : "")}
              >
                <input
                  required
                  type="radio"
                  name="choice"
                  value={i}
                  checked={option === i}
                  disabled={user.role !== "student" || busy}
                  onChange={() => setOption(i)}
                />
                <span className="flex-1">{o}</span>
                {(interaction.vote != null || user.role !== "student") && (
                  <b>{interaction.counts?.[i] || 0}</b>
                )}
              </label>
              {(interaction.vote != null || user.role !== "student") && (
                <div className="progress mt-2">
                  <i
                    style={{
                      width:
                        (votes
                          ? ((interaction.counts?.[i] || 0) / votes) * 100
                          : 0) + "%",
                    }}
                  />
                </div>
              )}
            </div>
          ))}
          {user.role === "student" ? (
            <Submit busy={busy}>
              {interaction.vote != null ? "Изменить ответ" : "Проголосовать"}
            </Submit>
          ) : (
            <p className="muted text-sm">
              В опросе участвуют студенты. Всего голосов: {votes}.
            </p>
          )}
        </form>
      )}
      {kind === "forum" && (
        <div>
          <form
            className="space-y-3 mb-6"
            onSubmit={(e) => {
              e.preventDefault();
              void act(async () => {
                await api("/lessons/" + lesson.id + "/posts", "POST", { body });
                setBody("");
                await load();
              });
            }}
          >
            <Field label="Ваше сообщение">
              <textarea
                rows={3}
                required
                minLength={2}
                maxLength={10000}
                value={body}
                onChange={(e) => setBody(e.target.value)}
              />
            </Field>
            <Submit busy={busy}>
              Опубликовать <Send size={15} />
            </Submit>
          </form>
          {interaction.posts?.map((p) => (
            <div className="discussion-post" key={p.id}>
              <span className="avatar">{initials(p.user.name)}</span>
              <div>
                <b>{p.user.name}</b>
                <span className="eyebrow ml-3">{date(p.created_at)}</span>
                <p className="mt-2 whitespace-pre-wrap text-sm muted">
                  {p.body}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}
      {error && (
        <p className="error mt-4" role="alert">
          {error}
        </p>
      )}
      <div className="flex justify-end mt-7">
        <button
          className="btn primary"
          disabled={busy || completed}
          onClick={() => void act(complete)}
        >
          <CheckCircle2 size={17} />
          {completed ? "Элемент изучен" : "Отметить изученным"}
        </button>
      </div>
    </Modal>
  );
}
function VideoPlayer({ url }: { url: string }) {
  try {
    const u = new URL(url);
    let source = "";
    if (
      [
        "youtube.com",
        "www.youtube.com",
        "youtu.be",
        "www.youtube-nocookie.com",
      ].includes(u.hostname)
    ) {
      const id =
        u.hostname === "youtu.be"
          ? u.pathname.slice(1)
          : u.searchParams.get("v") || u.pathname.split("/embed/")[1];
      if (id && /^[\w-]{11}$/.test(id))
        source = "https://www.youtube-nocookie.com/embed/" + id;
    } else if (
      ["vimeo.com", "www.vimeo.com"].includes(u.hostname) &&
      /^\/\d+$/.test(u.pathname)
    ) {
      source = "https://player.vimeo.com/video" + u.pathname;
    }
    return source ? (
      <iframe
        className="video-player"
        title="Видеоматериал"
        src={source}
        allow="fullscreen; picture-in-picture"
        referrerPolicy="no-referrer"
        allowFullScreen
      />
    ) : (
      <video className="video-player" src={url} controls preload="metadata" />
    );
  } catch {
    return null;
  }
}
