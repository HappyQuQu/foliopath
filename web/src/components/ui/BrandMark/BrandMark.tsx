import styles from "./BrandMark.module.css";

export type BrandMarkSize = "large" | "medium" | "small";

export function BrandMark({
  className,
  size = "medium",
}: {
  className?: string;
  size?: BrandMarkSize;
}) {
  return (
    <img
      alt=""
      aria-hidden="true"
      className={`${styles.mark} ${styles[size]} ${className ?? ""}`.trim()}
      src="/foliopath-mark.svg"
    />
  );
}
