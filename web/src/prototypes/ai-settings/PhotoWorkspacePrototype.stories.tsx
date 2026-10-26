import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";
import { ToastProvider } from "../../components/ui";
import { ThemeProvider } from "../../lib/theme/ThemeProvider";
import { PhotoWorkspacePrototype } from "./PhotoWorkspacePrototype";
const meta = {
  title: "Prototypes/Intelligence/Photo workspace",
  component: PhotoWorkspacePrototype,
  parameters: { layout: "fullscreen", docs: { description: { component: "Historical V2 design archive, not the production implementation. Tags and settings have moved to Features/Curation/Photo tags and Features/Intelligence/Settings with production components. Search is implemented by Features/Search/Mode guidance/Fused results (CR-2026-035). People retains its own gate. Synthetic samples only; Chinese fixture names are not model output. OCR unavailable." } } },
  decorators: [(Story) => <ThemeProvider><ToastProvider><MemoryRouter><Story /></MemoryRouter></ToastProvider></ThemeProvider>],
} satisfies Meta<typeof PhotoWorkspacePrototype>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Tags: Story = {};
// Search moved to the production SearchPage story; the old screen remains only in this historical archive.
export const People: Story = { args: { initialView: "people" } };
export const Settings: Story = { args: { initialView: "settings" } };
