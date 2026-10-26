import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useRef, useState } from "react";
import { FaceReviewUndo, type FaceReviewReceipt } from "./FaceReviewUndo";

const exclusion: FaceReviewReceipt = { result: { action: "exclude_face", reviewId: "review_story_only", revision: 2, affectedPersonIds: [], undoable: true }, subject: "合成照片.png · 人脸 2", source: "青岚（合成）" };
const meta = {
  title: "Features/Intelligence/Review undo", component: FaceReviewUndo,
  parameters: { layout: "padded", docs: { description: { component: "CR-2026-041 · Production component with synthetic confirmed API receipt. Not a real review or undo result." } } },
  render: args => {
    const [client] = useState(() => new QueryClient({ defaultOptions: { mutations: { retry: false } } }));
    const [visible, setVisible] = useState(true); const heading = useRef<HTMLHeadingElement>(null);
    return <QueryClientProvider client={client}><h1 ref={heading} tabIndex={-1}>整理撤销 · 合成合同预览</h1>{visible && <FaceReviewUndo {...args} fallbackFocusRef={heading} onDismiss={() => setVisible(false)} />}<p>仅展示正式组件及合成回执，不代表服务端存在该审核记录。</p></QueryClientProvider>;
  },
  args: { onDismiss: () => {}, fallbackFocusRef: { current: null }, receipt: exclusion, session: { csrfToken: "synthetic-story-not-a-credential", expiresAt: "2099-01-01", administrator: { id: "admin_story", username: "fixture", displayName: "Synthetic" } } },
} satisfies Meta<typeof FaceReviewUndo>;
export default meta;
type Story = StoryObj<typeof meta>;
export const ConfirmedExclusion: Story = {};
export const ConfirmedMove: Story = { args: { receipt: { ...exclusion, result: { ...exclusion.result, action: "assign_face" }, target: "白露（合成）" } } };
export const ConfirmedMerge: Story = { args: { receipt: { ...exclusion, result: { ...exclusion.result, action: "merge_people" }, subject: "合并人物", target: "白露（合成）" } } };
