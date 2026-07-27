import { MagnifyingGlass } from "@phosphor-icons/react";
import {
  forwardRef,
  type ChangeEventHandler,
  type FormEventHandler,
} from "react";

import { Button } from "../Button/Button";
import { Input } from "../FormField/Input";
import styles from "./SearchInput.module.css";

export interface SearchInputProps {
  label: string;
  onChange: ChangeEventHandler<HTMLInputElement>;
  onSubmit: FormEventHandler<HTMLFormElement>;
  placeholder?: string;
  submitLabel: string;
  value: string;
}

export const SearchInput = forwardRef<HTMLInputElement, SearchInputProps>(
  function SearchInput(
    { label, onChange, onSubmit, placeholder, submitLabel, value },
    ref,
  ) {
    return (
      <form className={styles.form} role="search" onSubmit={onSubmit}>
        <label className={styles.field}>
          <span className={styles.visuallyHidden}>{label}</span>
          <MagnifyingGlass aria-hidden="true" className={styles.icon} size={21} />
          <Input
            ref={ref}
            autoComplete="off"
            onChange={onChange}
            placeholder={placeholder}
            type="search"
            value={value}
          />
        </label>
        <Button type="submit" variant="primary">
          <MagnifyingGlass aria-hidden="true" size={18} />
          {submitLabel}
        </Button>
      </form>
    );
  },
);
