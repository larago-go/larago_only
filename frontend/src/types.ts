export type Role = "student" | "teacher" | "admin";
export interface User {
  id: number;
  name: string;
  email: string;
  role: Role;
  created_at: string;
}
export interface Course {
  short_name: string;
  format: "topics" | "weeks";
  language: string;
  start_at: string | null;
  end_at: string | null;
  disable_self_enrollment: boolean;
  id: number;
  title: string;
  description: string;
  category: string;
  color: string;
  published: boolean;
  teacher_id: number;
  teacher: User;
  enrolled: boolean;
  progress: number;
  lessons: number;
  students: number;
}
export interface Lesson {
  data?: ActivityData;
  files?: Resource[];
  url?: string;
  kind?: string;
  id: number;
  title: string;
  content: string;
  duration: number;
  section_id: number;
}
export interface Section {
  summary: string;
  hidden: boolean;
  id: number;
  title: string;
  position: number;
  lessons: Lesson[];
}
export interface Assignment {
  section_id: number;
  id: number;
  course_id: number;
  title: string;
  description: string;
  due_at: string;
  max_score: number;
}
export interface Submission {
  id: number;
  assignment_id: number;
  user_id: number;
  user: User;
  content: string;
  score: number | null;
  feedback: string;
  created_at: string;
  updated_at: string;
}
export interface Quiz {
  section_id: number;
  id: number;
  title: string;
  questions: { id: number; prompt: string; options: string[] }[];
}
export interface Attempt {
  id: number;
  quiz_id: number;
  user_id: number;
  score: number;
  total: number;
  created_at: string;
}
export interface Announcement {
  id: number;
  title: string;
  body: string;
  created_at: string;
}
export interface Discussion {
  id: number;
  user: User;
  body: string;
  created_at: string;
}
export interface Detail {
  course: Course;
  sections: Section[];
  assignments: Assignment[];
  quizzes: Quiz[];
  completed: number[];
  announcements: Announcement[];
  discussions: Discussion[];
  submissions: Submission[];
  attempts: Attempt[];
}
export interface Gradebook {
  rows: {
    user: User;
    completed: number;
    lessons: number;
    submissions: Submission[];
    attempts: Attempt[];
  }[];
  assignments: Assignment[];
  quizzes: Quiz[];
}
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  const response = await fetch("/api" + path, {
    method,
    credentials: "same-origin",
    headers: body ? { "Content-Type": "application/json" } : {},
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await response
    .json()
    .catch(() => ({ error: "Сервер вернул некорректный ответ" }));
  if (!response.ok) throw new Error(data.error || "Ошибка запроса");
  return data as T;
}
export const roles: Record<Role, string> = {
  student: "Студент",
  teacher: "Преподаватель",
  admin: "Администратор",
};
export const date = (s: string) =>
  new Date(s).toLocaleDateString("ru-RU", { day: "numeric", month: "short" });
export const initials = (s: string) =>
  s
    .split(" ")
    .slice(0, 2)
    .map((x) => x[0])
    .join("");
export interface Cohort {
  id: number;
  name: string;
  description: string;
  count: number;
  members?: { id: number; user: User; user_id: number }[];
}
export interface CourseCohort {
  id: number;
  cohort_id: number;
  cohort: Cohort;
}
export interface BankQuestion {
  prompt: string;
  options: string[];
  correct_index: number;
}
export interface BankItem {
  data?: ActivityData;
  id: number;
  author_id: number;
  author: User;
  title: string;
  category: string;
  kind: string;
  body: string;
  url: string;
  duration: number;
  max_score: number;
  questions: BankQuestion[];
}
export interface ImportReport {
  source: string;
  sections: number;
  activities: number;
  counts: Record<string, number>;
  question_bank: number;
  warnings: string[];
  skipped_files: number;
}
export const kinds: Record<string, string> = {
  page: "Страница",
  label: "Текст и медиа",
  url: "Внешняя ссылка",
  assignment: "Задание",
  quiz: "Тест",
};

export interface ActivityData {
  chapters?: { title: string; content: string }[];
  entries?: { term: string; definition: string }[];
  options?: string[];
}
export interface Resource {
  id: number;
  name: string;
  size: number;
  content_type: string;
}
export const activityKinds: Record<string, string> = {
  page: "Страница",
  label: "Текстовый блок",
  url: "Ссылка",
  video: "Видео",
  book: "Книга",
  file: "Файл",
  folder: "Папка",
  forum: "Форум",
  glossary: "Глоссарий",
  choice: "Опрос",
  assignment: "Задание",
  quiz: "Тест",
};
