import { useQuery } from "@tanstack/react-query";

import {
  getSystemReadiness,
  type ReadinessReason,
} from "../../lib/api/readiness";
import type { MessageKey } from "../../lib/i18n/LocaleProvider";

export const systemKeys = {
  readiness: ["system", "readiness"] as const,
};

export function useSystemReadinessQuery() {
  return useQuery({
    queryKey: systemKeys.readiness,
    queryFn: getSystemReadiness,
    staleTime: 5_000,
  });
}

export function messageForReadiness(
  reasonCode: ReadinessReason,
  t: (key: MessageKey) => string,
): string {
  switch (reasonCode) {
    case "application_data_unavailable":
      return t("readiness.applicationData");
    case "migration_failed":
      return t("readiness.migration");
    case "database_unavailable":
      return t("readiness.database");
    case "shutting_down":
      return t("readiness.shuttingDown");
  }
}
