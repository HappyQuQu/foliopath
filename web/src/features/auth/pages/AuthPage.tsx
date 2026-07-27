import { ImageSquare } from "@phosphor-icons/react";
import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { Button } from "../../../components/ui/Button/Button";
import { FormField } from "../../../components/ui/FormField/FormField";
import { InlineStatus } from "../../../components/ui/InlineStatus/InlineStatus";
import { ApiError } from "../../../lib/api/errors";
import {
  useLocale,
  type MessageKey,
} from "../../../lib/i18n/LocaleProvider";
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
  const { t } = useLocale();
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
    }, t);

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
      setPageError(messageFor(error, t));
    }
  }

  return (
    <section className={styles.card} aria-labelledby="auth-title">
      <div className={styles.mark}>
        <ImageSquare aria-hidden="true" size={30} weight="duotone" />
      </div>
      <p className={styles.eyebrow}>{setup ? t("auth.firstUse") : t("auth.login")}</p>
      <h1 id="auth-title">{setup ? t("auth.setupTitle") : t("auth.loginTitle")}</h1>
      <p className={styles.intro}>
        {setup ? t("auth.setupIntro") : t("auth.loginIntro")}
      </p>

      {showSessionNotice && (
        <InlineStatus onDismiss={() => setShowSessionNotice(false)}>
          {t("auth.sessionExpired")}
        </InlineStatus>
      )}

      {pageError && <InlineStatus tone="danger">{pageError}</InlineStatus>}

      <form className={styles.form} noValidate onSubmit={handleSubmit}>
        {setup && (
          <FormField
            autoComplete="name"
            error={fieldErrors.displayName}
            label={t("auth.displayName")}
            maxLength={128}
            name="displayName"
            required
          />
        )}
        <FormField
          autoCapitalize="none"
          autoComplete="username"
          error={fieldErrors.username}
          label={t("auth.username")}
          maxLength={64}
          name="username"
          required
        />
        <FormField
          autoComplete={setup ? "new-password" : "current-password"}
          error={fieldErrors.password}
          label={t("auth.password")}
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
            label={t("auth.confirmPassword")}
            maxLength={128}
            minLength={12}
            name="confirmPassword"
            required
            type="password"
          />
        )}
        <Button className={styles.submit} loading={pending} type="submit" variant="primary">
          {setup ? t("auth.create") : t("auth.login")}
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
}, t: (key: MessageKey) => string): FieldErrors {
  const errors: FieldErrors = {};

  if (!username) errors.username = t("validation.username");
  if (username.length > 64) errors.username = t("validation.usernameLength");
  if (!password) errors.password = t("validation.password");

  if (setup) {
    if (!displayName) errors.displayName = t("validation.displayName");
    if (displayName.length > 128) errors.displayName = t("validation.displayNameLength");
    if (password.length < 12) errors.password = t("validation.passwordLength");
    if (password !== confirmPassword) errors.confirmPassword = t("validation.confirmPassword");
  }

  return errors;
}

function messageFor(error: unknown, t: (key: MessageKey) => string): string {
  if (!(error instanceof ApiError)) return t("auth.unknownFailure");

  switch (error.code) {
    case "invalid_credentials":
      return t("auth.invalidCredentials");
    case "setup_in_progress":
      return t("auth.setupInProgress");
    case "rate_limited":
      return t("auth.rateLimited");
    case "origin_invalid":
      return t("auth.originInvalid");
    case "validation_failed":
    case "invalid_request":
      return t("auth.invalidInput");
    default:
      return t("auth.unavailable");
  }
}
