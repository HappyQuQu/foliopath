import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState, type ComponentProps } from "react";

import { Switch } from "./Switch";

const meta = {
  title: "UI/Switch",
  component: Switch,
  args: {
    "aria-label": "固定预览",
  },
} satisfies Meta<typeof Switch>;

export default meta;

type Story = StoryObj<typeof meta>;

function ControlledSwitch(args: ComponentProps<typeof Switch>) {
  const [checked, setChecked] = useState(false);
  return (
    <Switch
      {...args}
      checked={checked}
      onChange={(event) => setChecked(event.currentTarget.checked)}
    />
  );
}

export const Default: Story = {
  render: (args) => <ControlledSwitch {...args} />,
};
