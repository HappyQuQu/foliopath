import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";
import { ToastProvider } from "../../components/ui";
import { ThemeProvider } from "../../lib/theme/ThemeProvider";
import { SimpleAISettingsPrototype } from "./SimpleAISettingsPrototype";

const meta = {
  title: "Prototypes/Intelligence/Simplified settings",
  component: SimpleAISettingsPrototype,
  parameters: { layout: "fullscreen", docs: { description: { component: "PROPOSAL, not accepted production design. Disposable interactive prototype with in-memory synthetic data only. No API calls, no real analysis, no persistent settings. See docs/changes/PROTOTYPE-2026-10-22-ai-settings-simplification.md." } } },
  decorators: [(Story) => <ThemeProvider><ToastProvider><MemoryRouter initialEntries={["/settings/ai"]}><Story /></MemoryRouter></ToastProvider></ThemeProvider>],
} satisfies Meta<typeof SimpleAISettingsPrototype>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Overview: Story = {};
export const Offline: Story = { args: { initialLibrary: "archive" } };
