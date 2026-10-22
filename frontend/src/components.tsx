import { type ReactNode, useState } from "react";
import {
  X,
  LoaderCircle,
  ArrowUpRight,
  BookOpen,
  Code2,
  ChartNoAxesCombined,
  Shapes,
  Users,
} from "lucide-react";
import { type Course, initials } from "./types";
export function Modal({
  title,
  children,
  close,
}: {
  title: string;
  children: ReactNode;
  close: () => void;
}) {
  return (
    <div className="modal-shade" onClick={close}>
      <section
        className="modal"
        role="dialog"
        aria-modal="true"
        aria-label={title}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-6">
          <h2>{title}</h2>
          <button className="icon-btn" aria-label="Закрыть" onClick={close}>
            <X size={20} />
          </button>
        </div>
        {children}
      </section>
    </div>
  );
}
export function Submit({
  children,
  busy = false,
}: {
  children: ReactNode;
  busy?: boolean;
}) {
  return (
    <button type="submit" className="btn primary" disabled={busy}>
      {busy ? <LoaderCircle className="animate-spin" size={17} /> : null}
      {children}
    </button>
  );
}
export function Field({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <label className="field">
      <span>{label}</span>
      {children}
    </label>
  );
}
export function Empty({
  title,
  children,
}: {
  title: string;
  children?: ReactNode;
}) {
  return (
    <div className="empty">
      <BookOpen size={30} />
      <h3>{title}</h3>
      <p>
        {children || "Здесь появятся материалы, когда они будут добавлены."}
      </p>
    </div>
  );
}
export function FormModal({
  title,
  close,
  submit,
  children,
}: {
  title: string;
  close: () => void;
  submit: (form: FormData) => Promise<void>;
  children: ReactNode;
}) {
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  return (
    <Modal title={title} close={close}>
      <form
        className="space-y-4"
        onSubmit={async (e) => {
          e.preventDefault();
          const form = new FormData(e.currentTarget);
          setBusy(true);
          setError("");
          try {
            await submit(form);
            close();
          } catch (err) {
            setError((err as Error).message);
          } finally {
            setBusy(false);
          }
        }}
      >
        {children}
        {error && (
          <p role="alert" className="error">
            {error}
          </p>
        )}
        <div className="flex justify-end gap-3 pt-3">
          <button type="button" className="btn" onClick={close}>
            Отмена
          </button>
          <Submit busy={busy}>Сохранить</Submit>
        </div>
      </form>
    </Modal>
  );
}
export function Cover({
  course,
  small = false,
}: {
  course: Course;
  small?: boolean;
}) {
  const Icon =
    course.color === "blue"
      ? Code2
      : course.color === "green"
        ? ChartNoAxesCombined
        : course.color === "orange"
          ? Users
          : Shapes;
  return (
    <div className={"course-cover " + course.color + (small ? " small" : "")}>
      <div className="cover-grid" />
      <span className="cover-tag">{course.category || "Обучение"}</span>
      <div className="cover-orbit orbit-one" />
      <div className="cover-orbit orbit-two" />
      <div className="cover-object">
        <Icon size={small ? 34 : 54} strokeWidth={1.3} />
      </div>
      <span className="cover-spark">✳︎</span>
      <span className="cover-label">
        CAMPUS / {String(course.id).padStart(2, "0")}
      </span>
    </div>
  );
}
export function CourseCard({
  course,
  onOpen,
}: {
  course: Course;
  onOpen: () => void;
}) {
  const progress = course.lessons
    ? Math.round((course.progress / course.lessons) * 100)
    : 0;
  return (
    <button className="course-card group" onClick={onOpen}>
      <Cover course={course} />
      <div className="p-5">
        <div className="flex justify-between items-center mb-3">
          <span className="eyebrow">{course.lessons} уроков</span>
          {!course.published && <span className="badge">Черновик</span>}
          <ArrowUpRight
            size={17}
            className="text-slate-400 group-hover:text-violet-600"
          />
        </div>
        <h3>{course.title}</h3>
        <div className="teacher-line">
          <span className="avatar tiny">{initials(course.teacher.name)}</span>
          {course.teacher.name}
        </div>
        {course.enrolled ? (
          <div className="mt-5">
            <div className="flex justify-between text-xs mb-2">
              <span className="text-slate-500">Ваш прогресс</span>
              <b>{progress}%</b>
            </div>
            <div className="progress">
              <i style={{ width: progress + "%" }} />
            </div>
          </div>
        ) : (
          <div className="flex justify-between mt-5 text-xs text-slate-500">
            <span>{course.students} студентов</span>
            <span className="text-violet-600 font-semibold">
              Открыть курс →
            </span>
          </div>
        )}
      </div>
    </button>
  );
}
