import { useCallback, useRef } from "react";

export function useSubmissionGuard() {
  const submissionActive = useRef(false);

  return useCallback(async <Result,>(
    action: () => Promise<Result>,
  ): Promise<Result | undefined> => {
    if (submissionActive.current) return undefined;

    submissionActive.current = true;
    try {
      return await action();
    } finally {
      submissionActive.current = false;
    }
  }, []);
}
