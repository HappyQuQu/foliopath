import { ImageSquare } from "@phosphor-icons/react";
import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { Button } from "../../../components/ui/Button/Button";
import { FormField } from "../../../components/ui/FormField/FormField";
import { InlineStatus } from "../../../components/ui/InlineStatus/InlineStatus";
import { ApiError } from "../../../lib/api/errors";
import {
  useLoginMutation,
  useSetupAdministratorMutation,
} from "../queries";
import styles from "./AuthPage.module.css";

export type AuthPageMode = "login" | "setup";

interface FieldErrors {
  confirmPassword?: string;
  displayName?: string;
  password?: string;
  username?: string;
}

export function AuthPage({ mode }: { mode: AuthPageMode }) {
  const setup = mode === "setup";
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [pageError, setPageError] = useState<string>();
  const [showSessionNotice, setShowSessionNotice] = useState(
    !setup && searchParams.get("reason") === "session_expired",
  );
  const loginMutation = useLoginMutation();
  const setupMutation = useSetupAdministratorMutation();
  const pending = loginMutation.isPending || setupMutation.isPending;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPageError(undefined);

    const values = new FormData(event.currentTarget);
    const displayName = String(values.get("displayName") ?? "").trim();
    const username = String(values.get("username") ?? "").trim();
    const password = String(values.get("password") ?? "");
    const confirmPassword = String(values.get("confirmPassword") ?? "");
    const errors = validate({
      confirmPassword,
      displayName,
      password,
      setup,
      username,
    });

    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) return;

    try {
      if (setup) {
        await setupMutation.mutateAsync({ displayName, password, username });
      } else {
        await loginMutation.mutateAsync({ password, username });
      }
      navigate("/settings/general", { replace: true });
    } catch (error) {
      if (error instanceof ApiError && error.code === "setup_closed") {
        navigate("/login", { replace: true });
        return;
      }
      setPageError(messageFor(error));
    }
  }

  return (
    <section className={styles.card} aria-labelledby="auth-title">
      <div className={styles.mark}>
        <ImageSquare aria-hidden="true" size={30} weight="duotone" />
      </div>
      <p className={styles.eyebrow}>{setup ? "首次使用" : "欢迎回来"}</p>
      <h1 id="auth-title">{setup ? "创建管理员账户" : "登录 FolioPath"}</h1>
      <p className={styles.intro}>
        {setup
          ? "此账户用于管理媒体库、扫描任务与系统设置。"
          : "使用管理员账户继续访问您的媒体库。"}
      </p>

      {showSessionNotice && (
        <InlineStatus onDismiss={() => setShowSessionNotice(false)}>
          为了保护您的媒体库，会话已过期。请重新登录。
        </InlineStatus>
      )}

      {pageError && <InlineStatus tone="danger">{pageError}</InlineStatus>}

      <form className={styles.form} noValidate onSubmit={handleSubmit}>
        {setup && (
          <FormField
            autoComplete="name"
            error={fieldErrors.displayName}
            label="显示名称"
            maxLength={128}
            name="displayName"
            required
          />
        )}
        <FormField
          autoCapitalize="none"
          autoComplete="username"
          error={fieldErrors.username}
          label="用户名"
          maxLength={64}
          name="username"
          required
        />
        <FormField
          autoComplete={setup ? "new-password" : "current-password"}
          error={fieldErrors.password}
          label="密码"
          maxLength={128}
          minLength={setup ? 12 : 1}
          name="password"
          required
          type="password"
        />
        {setup && (
          <FormField
            autoComplete="new-password"
            error={fieldErrors.confirmPassword}
            label="确认密码"
            maxLength={128}
            minLength={12}
            name="confirmPassword"
            required
            type="password"
          />
        )}
        <Button className={styles.submit} loading={pending} type="submit" variant="primary">
          {setup ? "创建账户" : "登录"}
        </Button>
      </form>
    </section>
  );
}

function validate({
  confirmPassword,
  displayName,
  password,
  setup,
  username,
}: {
  confirmPassword: string;
  displayName: string;
  password: string;
  setup: boolean;
  username: string;
}): FieldErrors {
  const errors: FieldErrors = {};

  if (!username) errors.username = "请输入用户名。";
  if (username.length > 64) errors.username = "用户名不能超过 64 个字符。";
  if (!password) errors.password = "请输入密码。";

  if (setup) {
    if (!displayName) errors.displayName = "请输入显示名称。";
    if (displayName.length > 128) errors.displayName = "显示名称不能超过 128 个字符。";
    if (password.length < 12) errors.password = "密码至少需要 12 个字符。";
    if (password !== confirmPassword) errors.confirmPassword = "两次输入的密码不一致。";
  }

  return errors;
}

function messageFor(error: unknown): string {
  if (!(error instanceof ApiError)) return "操作没有完成，请稍后重试。";

  switch (error.code) {
    case "invalid_credentials":
      return "用户名或密码不正确。";
    case "setup_in_progress":
      return "另一项初始化正在进行，请稍后重试。";
    case "rate_limited":
      return "尝试次数过多，请稍后再试。";
    case "origin_invalid":
      return "当前页面来源未通过安全检查，请从 FolioPath 正式地址访问。";
    case "validation_failed":
    case "invalid_request":
      return "请检查输入内容后重试。";
    default:
      return "暂时无法完成操作，请稍后重试。";
  }
}
