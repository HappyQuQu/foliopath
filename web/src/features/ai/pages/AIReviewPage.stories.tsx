import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { within, userEvent, expect, waitFor } from "storybook/test";

import { ToastProvider } from "../../../components/ui";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { mediaKeys } from "../../media";
import { curationKeys } from "../../curation";
import { AIReviewPage } from "./AIReviewPage";
import { aiKeys } from "../queries";

const meta = {
  title: "Features/Intelligence/Review",
  component: AIReviewPage,
  args: { session: { administrator: { displayName: "管理员", id: "adm-story", username: "admin" }, csrfToken: "storybook-csrf-token-that-is-long-enough", expiresAt: "2099-09-03T00:00:00Z" } },
  decorators: [(Story, context) => <StoryBoundary mergeBlocked={context.parameters.mergeBlocked ?? false} separate={context.parameters.separate ?? false} recovery={context.parameters.recovery ?? false} emptyPeople={context.parameters.emptyPeople ?? false} noPending={context.parameters.noPending ?? false} review={context.parameters.review ?? "pending"} group={context.parameters.groupDetail} detail={context.parameters.personDetail ?? false} status={context.parameters.tagStatus ?? "pending"} empty={context.parameters.emptySuggestions ?? false} view={context.parameters.view ?? "tags"}><Story /></StoryBoundary>],
  parameters: { layout: "fullscreen", docs: { description: { component: "Synthetic UI fixtures, not live model analysis." } } },
} satisfies Meta<typeof AIReviewPage>;
export default meta;
type Story = StoryObj<typeof meta>;

export const EmptyPeopleDirectory: Story = { parameters: { view: "people", emptyPeople: true } };
export const PeopleDirectory: Story = { parameters: { view: "people" } };
export const PeopleSidebar: Story = { parameters: { view: "people" } };
export const SinglePhotoReview: Story = { parameters: { view: "faces" } };
export const SinglePhotoSeparatePeople: Story = {
 parameters: { view:"faces", separate:true, docs:{description:{story:"CR-2026-043 · Production controls; synthetic same-photo assignments only. Confirmation is not a successful write or model result."}} },
 play: async({canvasElement})=>{const canvas=within(canvasElement);const user=userEvent.setup({document:canvasElement.ownerDocument});const first=canvas.getByRole("radio",{name:/^(人脸 1 · 已归类|Face 1 · Assigned)$/});await waitFor(()=>expect(first).toBeEnabled());await user.click(first);await user.click(await canvas.findByText(/^(防止误合并|Prevent mistaken merges)$/,{selector:"summary"}));await user.selectOptions(canvas.getByRole("combobox",{name:/^(另一张已归类的人脸|Other assigned face)$/}),"face-story-2");await user.click(canvas.getByRole("button",{name:/^(标记为不同人物|Mark as different people)$/}));},
};
export const PersonDetail: Story = { parameters: { view: "people", personDetail: true } };
export const PersonDetailRecovery: Story = {
  parameters: { view: "people", personDetail: true, recovery: true, docs: { description: { story: "CR-2026-042 · Production recovery UI with synthetic503 and merged-source404. No write reaches a real API; this is not a successful review or model result." } } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement); const user = userEvent.setup({ document: canvasElement.ownerDocument });
    await user.click(canvas.getByRole("button", { name: /^(合并人物|Merge people)$/ }));
    const dialog = within(canvas.getByRole("dialog", { name: /^(合并人物|Merge people)$/ }));
    await user.selectOptions(dialog.getByRole("combobox"), "person-target-story"); await user.click(dialog.getByRole("checkbox"));
    await user.click(dialog.getByRole("button", { name: /^(确认合并|Confirm merge)$/ }));
    await dialog.findByText(/尚未确认保存结果|Saving has not been confirmed/);
  },
};
export const PersonMergeBlocked: Story = {
 parameters:{view:"people",personDetail:true,mergeBlocked:true,docs:{description:{story:"CR-2026-044 · Production merge dialog; synthetic four constraints with one readable photo pair. No API write succeeds or reaches a real service."}}},
 play:async({canvasElement})=>{const canvas=within(canvasElement);const user=userEvent.setup({document:canvasElement.ownerDocument});await user.click(canvas.getByRole("button",{name:/^(合并人物|Merge people)$/}));const dialog=within(canvas.getByRole("dialog"));await user.selectOptions(dialog.getByRole("combobox"),"person-target-story");await dialog.findByText(/有 4 对人脸|Face pairs marked as different people: 4/);},
};
export const GroupDetail: Story = { parameters: { view: "clusters", groupDetail: "core" } };
export const SimilarGroupDetail: Story = { parameters: { view: "clusters", groupDetail: "edge" } };
export const NoPendingGroups: Story = { parameters: { view: "clusters", noPending: true } };
export const AllGroups: Story = { parameters: { view: "clusters", review: "all" } };
export const UnnamedGroupsSidebar: Story = { parameters: { view: "clusters" } };
export const PendingSuggestions: Story = {};
export const AcceptedSuggestions: Story = { parameters: { tagStatus: "accepted" } };
export const DismissedSuggestions: Story = { parameters: { tagStatus: "dismissed" } };
export const NoPendingSuggestions: Story = { parameters: { emptySuggestions: true } };

