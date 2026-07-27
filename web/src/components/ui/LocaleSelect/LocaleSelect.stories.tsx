import type { Meta, StoryObj } from "@storybook/react-vite";

import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { LocaleSelect } from "./LocaleSelect";

const meta = {
  title: "UI/Locale select",
  component: LocaleSelect,
  decorators: [
    (Story) => (
      <LocaleProvider>
        <Story />
      </LocaleProvider>
    ),
  ],
  tags: ["autodocs"],
} satisfies Meta<typeof LocaleSelect>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
