import { forwardRef, type InputHTMLAttributes, type ReactNode } from "react";
import styles from "./RadioOption.module.css";

export interface RadioOptionProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "type" | "children"> { children: ReactNode }
/** Native radio semantics and keyboard navigation; feature owners supply group state. */
export const RadioOption = forwardRef<HTMLInputElement, RadioOptionProps>(function RadioOption({ children, className, ...props }, ref) {
  return <div className={styles.root}><label className={[styles.label, className].filter(Boolean).join(" ")}><input {...props} ref={ref} type="radio" /><span>{children}</span></label></div>;
});
