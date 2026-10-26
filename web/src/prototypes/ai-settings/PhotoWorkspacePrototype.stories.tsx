import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";
import { ToastProvider } from "../../components/ui";
import { ThemeProvider } from "../../lib/theme/ThemeProvider";
import { PhotoWorkspacePrototype } from "./PhotoWorkspacePrototype";
const meta = {
  title: "Prototypes/Intelligence/Photo workspace",
  component: PhotoWorkspacePrototype,
  parameters: { layout: "fullscreen", docs: { description: { component: "Historical V2 design archive, not the production implementation. Tags and settings use production component stories. Search uses Features/Search/Mode guidance/Fused results (CR-2026-035). Named-person detail uses Features/Intelligence/Review/Person detail (CR-2026-036); named directories use People directory/Empty people directory (CR-2026-039), group review uses Group detail/All groups (CR-2026-037/038). Recognition quality retains separate gates. Synthetic samples only. OCR unavailable." } } },
  decorators: [(Story) => <ThemeProvider><ToastProvider><MemoryRouter><Story /></MemoryRouter></ToastProvider></ThemeProvider>],
} satisfies Meta<typeof PhotoWorkspacePrototype>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Tags: Story = {};
// Search moved to the production SearchPage story; the old screen remains only in this historical archive.
// Person-detail curation moved to Features/Intelligence/Review/Person detail
// (CR-2026-036). The historical composite is not another acceptance screen.
export const Settings: Story = { args: { initialView: "settings" } };
