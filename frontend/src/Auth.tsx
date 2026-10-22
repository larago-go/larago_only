import { useState } from "react";
import {
  ArrowRight,
  GraduationCap,
  BookOpen,
  ShieldCheck,
  Sparkles,
} from "lucide-react";
import { api, type User, roles, type Role } from "./types";
import { Field, Submit } from "./components";
export default function Auth({ onLogin }: { onLogin: (u: User) => void }) {
  const [register, setRegister] = useState(false),
    [email, setEmail] = useState(""),
    [password, setPassword] = useState(""),
    [name, setName] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const login = async (e: string, p: string) => {
    setBusy(true);
    setError("");
    try {
      onLogin(
        await api<User>(
          "/auth/" + (register ? "register" : "login"),
          "POST",
          register
            ? { email: e, password: p, name }
            : { email: e, password: p },
        ),
      );
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="auth-page">
      <div className="auth-story">
        <a className="brand" href="#/">
          <span className="brand-icon">
            <GraduationCap />
          </span>
          кампус<span className="brand-dot">.</span>
        </a>
        <div>
          <span className="story-kicker">
            <Sparkles size={16} /> ВАШ СЛЕДУЮЩИЙ ШАГ
          </span>
          <h1>
            Большие идеи
            <br />
            начинаются
            <br />с любопытства<span>.</span>
          </h1>
          <p>
            Учитесь новому. Делитесь опытом.
            <br />
            Создавайте то, что имеет значение.
          </p>
          <div className="story-art">
            <div className="art-book">
              <BookOpen size={100} strokeWidth={1} />
            </div>
            <div className="art-dot one" />
            <div className="art-dot two" />
            <div className="art-note">
              Маленькие шаги.
              <br />
              <b>Большие возможности.</b>
            </div>
          </div>
        </div>
        <span className="text-sm text-violet-200">
          Пространство, в котором хочется учиться
        </span>
      </div>
      <main className="auth-form">
        <div className="w-full max-w-sm">
          <div className="badge mb-5">ПЛАТФОРМА ОБУЧЕНИЯ</div>
          <h2 className="text-3xl">
            {register ? "Начнём знакомство" : "С возвращением!"}
          </h2>
          <p className="text-slate-500 mt-3 mb-8">
            {register
              ? "Создайте аккаунт и найдите свой первый курс."
              : "Войдите, чтобы продолжить своё обучение."}
          </p>
          <form
            className="space-y-5"
            onSubmit={(e) => {
              e.preventDefault();
              void login(email, password);
            }}
          >
            {register && (
              <Field label="Ваше имя">
                <input
                  required
                  minLength={2}
                  maxLength={100}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Александра Морозова"
                  autoComplete="name"
                />
              </Field>
            )}
            <Field label="Электронная почта">
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                placeholder="you@example.com"
                autoComplete="email"
              />
            </Field>
            <Field label="Пароль">
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={register ? 8 : 1}
                maxLength={72}
                placeholder="Не менее 8 символов"
                autoComplete={register ? "new-password" : "current-password"}
              />
            </Field>
            {error && (
              <p className="error" role="alert">
                {error}
              </p>
            )}
            <Submit busy={busy}>
              {register ? "Создать аккаунт" : "Войти в кампус"}
              <ArrowRight size={17} />
            </Submit>
          </form>
          <p className="text-sm text-slate-500 mt-6">
            {register ? "Уже есть аккаунт?" : "Впервые здесь?"}{" "}
            <button
              className="text-violet-600 font-semibold"
              onClick={() => {
                setRegister(!register);
                setError("");
              }}
            >
              {register ? "Войти" : "Зарегистрироваться"}
            </button>
          </p>
          {!register && (
            <div className="demo-box">
              <div className="flex items-center gap-2 mb-3 text-sm font-semibold">
                <ShieldCheck size={16} /> Познакомьтесь с платформой
              </div>
              <p className="text-xs text-slate-500 mb-4">
                Демонстрационные аккаунты доступны, если при запуске включено
                демонаполнение.
              </p>
              <div className="flex flex-wrap gap-2">
                {(["student", "teacher", "admin"] as Role[]).map((role) => (
                  <button
                    className="demo-btn"
                    key={role}
                    disabled={busy}
                    onClick={() => {
                      setEmail(role + "@campus.local");
                      setPassword("Learn2026!");
                      void login(role + "@campus.local", "Learn2026!");
                    }}
                  >
                    {roles[role]}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
