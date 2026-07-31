import type { Meta, StoryObj } from "@storybook/react-vite";

import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { LocaleToggle } from "./LocaleToggle";

const meta = {
  title: "UI/Locale toggle",
  component: LocaleToggle,
  decorators: [
    (Story) => (
      <LocaleProvider>
        <Story />
      </LocaleProvider>
    ),
  ],
  tags: ["autodocs"],
} satisfies Meta<typeof LocaleToggle>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