function StoryBoundary({ mergeBlocked, separate, children, status, empty, view, detail, group, review, noPending, emptyPeople, recovery }: { mergeBlocked: boolean; separate: boolean; recovery: boolean; emptyPeople: boolean; noPending: boolean; review: "all" | "pending"; group?: "core" | "edge"; view: string; children: ReactNode; status: "pending" | "accepted" | "dismissed"; empty: boolean; detail: boolean }) {
  const [client] = useState(() => {
    const client = new QueryClient({ defaultOptions: { queries: { enabled: false, retry: false, staleTime: Infinity, refetchOnMount: false, refetchOnWindowFocus: false, refetchOnReconnect: false } } });
    client.setQueryData(["libraries", "list"], { pages: [{ items: [{ id: "lib-story", name: "界面测试媒体库", status: "ready", assetCount: 10, directoryCount: 1 }], nextCursor: null }], pageParams: [undefined] });
    const coverage = { eligible: 10, completed: 8, degraded: 1, failed: 0, stale: 1, complete: false, revision: 2 };
    const sample = {
      id: "suggestion-story", libraryId: "lib-story", generationId: "gen-story", status: "pending", confidence: .93,
      revision: 2, vocabularyRevision: 1, reviewedAt: null, tag: { tagId: "tag-story", name: "蓝色" },
      asset: { id: "asset-story", libraryId: "lib-story", libraryName: "界面测试媒体库", directoryId: "dir-story",
        name: "合成色块.png", relativePath: "合成色块.png", kind: "image", mimeType: "image/png", sizeBytes: 1000,
        modifiedAt: "2026-09-01T00:00:00Z", width: 1200, height: 800, durationMs: null, probeStatus: "ready",
        playbackStatus: "not_applicable", sourceAvailability: "available",
        thumbnail: { status: "ready", url: "/test-media/viewer-blue-violet.png", errorCode: null } },
    };
    const items = [sample, { ...sample, id: "suggestion-violet", confidence: .81, tag: { tagId: "tag-violet", name: "紫色" } }, { ...sample, id: "suggestion-second", confidence: .76, asset: { ...sample.asset, id: "asset-second", name: "测试色块-02.png" } }];
    for (const filter of ["pending", "accepted", "dismissed"] as const) client.setQueryData(aiKeys.suggestions("lib-story", filter), { items: !empty && filter === status ? items.map(item => ({ ...item, status: filter, reviewedAt: filter === "pending" ? null : "2026-10-22T00:00:00Z" })) : [], nextCursor: null, coverage });
    client.setQueryData(mediaKeys.asset("asset-story"), sample.asset);
    client.setQueryData(curationKeys.asset("asset-story"), { assetId: "asset-story", favorite: false, favoritedAt: null, revision: 1, tags: [], etag: '"curation-r1"' });
    client.setQueryData(curationKeys.asset("asset-second"), { assetId: "asset-second", favorite: false, favoritedAt: null, revision: 1, tags: [], etag: '"curation-r1"' });
    client.setQueryData(aiKeys.people(""), { items: emptyPeople ? [] : detail ? [{ id: "person-story", name: "青岚（合成）", previewAssetId: "asset-story", revision: 1, confirmedFaceCount: 2, assetCount: 1 }, { id: "person-target-story", name: "白露（合成）", previewAssetId: "asset-second", revision: 1, confirmedFaceCount: 1, assetCount: 1 }] : [{ id: "person-story", name: "青岚（合成）", previewAssetId: "asset-story", revision: 1, confirmedFaceCount: 4, assetCount: 3 }], nextCursor: null });
    client.setQueryData(aiKeys.person("person-story"), { id: "person-story", name: "青岚（合成）", previewAssetId: "asset-story", revision: 1, confirmedFaceCount: 2, assetCount: 1, etag: '"person-r1"' });
    client.setQueryData(aiKeys.person("person-target-story"), {id:"person-target-story",name:"白露（合成）",revision:1,confirmedFaceCount:1,assetCount:1,previewAssetId:"asset-story"});
    client.setQueryData(aiKeys.mergeCheck("person-story", "person-target-story"), {sourcePersonId:"person-story",targetPersonId:"person-target-story",sourceRevision:1,targetRevision:1,canMerge:!mergeBlocked,cannotLinkCount:mergeBlocked?4:0,duplicateAssignmentCount:0,sourceLimitExceeded:false,examples:mergeBlocked?[{sourceLibraryId:"lib-story",sourceAssetId:"asset-story",targetLibraryId:"lib-story",targetAssetId:"asset-second"}]:[]});
    client.setQueryData(aiKeys.personAssets("person-story"), { items: [{ asset: sample.asset, faceIds: ["face-story-1", "face-story-2"] }], nextCursor: null });
    client.setQueryData(aiKeys.assetFaces("asset-story"), { items: [1, 2].map(ordinal => ({ faceId: group ? `face-group-${ordinal}` : `face-story-${ordinal}`, assetId: "asset-story", personId: group ? null : separate && ordinal === 2 ? "person-target-story" : "person-story", cannotLinkFaceIds: [], ordinal, revision: 1, state: group ? `anonymous_${group}` : "assigned", region: { xPercent: ordinal === 1 ? 15 : 60, yPercent: 15, widthPercent: 20, heightPercent: 30 } })) });
    client.setQueryData(aiKeys.assetFaces("asset-second"), { items: [{ faceId: "face-group-3", assetId: "asset-second", personId: null, ordinal: 1, revision: 1, state: noPending ? "excluded" : `anonymous_${group ?? "core"}`, region: { xPercent: 15, yPercent: 15, widthPercent: 20, heightPercent: 30 } }] });
    const groupKind = group ?? "core";
    client.setQueryData(aiKeys.cluster("lib-story", "cluster-story"), { cluster: { id: "cluster-story", libraryId: "lib-story", kind: groupKind, revision: 1, memberCount: 3, pendingMemberCount: noPending ? 0 : group ? 3 : 1, previewAssetIds: ["asset-story", "asset-second"] }, nextCursor: null, items: [1, 2, 3].map(number => ({ faceId: `face-group-${number}`, assetId: number === 3 ? "asset-second" : "asset-story", kind: groupKind, revision: 1, region: { xPercent: number === 2 ? 60 : 15, yPercent: 15, widthPercent: 20, heightPercent: 30 } })) });
    const groups = [{ id: "cluster-story", libraryId: "lib-story", kind: "core", revision: 1, memberCount: 3, pendingMemberCount: noPending ? 0 : group ? 3 : 1, previewAssetIds: ["asset-story", "asset-second"] }, { id: "cluster-done", libraryId: "lib-story", kind: "core", revision: 1, memberCount: 2, pendingMemberCount: 0, previewAssetIds: ["asset-story"] }];
    for (const filter of ["pending", "all"] as const) client.setQueryData(aiKeys.clusters("lib-story", "core", "", filter), { items: filter === "all" ? groups : noPending ? [] : groups.slice(0,1), nextCursor: null, coverage, groupAssignmentAllowed: false, counts: { totalGroups: 2, pendingGroups: noPending ? 0 : 1, totalFaces: 5, pendingFaces: noPending ? 0 : group ? 3 : 1 } });
    return client;
  });
  useEffect(() => {
    const original = window.fetch; let merged = false;
    window.fetch = async (input, init) => {
      const url = new URL(input instanceof Request ? input.url : String(input), window.location.origin);
      if (!url.pathname.startsWith("/api/v1/")) return original(input, init);
      const method = init?.method ?? (input instanceof Request ? input.method : "GET");
      if (!recovery && !mergeBlocked) {
        if(method === "GET" && url.pathname === "/api/v1/people/person-story/merge-check")return Response.json(client.getQueryData(aiKeys.mergeCheck("person-story","person-target-story")));
        return original(input,init);
      }
      if (method === "POST" && recovery && url.pathname === "/api/v1/ai/face-reviews") {
        merged = true;
        return Response.json({ error: { code: "internal_error", message: "Synthetic lost response; no real write", requestId: "story-only" } }, { status: 503 });
      }
      const source = "/api/v1/people/person-story";
      if (method === "GET" && merged && (url.pathname === source || url.pathname === source + "/assets")) return Response.json({ error: { code: "person_not_found", message: "Synthetic merged source", requestId: "story-only" } }, { status: 404 });
      const key = url.pathname.endsWith("/merge-check") ? aiKeys.mergeCheck("person-story","person-target-story") : url.pathname === "/api/v1/people" ? aiKeys.people("") : url.pathname === source ? aiKeys.person("person-story") : url.pathname === source + "/assets" ? aiKeys.personAssets("person-story") : null;
      const value = key && client.getQueryData<Record<string, unknown>>(key);
      if (method === "GET" && value) {
        const { etag, ...body } = value;
        return Response.json(body, typeof etag === "string" ? { headers: { ETag: etag } } : {});
      }
      return Response.json({ error: { code: "precondition_failed", message: "Read-only synthetic recovery fixture", requestId: "story-only" } }, { status: 412 });
    };
    return () => { window.fetch = original; };
  }, [client, recovery, mergeBlocked]);
  return <ThemeProvider><QueryClientProvider client={client}><ToastProvider><MemoryRouter initialEntries={[`/intelligence?view=${view}&libraryId=lib-story&status=${status}${review === "all" ? "&review=all" : ""}${detail ? "&person=person-story" : ""}${view === "faces" ? "&asset=asset-story" : ""}${group ? `&cluster=cluster-story&kind=${group}` : ""}`]}>{children}<aside role="note" style={{ padding: "var(--space-3)", color: "var(--color-text-muted)" }}>组件预览 · 合成数据，不是真实识别结果 / Synthetic preview, not model output</aside></MemoryRouter></ToastProvider></QueryClientProvider></ThemeProvider>;
}
