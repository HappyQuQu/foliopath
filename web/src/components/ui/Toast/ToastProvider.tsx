import {
  createContext,
  useEffect,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { X } from "@phosphor-icons/react";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { IconButton } from "../Button/IconButton";
import styles from "./Toast.module.css";

export type ToastTone = "neutral" | "success" | "danger";

export interface ToastInput {
  message: string;
  tone?: ToastTone;
}

interface ToastRecord extends Required<ToastInput> {
  id: number;
}

interface ToastContextValue {
  dismiss: (id: number) => void;
  show: (toast: ToastInput) => number;
}

const ToastContext = createContext<ToastContextValue | null>(null);
export const TOAST_AUTO_DISMISS_MS = 6_000;

function ToastItem({
  toast,
  dismiss,
  closeLabel,
}: {
  toast: ToastRecord;
  dismiss: (id: number) => void;
  closeLabel: string;
}) {
  useEffect(() => {
    const timeout = window.setTimeout(() => dismiss(toast.id), TOAST_AUTO_DISMISS_MS);
    return () => window.clearTimeout(timeout);
  }, [dismiss, toast.id]);

  return (
    <div
      className={`${styles.toast} ${styles[toast.tone]}`}
      role={toast.tone === "danger" ? "alert" : "status"}
    >
      <span>{toast.message}</span>
      <IconButton label={closeLabel} onClick={() => dismiss(toast.id)}>
        <X aria-hidden="true" size={18} weight="bold" />
      </IconButton>
    </div>
  );
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const { t } = useLocale();
  const nextId = useRef(1);
  const [toasts, setToasts] = useState<ToastRecord[]>([]);

  const dismiss = useCallback((id: number) => {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, []);

  const show = useCallback((toast: ToastInput) => {
    const id = nextId.current++;
    setToasts((current) => [
      ...current,
      { id, message: toast.message, tone: toast.tone ?? "neutral" },
    ]);
    return id;
  }, []);

  const value = useMemo(() => ({ dismiss, show }), [dismiss, show]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className={styles.viewport} aria-label={t("common.toastRegion")} role="region">
        {toasts.map((toast) => (
          <ToastItem
            closeLabel={t("common.closeToast")}
            dismiss={dismiss}
            key={toast.id}
            toast={toast}
          />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const value = useContext(ToastContext);
  if (value === null) throw new Error("useToast must be used within ToastProvider");
  return value;
}
