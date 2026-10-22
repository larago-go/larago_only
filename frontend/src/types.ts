export type Role = "student" | "teacher" | "admin";
export interface User {
  id: number;
  name: string;
  email: string;
  role: Role;
  created_at: string;
}
export interface Course {
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
  url?: string;
  kind?: string;
  id: number;
  title: string;
  content: string;
  duration: number;
  section_id: number;
}
export interface Section {
  id: number;
  title: string;
  position: number;
  lessons: Lesson[];
}
export interface Assignment {
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
