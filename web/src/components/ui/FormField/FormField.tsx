import { useId, type InputHTMLAttributes } from "react";

import { Input } from "./Input";
import styles from "./FormField.module.css";

export interface FormFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "id"> {
  description?: string | undefined;
  error?: string | undefined;
  id?: string | undefined;
  label: string;
}

export function FormField({
  description,
  error,
  id,
  label,
  required,
  ...inputProps
}: FormFieldProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const descriptionId = description ? `${inputId}-description` : undefined;
  const errorId = error ? `${inputId}-error` : undefined;
  const describedBy = [descriptionId, errorId].filter(Boolean).join(" ") || undefined;

  return (
    <div className={styles.field}>
      <label htmlFor={inputId}>
        {label}
        {required && <span aria-hidden="true"> *</span>}
      </label>
      {description && (
        <span className={styles.description} id={descriptionId}>
          {description}
        </span>
      )}
      <Input
        {...inputProps}
        id={inputId}
        required={required}
        invalid={Boolean(error)}
        aria-describedby={describedBy}
      />
      {error && (
        <span className={styles.error} id={errorId} role="alert">
          {error}
        </span>
      )}
    </div>
  );
}
