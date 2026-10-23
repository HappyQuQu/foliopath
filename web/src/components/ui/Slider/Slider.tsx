import { useId } from "react";

import styles from "./Slider.module.css";

export interface SliderProps {
  label: string;
  min: number;
  max: number;
  step: number;
  value: number;
  onChange: (value: number) => void;
}

export function Slider({ label, min, max, step, value, onChange }: SliderProps) {
  const id = useId();
  return (
    <label className={styles.slider} htmlFor={id}>
      <span>{label}</span>
      <input id={id} type="range" min={min} max={max} step={step} value={value}
        onChange={(event) => onChange(event.currentTarget.valueAsNumber)} />
    </label>
  );
}
