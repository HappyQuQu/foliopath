import type { InputHTMLAttributes } from "react";
import styles from "./Checkbox.module.css";

export function Checkbox({ className, ...props }: Omit<InputHTMLAttributes<HTMLInputElement>, "type">) {
  return <input {...props} type="checkbox" className={[styles.checkbox, className].filter(Boolean).join(" ")} />;
}
