import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";
import { ToastProvider } from "../../components/ui";
import { ThemeProvider } from "../../lib/theme/ThemeProvider";
import { PhotoWorkspacePrototype } from "./PhotoWorkspacePrototype";
const meta = {
  title: "Prototypes/Intelligence/Photo workspace",
  component: PhotoWorkspacePrototype,
  parameters: { layout: "fullscreen", docs: { description: { component: "Historical V2 design archive, not the production implementation. Tags and settings have moved to Features/Curation/Photo tags and Features/Intelligence/Settings with production components. People/search provenance remain proposals. Synthetic samples only; Chinese fixture names are not model output. OCR unavailable." } } },
  decorators: [(Story) => <ThemeProvider><ToastProvider><MemoryRouter><Story /></MemoryRouter></ToastProvider></ThemeProvider>],
} satisfies Meta<typeof PhotoWorkspacePrototype>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Tags: Story = {};
export const Search: Story = { args: { initialView: "search" } };
export const People: Story = { args: { initialView: "people" } };
export const Settings: Story = { args: { initialView: "settings" } };
