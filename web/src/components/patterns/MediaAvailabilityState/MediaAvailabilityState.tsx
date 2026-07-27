import {
  CloudSlash,
  FileX,
  FilmSlate,
  LockKey,
  WarningDiamond,
} from "@phosphor-icons/react";

import { Button } from "../../ui";
import styles from "./MediaAvailabilityState.module.css";

export type MediaAvailabilityKind =
  | "deleted"
  | "invalid"
  | "loadFailed"
  | "missing"
  | "offline"
  | "unreadable"
  | "unsupported"
  | "unsupportedCodec";

export interface MediaAvailabilityPresentation {
  actionLabel?: string;
  description: string;
  kind: MediaAvailabilityKind;
  onAction?: () => void;
  title: string;
}

export function MediaAvailabilityState({
  compact = false,
  state,
}: {
  compact?: boolean;
  state: MediaAvailabilityPresentation;
}) {
  const Icon =
    state.kind === "offline"
      ? CloudSlash
      : state.kind === "unreadable"
        ? LockKey
        : state.kind === "unsupportedCodec"
          ? FilmSlate
          : state.kind === "invalid" ||
              state.kind === "unsupported" ||
              state.kind === "loadFailed"
            ? WarningDiamond
            : FileX;

  return (
    <section
      className={styles.state}
      data-compact={compact || undefined}
      role="status"
    >
      <span className={styles.icon}>
        <Icon aria-hidden="true" size={compact ? 30 : 38} weight="duotone" />
      </span>
      <div>
        <h2>{state.title}</h2>
        <p>{state.description}</p>
      </div>
      {state.actionLabel && state.onAction && (
        <Button onClick={state.onAction} size="small" variant="secondary">
          {state.actionLabel}
        </Button>
      )}
    </section>
  );
}
