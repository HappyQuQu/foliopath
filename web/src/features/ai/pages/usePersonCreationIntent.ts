import { useRef } from "react";
import { createRequestKey } from "../../../lib/requestKey";

// A retry of the same normalized draft reuses its key until confirmed success.
// Deliberately page-local: after reload the caller must inspect real server state.
export function usePersonCreationIntent() {
  const intent = useRef<{ name: string; idempotencyKey: string } | null>(null);
  return {
    forName(value: string) {
      const name = value.trim().normalize("NFC");
      if (intent.current?.name !== name) intent.current = { name, idempotencyKey: createRequestKey() };
      return intent.current!;
    },
    complete() { intent.current = null; },
  };
}
