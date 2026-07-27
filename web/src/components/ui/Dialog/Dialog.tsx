import { useEffect, useId, useRef, type ReactNode } from "react";

import { Button } from "../Button/Button";
import { IconButton } from "../Button/IconButton";
import styles from "./Dialog.module.css";

export interface DialogProps {
  actions?: ReactNode;
  children: ReactNode;
  description?: string;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  title: string;
}

export function Dialog({
  actions,
  children,
  description,
  onOpenChange,
  open,
  title,
}: DialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const restoreFocusRef = useRef<HTMLElement | null>(null);
  const titleId = useId();
  const descriptionId = useId();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (dialog === null) return;

    if (open && !dialog.open) {
      restoreFocusRef.current =
        document.activeElement instanceof HTMLElement ? document.activeElement : null;
      dialog.showModal();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  }, [open]);

  useEffect(
    () => () => {
      restoreFocusRef.current?.focus();
    },
    [],
  );

  return (
    <dialog
      ref={dialogRef}
      className={styles.dialog}
      aria-labelledby={titleId}
      aria-describedby={description ? descriptionId : undefined}
      onCancel={(event) => {
        event.preventDefault();
        onOpenChange(false);
      }}
      onClose={() => {
        if (open) onOpenChange(false);
        restoreFocusRef.current?.focus();
      }}
    >
      <header className={styles.header}>
        <div>
          <h2 id={titleId}>{title}</h2>
          {description && <p id={descriptionId}>{description}</p>}
        </div>
        <IconButton label="关闭对话框" onClick={() => onOpenChange(false)}>
          <span aria-hidden="true">×</span>
        </IconButton>
      </header>
      <div className={styles.content}>{children}</div>
      {actions && <footer className={styles.actions}>{actions}</footer>}
    </dialog>
  );
}

export function DialogCloseButton({
  children = "取消",
  onClick,
}: {
  children?: ReactNode;
  onClick: () => void;
}) {
  return <Button onClick={onClick}>{children}</Button>;
}
