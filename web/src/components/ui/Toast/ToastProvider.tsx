import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";

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

export function ToastProvider({ children }: { children: ReactNode }) {
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
      <div className={styles.viewport} aria-label="通知">
        {toasts.map((toast) => (
          <div
            className={`${styles.toast} ${styles[toast.tone]}`}
            key={toast.id}
            role={toast.tone === "danger" ? "alert" : "status"}
          >
            <span>{toast.message}</span>
            <IconButton label="关闭通知" onClick={() => dismiss(toast.id)}>
              <span aria-hidden="true">×</span>
            </IconButton>
          </div>
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
