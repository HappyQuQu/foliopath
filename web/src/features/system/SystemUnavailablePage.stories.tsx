import type { Meta, StoryObj } from "@storybook/react-vite";

import { PublicLayout } from "../../components/patterns/PublicLayout/PublicLayout";
import { ThemeProvider } from "../../lib/theme/ThemeProvider";
import { SystemUnavailablePage } from "./SystemUnavailablePage";

const meta = {
  title: "Features/System unavailable",
  component: SystemUnavailablePage,
  args: {
    onRetry: () => undefined,
  },
  decorators: [
    (Story) => (
      <ThemeProvider>
        <PublicLayout>
          <Story />
        </PublicLayout>
      </ThemeProvider>
    ),
  ],
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof SystemUnavailablePage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
