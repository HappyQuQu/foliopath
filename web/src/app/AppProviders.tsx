import { QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";

import { ToastProvider } from "../components/ui/Toast/ToastProvider";
import { LocaleProvider } from "../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../lib/theme/ThemeProvider";
import { HeaderAddonProvider } from "../components/patterns/GlobalHeader/HeaderAddonContext";
import { NotificationCenter } from "../features/notifications/NotificationCenter";
import { createQueryClient } from "./query/create-query-client";

export function AppProviders({ children }: { children: ReactNode }) {
  const [queryClient] = useState(createQueryClient);

  return (
    <LocaleProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <ToastProvider>
            <HeaderAddonProvider value={<NotificationCenter />}>
              {children}
            </HeaderAddonProvider>
          </ToastProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </LocaleProvider>
  );
}
