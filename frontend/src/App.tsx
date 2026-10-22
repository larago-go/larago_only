import { useCallback, useEffect, useState } from "react";
import {
  GraduationCap,
  LayoutDashboard,
  BookOpen,
  CalendarDays,
  ClipboardList,
  ChartNoAxesCombined,
  Users,
  Search,
  ArrowUpRight,
  ArrowRight,
  ChevronLeft,
  ChevronRight,
  Menu,
  X,
  LogOut,
  Plus,
  CheckCircle2,
  Clock,
  Flame,
  Sparkles,
  Check,
  FileText,
} from "lucide-react";
import Auth from "./Auth";
import { CohortsView, BankView } from "./Extended";
import CourseView, { CourseForm, GradeTable } from "./CourseView";
import {
  api,
  type Course,
  type User,
  type Detail,
  type Assignment,
  roles,
  initials,
  date,
} from "./types";
import { CourseCard, Empty, Modal } from "./components";
type Task = Assignment & {
  course: Course;
  submitted: boolean;
  score: number | null;
};
const route = () => window.location.hash.slice(2) || "dashboard";
const titles: Record<string, string> = {
  dashboard: "Обзор",
  courses: "Мои курсы",
  catalog: "Каталог курсов",
  calendar: "Календарь",
  tasks: "Задания",
  grades: "Мои результаты",
  users: "Пользователи",
  cohorts: "Глобальные группы",
  bank: "Банк контента",
};
export default function App() {
  const [user, setUser] = useState<User | null>(),
    [courses, setCourses] = useState<Course[]>([]),
    [tasks, setTasks] = useState<Task[]>([]),
    [page, setPage] = useState(route),
    [query, setQuery] = useState(""),
    [mobile, setMobile] = useState(false),
    [toast, setToast] = useState(""),
    [error, setError] = useState(""),
    [create, setCreate] = useState(false),
    [profile, setProfile] = useState(false);
  useEffect(() => {
    api<User>("/auth/me")
      .then(setUser)
      .catch(() => setUser(null));
    const handler = () => {
      setPage(route());
      setMobile(false);
      window.scrollTo(0, 0);
    };
    window.addEventListener("hashchange", handler);
    return () => window.removeEventListener("hashchange", handler);
  }, []);
  const notify = useCallback((message: string) => setToast(message), []);
  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => setToast(""), 5000);
    return () => clearTimeout(t);
  }, [toast]);
  const refresh = useCallback(async () => {
    const list = await api<Course[]>("/courses");
    setCourses(list);
    const accessible = list.filter(
      (c) =>
        (c.enrolled &&
          (!c.start_at || new Date(c.start_at) <= new Date()) &&
          (!c.end_at || new Date(c.end_at) >= new Date())) ||
        (user?.role === "teacher" && c.teacher_id === user?.id) ||
        user?.role === "admin",
    );
    const details = await Promise.all(
      accessible.map((c) =>
        api<Detail>("/courses/" + c.id).then((d) => ({ c, d })),
      ),
    );
    setTasks(
      details
        .flatMap(({ c, d }) =>
          d.assignments.map((a) => {
            const sub = d.submissions.find(
              (s) => s.assignment_id === a.id && s.user_id === user?.id,
            );
            return {
              ...a,
              course: c,
              submitted: !!sub,
              score: sub?.score ?? null,
            };
          }),
        )
        .sort(
          (a, b) => new Date(a.due_at).getTime() - new Date(b.due_at).getTime(),
        ),
    );
    setError("");
  }, [user?.id, user?.role]);
  useEffect(() => {
    if (user) void refresh().catch((e) => setError(e.message));
  }, [user, refresh]);
  const go = (path: string) => {
    window.location.hash = "/" + path;
    setQuery("");
  };
  if (user === undefined)
    return (
      <div className="loading-screen">
        <GraduationCap size={38} />
        <p>Открываем кампус…</p>
      </div>
    );
  if (!user)
    return (
      <Auth
        onLogin={(u) => {
          setUser(u);
          go("dashboard");
        }}
      />
    );
  const isStaff = user.role !== "student",
    mine = courses.filter(
      (c) => c.enrolled || c.teacher_id === user.id || user.role === "admin",
    ),
    filtered = courses.filter((c) =>
      (c.title + " " + c.category + " " + c.teacher.name)
        .toLowerCase()
        .includes(query.toLowerCase()),
    ),
    upcoming = tasks.filter((t) => !t.submitted),
    completed = courses.filter(
      (c) => c.lessons > 0 && c.progress === c.lessons,
    ).length;
  const nav = [
    { id: "dashboard", title: "Обзор", icon: LayoutDashboard },
    {
      id: "courses",
      title: isStaff ? "Мои курсы" : "Моё обучение",
      icon: BookOpen,
    },
    { id: "catalog", title: "Каталог курсов", icon: Search },
    { id: "calendar", title: "Календарь", icon: CalendarDays },
    { id: "tasks", title: "Задания", icon: ClipboardList },
    {
      id: "grades",
      title: isStaff ? "Журнал оценок" : "Мои результаты",
      icon: ChartNoAxesCombined,
    },
    ...(isStaff
      ? [
          { id: "cohorts", title: "Глобальные группы", icon: Users },
          { id: "bank", title: "Банк контента", icon: BookOpen },
        ]
      : []),
    ...(user.role === "admin"
      ? [{ id: "users", title: "Пользователи", icon: Users }]
      : []),
  ];
  const courseId = page.startsWith("course/")
    ? Number(page.split("/")[1])
    : null;
  return (
    <div className="app-shell">
      {mobile && (
        <button
          className="mobile-shade"
          aria-label="Закрыть меню"
          onClick={() => setMobile(false)}
        />
      )}
      <aside className={"sidebar " + (mobile ? "open" : "")}>
        <button className="brand" onClick={() => go("dashboard")}>
          <span className="brand-icon">
            <GraduationCap size={25} />
          </span>
          кампус<span className="brand-dot">.</span>
        </button>
        <span className="nav-label">ВАШЕ ПРОСТРАНСТВО</span>
        <nav>
          {nav.map((n) => (
            <button
              key={n.id}
              className={
                "nav-item " +
                (page === n.id || (courseId && n.id === "courses")
                  ? "active"
                  : "")
              }
              onClick={() => go(n.id)}
            >
              <n.icon size={19} />
              <span>{n.title}</span>
              {n.id === "tasks" && upcoming.length > 0 && (
                <span className="nav-count">{upcoming.length}</span>
              )}
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <div className="help-card">
            <span className="help-spark">✳︎</span>
            <h4>Знания меняют всё</h4>
            <p>Найдите то, что вдохновит вас на следующий шаг.</p>
            <button onClick={() => go("catalog")}>
              Открыть каталог <ArrowUpRight size={15} />
            </button>
          </div>
          <div className="profile-row">
            <button
              className="flex gap-3 items-center min-w-0 text-left"
              onClick={() => setProfile(true)}
            >
              <span className="avatar">{initials(user.name)}</span>
              <span className="min-w-0">
                <b className="block text-xs truncate">
                  {user.name.split(" ").slice(0, 2).join(" ")}
                </b>
                <span className="text-xs text-slate-400">
                  {roles[user.role]}
                </span>
              </span>
            </button>
            <button
              className="icon-btn"
              title="Выйти"
              aria-label="Выйти"
              onClick={() =>
                void api("/auth/logout", "POST")
                  .then(() => {
                    setUser(null);
                    setCourses([]);
                    setTasks([]);
                  })
                  .catch((e) => notify(e.message))
              }
            >
              <LogOut size={17} />
            </button>
          </div>
        </div>
      </aside>
      <div className="main-shell">
        <header className="topbar">
          <div className="flex items-center gap-3">
            <button
              className="icon-btn mobile-menu"
              aria-label="Открыть меню"
              onClick={() => setMobile(true)}
            >
              <Menu size={21} />
            </button>
            <span className="text-sm text-slate-400">Мой кампус</span>
            <ChevronRight size={14} className="text-slate-300" />
            <b className="text-sm font-medium">
              {courseId ? "Курс" : titles[page] || "Обзор"}
            </b>
          </div>
          <div className="header-right">
            <label className="search">
              <Search size={17} />
              <input
                aria-label="Найти курс"
                placeholder="Найти что-нибудь…"
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value);
                  if (courseId) go("catalog");
                }}
              />
              <kbd>⌕</kbd>
            </label>
            <span className="header-divider" />
            <button
              className="avatar"
              aria-label="Профиль"
              onClick={() => setProfile(true)}
            >
              {initials(user.name)}
            </button>
          </div>
        </header>
        <main className="main-content">
          {error && (
            <div className="error mb-6" role="alert">
              {error}
              <button
                className="ml-4 underline"
                onClick={() => void refresh().catch((e) => setError(e.message))}
              >
                Повторить загрузку
              </button>
            </div>
          )}
          {courseId ? (
            <CourseView
              key={courseId}
              id={courseId}
              user={user}
              courses={courses}
              back={() => go("courses")}
              refresh={refresh}
              notify={notify}
            />
          ) : (
            <>
              {page === "dashboard" && (
                <>
                  <div className="page-heading">
                    <div>
                      <span className="eyebrow">РАДЫ ВАС ВИДЕТЬ</span>
                      <h1>
                        Привет, {user.name.split(" ")[0]}{" "}
                        <span className="wave">✺</span>
                      </h1>
                      <p>Хороший день, чтобы узнать что-то новое.</p>
                    </div>
                    <span className="today">
                      <CalendarDays size={16} />
                      {new Date().toLocaleDateString("ru-RU", {
                        day: "numeric",
                        month: "long",
                        weekday: "short",
                      })}
                    </span>
                  </div>
                  <div className="dashboard-grid">
                    <div className="dashboard-main">
                      <section className="welcome-banner">
                        <div className="banner-copy">
                          <span className="banner-tag">
                            <span /> УЧИТЕСЬ В СВОЁМ ТЕМПЕ
                          </span>
                          <h2>
                            Ваше будущее
                            <br />
                            начинается здесь
                            <span>
                              <ArrowUpRight size={29} className="inline" />
                            </span>
                          </h2>
                          <p>
                            Новые навыки, интересные задачи
                            <br />и ещё один шаг к вашим целям.
                          </p>
                          <button
                            className="btn dark"
                            onClick={() =>
                              mine.length
                                ? go("course/" + mine[0].id)
                                : go("catalog")
                            }
                          >
                            {mine.length
                              ? "Продолжить обучение"
                              : "Найти первый курс"}
                            <ArrowRight size={16} />
                          </button>
                        </div>
                        <div className="banner-art" aria-hidden="true">
                          <div className="planet-ring" />
                          <div className="floating-card card-back">
                            <span>✳︎</span>
                            <div />
                            <div />
                          </div>
                          <div className="floating-card card-front">
                            <GraduationCap size={48} strokeWidth={1.3} />
                            <b>Всё получится.</b>
                            <span>Шаг за шагом</span>
                            <div className="art-progress">
                              <i />
                            </div>
                          </div>
                          <span className="art-star">✳︎</span>
                          <span className="art-star small-star">✧</span>
                        </div>
                      </section>
                      <div className="stats-grid">
                        {[
                          {
                            label: isStaff
                              ? "Доступных курсов"
                              : "Курсов в обучении",
                            value: mine.length,
                            icon: BookOpen,
                            color: "violet",
                            sub: "Продолжайте открывать новое",
                          },
                          {
                            label: isStaff
                              ? "Заданий в курсах"
                              : "Ближайших заданий",
                            value: isStaff ? tasks.length : upcoming.length,
                            icon: ClipboardList,
                            color: "orange",
                            sub: "Всё важное под рукой",
                          },
                          {
                            label: "Курсов завершено",
                            value: completed,
                            icon: CheckCircle2,
                            color: "green",
                            sub: "Каждый шаг имеет значение",
                          },
                        ].map((s) => (
                          <div className="stat-card" key={s.label}>
                            <span className={"activity-icon " + s.color}>
                              <s.icon size={21} />
                            </span>
                            <div>
                              <span className="muted text-xs">{s.label}</span>
                              <div className="stat-value">
                                {s.value}
                                <span>{s.sub}</span>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                      <div className="section-heading mt-8 mb-5">
                        <h2>
                          {isStaff ? "Ваши курсы" : "Продолжить обучение"}{" "}
                          <span className="count">{mine.length}</span>
                        </h2>
                        <button
                          className="text-button"
                          onClick={() => go("courses")}
                        >
                          Все курсы <ArrowRight size={16} />
                        </button>
                      </div>
                      <div className="course-grid dashboard-courses">
                        {filtered
                          .filter((c) => mine.some((m) => m.id === c.id))
                          .slice(0, 4)
                          .map((c) => (
                            <CourseCard
                              key={c.id}
                              course={c}
                              onOpen={() => go("course/" + c.id)}
                            />
                          ))}
                      </div>
                      {!mine.length && (
                        <Empty title="Здесь начнётся ваше обучение">
                          <button
                            className="btn primary mt-4"
                            onClick={() => go("catalog")}
                          >
                            Выбрать курс
                          </button>
                        </Empty>
                      )}
                    </div>
                    <aside className="dashboard-aside">
                      <MiniCalendar tasks={tasks} />
                      <div className="panel p-5 mt-5">
                        <div className="section-heading mb-5">
                          <h3>Скоро сдавать</h3>
                          <button
                            className="icon-btn"
                            aria-label="Все задания"
                            onClick={() => go("tasks")}
                          >
                            <ArrowUpRight size={17} />
                          </button>
                        </div>
                        {upcoming.slice(0, 3).map((t) => (
                          <button
                            key={t.id}
                            className="deadline"
                            onClick={() => go("course/" + t.course_id)}
                          >
                            <span
                              className={"deadline-marker " + t.course.color}
                            />
                            <div>
                              <span className="deadline-course">
                                {t.course.title}
                              </span>
                              <b>{t.title}</b>
                              <span className="deadline-date">
                                <Clock size={12} />
                                {date(t.due_at)}
                              </span>
                            </div>
                          </button>
                        ))}
                        {!upcoming.length && (
                          <div className="text-sm muted flex gap-2">
                            <CheckCircle2 size={17} /> Все работы отправлены
                          </div>
                        )}
                      </div>
                      <div className="quote-card">
                        <Sparkles size={22} />
                        <p>
                          «Образование — это не подготовка к жизни. Это и есть
                          жизнь».
                        </p>
                        <span>Джон Дьюи</span>
                      </div>
                    </aside>
                  </div>
                </>
              )}
              {(page === "courses" || page === "catalog") && (
                <CourseList
                  courses={filtered.filter(
                    (c) =>
                      page === "catalog" || mine.some((m) => m.id === c.id),
                  )}
                  catalog={page === "catalog"}
                  staff={isStaff}
                  create={() => setCreate(true)}
                  open={(c) => go("course/" + c.id)}
                />
              )}
              {page === "tasks" && (
                <>
                  <div className="page-heading">
                    <div>
                      <span className="eyebrow">ОТ ТЕОРИИ К ПРАКТИКЕ</span>
                      <h1>Задания</h1>
                      <p>Ваши практические работы и обратная связь.</p>
                    </div>
                  </div>
                  <TaskList
                    tasks={tasks}
                    open={(t) => go("course/" + t.course_id)}
                    staff={isStaff}
                  />
                </>
              )}
              {page === "calendar" && (
                <>
                  <div className="page-heading">
                    <div>
                      <span className="eyebrow">ПЛАНИРУЙТЕ СВОЙ РИТМ</span>
                      <h1>Календарь обучения</h1>
                      <p>Важные даты и сроки сдачи заданий.</p>
                    </div>
                  </div>
                  <div className="calendar-layout">
                    <MiniCalendar tasks={tasks} large />
                    <div className="panel p-6">
                      <h2 className="mb-6">Расписание заданий</h2>
                      {tasks.map((t) => (
                        <button
                          className="calendar-task"
                          key={t.id}
                          onClick={() => go("course/" + t.course_id)}
                        >
                          <span className="calendar-date">
                            <b>{new Date(t.due_at).getDate()}</b>
                            {new Date(t.due_at).toLocaleDateString("ru-RU", {
                              month: "short",
                            })}
                          </span>
                          <div>
                            <h3>{t.title}</h3>
                            <p className="muted text-sm mt-2">
                              {t.course.title}
                            </p>
                            <span
                              className={
                                "badge mt-3 " +
                                (t.submitted ? "green" : "orange")
                              }
                            >
                              {t.submitted ? "Работа отправлена" : "К сдаче"}
                            </span>
                          </div>
                          <ChevronRight size={16} />
                        </button>
                      ))}
                      {!tasks.length && <Empty title="Ближайших событий нет" />}
                    </div>
                  </div>
                </>
              )}
              {page === "grades" && (
                <>
                  <div className="page-heading">
                    <div>
                      <span className="eyebrow">КАЖДЫЙ ШАГ ИМЕЕТ ЗНАЧЕНИЕ</span>
                      <h1>{isStaff ? "Журнал оценок" : "Мои результаты"}</h1>
                      <p>
                        Прогресс, практические работы и результаты тестирования.
                      </p>
                    </div>
                  </div>
                  <div className="space-y-6">
                    {mine.map((c) => (
                      <section className="panel" key={c.id}>
                        <div className="p-6 border-b border-slate-100">
                          <h2>{c.title}</h2>
                        </div>
                        <GradeTable courseId={c.id} />
                      </section>
                    ))}
                    {!mine.length && <Empty title="Результатов пока нет" />}
                  </div>
                </>
              )}
              {page === "users" && user.role === "admin" && (
                <UsersView notify={notify} />
              )}
              {page === "cohorts" && isStaff && (
                <CohortsView user={user} notify={notify} />
              )}
              {page === "bank" && isStaff && (
                <BankView
                  user={user}
                  courses={courses}
                  notify={notify}
                  refresh={refresh}
                />
              )}
              {!titles[page] && (
                <Empty title="Страница не найдена">
                  <button className="btn mt-3" onClick={() => go("dashboard")}>
                    На главную
                  </button>
                </Empty>
              )}
            </>
          )}
          <footer>
            кампус<span>.</span>
            <p>Место для новых возможностей</p>
            <span className="footer-dot" /> Учитесь с удовольствием
          </footer>
        </main>
      </div>
      {toast && (
        <div className="toast" role="status">
          <CheckCircle2 size={19} />
          {toast}
          <button aria-label="Закрыть уведомление" onClick={() => setToast("")}>
            <X size={16} />
          </button>
        </div>
      )}
      {create && (
        <CourseForm
          close={() => setCreate(false)}
          saved={async () => {
            await refresh();
            notify("Курс создан");
          }}
        />
      )}
      {profile && (
        <Modal title="Мой профиль" close={() => setProfile(false)}>
          <div className="flex items-center gap-4 mb-6">
            <span className="avatar large">{initials(user.name)}</span>
            <div>
              <h2>{user.name}</h2>
              <p className="muted mt-1">{user.email}</p>
            </div>
          </div>
          <span className="badge">{roles[user.role]}</span>
          <p className="muted text-sm mt-6">
            Роль и доступ к курсам назначаются администратором. Ваши результаты
            сохраняются в личном аккаунте.
          </p>
        </Modal>
      )}
    </div>
  );
}
function CourseList({
  courses,
  catalog,
  staff,
  create,
  open,
}: {
  courses: Course[];
  catalog: boolean;
  staff: boolean;
  create: () => void;
  open: (c: Course) => void;
}) {
  const [category, setCategory] = useState("Все"),
    [status, setStatus] = useState("Все курсы");
  const filtered = courses.filter(
    (c) =>
      (category === "Все" || c.category === category) &&
      (status === "Все курсы" ||
        (status === "Завершённые"
          ? c.lessons > 0 && c.progress === c.lessons
          : c.progress < c.lessons || c.lessons === 0)),
  );
  return (
    <>
      <div className="page-heading">
        <div>
          <span className="eyebrow">ОТКРЫВАЙТЕ НОВЫЕ ВОЗМОЖНОСТИ</span>
          <h1>{catalog ? "Каталог курсов" : "Мои курсы"}</h1>
          <p>
            {catalog
              ? "Найдите направление, которое вдохновляет."
              : "Всё, что вы изучаете и создаёте, в одном месте."}
          </p>
        </div>
        {staff && (
          <button className="btn primary" onClick={create}>
            <Plus size={17} /> Создать курс
          </button>
        )}
      </div>
      <div className="filters">
        <div className="category-tabs">
          {[
            "Все",
            ...new Set(courses.map((c) => c.category).filter(Boolean)),
          ].map((cat) => (
            <button
              key={cat}
              className={category === cat ? "active" : ""}
              onClick={() => setCategory(cat)}
            >
              {cat}
            </button>
          ))}
        </div>
        {!catalog && (
          <select
            aria-label="Статус курса"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option>Все курсы</option>
            <option>В процессе</option>
            <option>Завершённые</option>
          </select>
        )}
      </div>
      <p className="eyebrow mb-5">КУРСОВ: {filtered.length}</p>
      <div className="course-grid">
        {filtered.map((c) => (
          <CourseCard key={c.id} course={c} onOpen={() => open(c)} />
        ))}
      </div>
      {!filtered.length && (
        <Empty title="Курсы не найдены">
          Попробуйте изменить поиск или фильтры.
        </Empty>
      )}
    </>
  );
}
function TaskList({
  tasks,
  open,
  staff,
}: {
  tasks: Task[];
  open: (t: Task) => void;
  staff: boolean;
}) {
  const [filter, setFilter] = useState("all");
  const list = tasks.filter(
    (t) => filter === "all" || (filter === "todo" ? !t.submitted : t.submitted),
  );
  return (
    <>
      <div className="category-tabs mb-6">
        {[
          { id: "all", text: "Все задания" },
          { id: "todo", text: "К выполнению" },
          { id: "done", text: "Отправленные" },
        ]
          .filter((t) => !staff || t.id === "all")
          .map((t) => (
            <button
              className={filter === t.id ? "active" : ""}
              key={t.id}
              onClick={() => setFilter(t.id)}
            >
              {t.text}
            </button>
          ))}
      </div>
      <div className="panel">
        {list.map((t) => (
          <button key={t.id} className="task-row" onClick={() => open(t)}>
            <span className={"activity-icon " + t.course.color}>
              <FileText size={21} />
            </span>
            <div className="flex-1 text-left">
              <h3>{t.title}</h3>
              <p className="muted text-sm mt-1">{t.course.title}</p>
            </div>
            <span className="muted text-sm">{date(t.due_at)}</span>
            {!staff && (
              <span className={"badge " + (t.submitted ? "green" : "orange")}>
                {t.score !== null
                  ? `${t.score} / ${t.max_score}`
                  : t.submitted
                    ? "На проверке"
                    : "К сдаче"}
              </span>
            )}
            <ChevronRight size={16} />
          </button>
        ))}
        {!list.length && <Empty title="Заданий пока нет" />}
      </div>
    </>
  );
}
function MiniCalendar({
  tasks,
  large = false,
}: {
  tasks: Task[];
  large?: boolean;
}) {
  const [month, setMonth] = useState(
      () => new Date(new Date().getFullYear(), new Date().getMonth(), 1),
    ),
    [selected, setSelected] = useState<Date>();
  const first =
      (new Date(month.getFullYear(), month.getMonth(), 1).getDay() + 6) % 7,
    count = new Date(month.getFullYear(), month.getMonth() + 1, 0).getDate(),
    today = new Date();
  const days = Array.from(
    { length: Math.ceil((first + count) / 7) * 7 },
    (_, i) => i - first + 1,
  );
  const same = (a: Date, b: Date) =>
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate();
  return (
    <div className={"panel mini-calendar " + (large ? "large-calendar" : "")}>
      <div className="calendar-heading">
        <h3>
          {month
            .toLocaleDateString("ru-RU", { month: "long", year: "numeric" })
            .replace(" г.", "")}
        </h3>
        <div className="flex">
          <button
            className="icon-btn"
            aria-label="Предыдущий месяц"
            onClick={() =>
              setMonth((d) => new Date(d.getFullYear(), d.getMonth() - 1, 1))
            }
          >
            <ChevronLeft size={16} />
          </button>
          <button
            className="icon-btn"
            aria-label="Следующий месяц"
            onClick={() =>
              setMonth((d) => new Date(d.getFullYear(), d.getMonth() + 1, 1))
            }
          >
            <ChevronRight size={16} />
          </button>
        </div>
      </div>
      <div className="calendar-grid">
        {["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"].map((d) => (
          <span className="weekday" key={d}>
            {d}
          </span>
        ))}
        {days.map((d, i) => {
          const real = d > 0 && d <= count,
            day = new Date(month.getFullYear(), month.getMonth(), d),
            has = tasks.some((t) => same(new Date(t.due_at), day));
          return (
            <button
              disabled={!real}
              aria-label={real ? day.toLocaleDateString("ru-RU") : undefined}
              key={i}
              className={
                "calendar-day " +
                (real && same(day, today) ? "today " : "") +
                (selected && same(day, selected) ? "selected " : "") +
                (has && real ? "has-event" : "")
              }
              onClick={() => setSelected(day)}
            >
              {real ? d : ""}
            </button>
          );
        })}
      </div>
      {selected && (
        <div className="calendar-selected">
          <b>
            {selected.toLocaleDateString("ru-RU", {
              day: "numeric",
              month: "long",
            })}
          </b>
          {tasks
            .filter((t) => same(new Date(t.due_at), selected))
            .map((t) => (
              <p key={t.id}>{t.title}</p>
            ))}
          {!tasks.some((t) => same(new Date(t.due_at), selected)) && (
            <p>Нет запланированных заданий</p>
          )}
        </div>
      )}
      <div className="calendar-key">
        <span /> Срок сдачи задания
      </div>
    </div>
  );
}
function UsersView({ notify }: { notify: (s: string) => void }) {
  const [users, setUsers] = useState<User[]>([]),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(0);
  const load = () =>
    api<User[]>("/users")
      .then(setUsers)
      .catch((e) => setError(e.message));
  useEffect(() => {
    void load();
  }, []);
  return (
    <>
      <div className="page-heading">
        <div>
          <span className="eyebrow">УПРАВЛЕНИЕ ПЛАТФОРМОЙ</span>
          <h1>Пользователи</h1>
          <p>Назначайте роли и открывайте возможности для обучения.</p>
        </div>
      </div>
      {error && <p className="error">{error}</p>}
      <div className="panel table-scroll">
        <table>
          <thead>
            <tr>
              <th>Пользователь</th>
              <th>Почта</th>
              <th>Роль</th>
              <th>Дата регистрации</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id}>
                <td>
                  <div className="flex gap-3 items-center">
                    <span className="avatar">{initials(u.name)}</span>
                    <b>{u.name}</b>
                  </div>
                </td>
                <td>{u.email}</td>
                <td>
                  <select
                    aria-label={"Роль пользователя " + u.name}
                    value={u.role}
                    disabled={busy === u.id}
                    onChange={(e) => {
                      setBusy(u.id);
                      void api("/users/" + u.id + "/role", "PATCH", {
                        role: e.target.value,
                      })
                        .then(async () => {
                          await load();
                          notify("Роль обновлена");
                        })
                        .catch((e) => notify(e.message))
                        .finally(() => setBusy(0));
                    }}
                  >
                    {Object.entries(roles).map(([key, v]) => (
                      <option key={key} value={key}>
                        {v}
                      </option>
                    ))}
                  </select>
                </td>
                <td>{date(u.created_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
