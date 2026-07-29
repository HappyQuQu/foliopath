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
    <span
      aria-hidden="true"
      className={`${styles.mark} ${styles[size]} ${className ?? ""}`.trim()}
    >
      <img alt="" className={styles.art} src="/foliopath-mark-tree.svg" />
    </span>
  );
}
