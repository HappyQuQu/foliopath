import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";

import { PublicLayout } from "../../../components/patterns/PublicLayout/PublicLayout";
import { ToastProvider } from "../../../components/ui/Toast/ToastProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AuthPage } from "./AuthPage";

const meta = {
  title: "Features/Authentication",
  component: AuthPage,
  args: {
    mode: "setup",
  },
  decorators: [
    (Story) => {
      const queryClient = new QueryClient({
        defaultOptions: {
          mutations: { retry: false },
          queries: { retry: false },
        },
      });

      return (
        <ThemeProvider>
          <QueryClientProvider client={queryClient}>
            <ToastProvider>
              <MemoryRouter>
                <PublicLayout>
                  <Story />
                </PublicLayout>
              </MemoryRouter>
            </ToastProvider>
          </QueryClientProvider>
        </ThemeProvider>
      );
    },
  ],
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof AuthPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const FirstAdministratorSetup: Story = {};

export const Login: Story = {
  args: {
    mode: "login",
  },
};
