import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { ApiError } from "../../../lib/api/errors";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { PersonPanel } from "./PersonPanel";

const mocks = vi.hoisted(() => ({ check: vi.fn(), person: vi.fn(), assets: vi.fn(), faces: vi.fn(), people: vi.fn(), rename: vi.fn(), review: vi.fn(), refresh: vi.fn().mockResolvedValue({}), back: vi.fn(), committed: vi.fn(), started: vi.fn() }));
const source = { id: "person_source", name: "青岚", revision: 7, assetCount: 1, confirmedFaceCount: 2, etag: '"person-r7"' };
const target = { id: "person_target", name: "白露", revision: 4, assetCount: 3, confirmedFaceCount: 3 };
const asset = { id: "asset_pair", libraryId: "lib_1", name: "合成双人.png", thumbnail: { url: "/synthetic.png" } };
const face = (id: string, ordinal: number, personId = source.id) => ({ faceId: id, assetId: asset.id, ordinal, personId, state: "assigned", revision: ordinal + 2, region: { xPercent: ordinal * 10, yPercent: 10, widthPercent: 20, heightPercent: 30 } });
vi.mock("../queries", () => ({
  usePersonMergeCheckQuery: mocks.check, usePersonQuery: mocks.person, usePersonAssetsQuery: mocks.assets, useAssetFacesQuery: mocks.faces, usePeopleQuery: mocks.people,
  useRenamePersonMutation: () => ({ mutateAsync: mocks.rename }), useApplyFaceReviewMutation: () => ({ mutateAsync: mocks.review }),
}));
function snapshot(data: unknown) { return { data, isPending: false, isError: false, isFetching: false, refetch: mocks.refresh }; }
function show(callbacks = {}) { const client = new QueryClient(); return render(<LocaleProvider><QueryClientProvider client={client}><ToastProvider><MemoryRouter initialEntries={["/intelligence?view=people&person=person_source"]}><PersonPanel {...callbacks} personId={source.id} session={{ csrfToken: "synthetic-csrf", administrator: { id: "admin", username: "fixture", displayName: "测试" }, expiresAt: "2099-01-01T00:00:00Z" }} onBack={mocks.back} /></MemoryRouter></ToastProvider></QueryClientProvider></LocaleProvider>); }
beforeEach(() => {
  vi.clearAllMocks(); window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
  mocks.check.mockReturnValue(snapshot({sourcePersonId:source.id,targetPersonId:target.id,sourceRevision:7,targetRevision:4,canMerge:true,cannotLinkCount:0,duplicateAssignmentCount:0,sourceLimitExceeded:false,examples:[]}));
  mocks.person.mockReturnValue(snapshot(source)); mocks.assets.mockReturnValue(snapshot({ items: [{ asset, faceIds: ["face_1", "face_2"] }], nextCursor: null }));
  mocks.faces.mockReturnValue(snapshot({ items: [face("face_1", 1), face("face_2", 2), face("other_person", 3, target.id)] }));
  mocks.people.mockReturnValue(snapshot({ items: [source, target], nextCursor: null })); mocks.review.mockResolvedValue({}); mocks.rename.mockResolvedValue({});
});
it("requires an explicit face on a multi-face photo and excludes unrelated faces", async () => {
  show(); const user = userEvent.setup();
  expect(screen.getByRole("button", { name: "从人物中移除" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "选择人脸 1" })).toHaveAttribute("aria-pressed", "false");
  expect(screen.queryByRole("button", { name: "选择人脸 3" })).not.toBeInTheDocument();
  await user.selectOptions(screen.getByRole("combobox", { name: "要整理的人脸" }), "face_2");
  await user.click(screen.getByRole("button", { name: "从人物中移除" }));
  const dialog = screen.getByRole("dialog", { name: "从人物中移出这张人脸？" });
  expect(within(dialog).getByText("人脸 2")).toBeVisible(); expect(mocks.review).not.toHaveBeenCalled();
  await user.click(within(dialog).getByRole("button", { name: "确认修改关联" }));
  await waitFor(() => expect(mocks.review).toHaveBeenCalledOnce());
  expect(mocks.review).toHaveBeenCalledWith(expect.objectContaining({ review: { action: "split_face", faceId: "face_2", expectedFaceRevision: 4, sourcePersonId: source.id, expectedSourceRevision: 7 } }));
});
it("cancels a removal without submitting and restores focus", async () => {
  mocks.faces.mockReturnValue(snapshot({ items: [face("face_1", 1)] })); show(); const user = userEvent.setup();
  const opener = screen.getByRole("button", { name: "从人物中移除" }); await user.click(opener);
  await user.click(within(screen.getByRole("dialog", { name: "从人物中移出这张人脸？" })).getByRole("button", { name: "取消" }));
  expect(mocks.review).not.toHaveBeenCalled(); expect(opener).toHaveFocus();
});
it("moves exactly the selected face with source and target revision guards", async () => {
  show(); const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "选择人脸 2" }));
  await user.click(screen.getByRole("button", { name: "移动到人物" })); await user.selectOptions(screen.getByRole("combobox", { name: "目标人物" }), target.id);
  await user.click(screen.getByRole("button", { name: "核对移动" }));
  const dialog = screen.getByRole("dialog", { name: "移动这张人脸？" }); expect(within(dialog).getByText("青岚 → 白露")).toBeVisible();
  await user.click(within(dialog).getByRole("button", { name: "确认修改关联" }));
  expect(mocks.review).toHaveBeenCalledWith(expect.objectContaining({ review: { action: "assign_face", faceId: "face_2", expectedFaceRevision: 4, sourcePersonId: source.id, expectedSourceRevision: 7, personId: target.id, expectedPersonRevision: 4 } }));
});
it("freezes merge revisions, requires acknowledgement and never resends a conflict", async () => {
  mocks.review.mockRejectedValue(new ApiError({ status: 409, code: "face_constraint_conflict", message: "fixture", requestId: undefined }));
  show(); const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "合并人物" }));
  const dialog = screen.getByRole("dialog", { name: "合并人物" }); await user.selectOptions(within(dialog).getByRole("combobox"), target.id);
  expect(within(dialog).getByRole("button", { name: "确认合并" })).toBeDisabled(); expect(mocks.review).not.toHaveBeenCalled();
  await user.click(within(dialog).getByRole("checkbox"));
  await user.click(within(dialog).getByRole("button", { name: "确认合并" }));
  await waitFor(() => expect(screen.getByText("状态已被其他操作更新；已刷新，请重新确认。")).toBeVisible());
  expect(mocks.review).toHaveBeenCalledOnce(); expect(mocks.back).not.toHaveBeenCalled();
});
it("keeps confirmation snapshot despite a background source refresh", async () => {
  show(); const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "合并人物" }));
  const dialog = screen.getByRole("dialog", { name: "合并人物" }); await user.selectOptions(within(dialog).getByRole("combobox"), target.id);
  mocks.person.mockReturnValue(snapshot({ ...source, revision: 9 }));
  await user.click(within(dialog).getByRole("checkbox"));
  await user.click(within(dialog).getByRole("button", { name: "确认合并" }));
  expect(mocks.review).toHaveBeenCalledWith(expect.objectContaining({ review: { action: "merge_people", sourcePersonId: source.id, targetPersonId: target.id, expectedSourceRevision: 7, expectedTargetRevision: 4, conflictsAcknowledged: true } }));
});
it.each(["precondition_failed", "network_error"] as const)("shows truthful %s failure and does not retry or navigate away", async code => {
  mocks.review.mockRejectedValue(new ApiError({ status: code === "network_error" ? 0 : 412, code, message: "fixture", requestId: undefined }));
  mocks.faces.mockReturnValue(snapshot({ items: [face("face_1", 1)] })); show(); const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "从人物中移除" })); await user.click(screen.getByRole("button", { name: "确认修改关联" }));
  await waitFor(() => expect(screen.getByText(code === "network_error" ? "未确认操作是否保存；已刷新，请核对当前状态后再重试。" : "状态已被其他操作更新；已刷新，请重新确认。")).toBeVisible());
  expect(mocks.review).toHaveBeenCalledOnce(); expect(mocks.back).not.toHaveBeenCalled();
});
it("blocks double submission while the exact action is pending", async () => {
  let done!: (value: object) => void; mocks.review.mockImplementation(() => new Promise(resolve => { done = resolve; }));
  mocks.faces.mockReturnValue(snapshot({ items: [face("face_1", 1)] })); show(); const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "从人物中移除" })); const submit = screen.getByRole("button", { name: "确认修改关联" }); await user.dblClick(submit);
  expect(mocks.review).toHaveBeenCalledOnce(); expect(submit).toBeDisabled();
  await act(async () => done({}));
});

it("recovers a committed merge with the same key even when the source becomes unavailable", async () => {
  mocks.review.mockImplementationOnce(async () => { mocks.person.mockReturnValue({ ...snapshot(undefined), isError: true, error: new ApiError({ code: "person_not_found", status: 404, message: "merged synthetic source", requestId: undefined }) }); throw new Error("lost response after commit"); }).mockResolvedValueOnce({ action: "merge_people", reviewId: "review_replayed", revision: 3, undoable: true });
  show({ onReviewCommitted: mocks.committed, onReviewStart: mocks.started }); const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "合并人物" })); const dialog = screen.getByRole("dialog", { name: "合并人物" }); await user.selectOptions(within(dialog).getByRole("combobox"), target.id); await user.click(within(dialog).getByRole("checkbox")); await user.click(within(dialog).getByRole("button", { name: "确认合并" }));
  await waitFor(() => expect(within(dialog).getByText(/保存结果尚未确认/)).toBeVisible()); expect(mocks.committed).not.toHaveBeenCalled(); expect(mocks.back).not.toHaveBeenCalled(); expect(within(dialog).queryByRole("combobox")).not.toBeInTheDocument();
  await user.keyboard("{Escape}"); await user.click(screen.getByRole("button", { name: "继续核对上次操作" })); await user.click(within(dialog).getByRole("button", { name: "确认合并" }));
  await waitFor(() => expect(mocks.back).toHaveBeenCalledOnce()); expect(mocks.review.mock.calls[1]?.[0]).toEqual(mocks.review.mock.calls[0]?.[0]); expect(mocks.committed).toHaveBeenCalledWith(expect.objectContaining({ result: expect.objectContaining({ reviewId: "review_replayed" }), source: source.name, target: target.name }));
});
it("retains exact removal intent after its photo disappears and blocks new edits until resolved", async () => {
  mocks.faces.mockReturnValue(snapshot({ items: [face("face_1", 1)] })); mocks.review.mockImplementationOnce(async () => { mocks.assets.mockReturnValue(snapshot({ items: [], nextCursor: null })); mocks.person.mockReturnValue(snapshot({ ...source, revision: 99 })); throw new Error("lost committed split response"); }); show(); const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "从人物中移除" })); await user.click(screen.getByRole("button", { name: "确认修改关联" })); await waitFor(() => expect(screen.getByText(/保存结果尚未确认/)).toBeVisible()); await user.keyboard("{Escape}"); expect(screen.getByRole("button", { name: "合并人物" })).toBeDisabled(); expect(screen.getByLabelText("管理员填写的名称")).toBeDisabled(); await user.click(screen.getByRole("button", { name: "继续核对上次操作" })); expect(within(screen.getByRole("dialog")).getByText("合成双人.png")).toBeVisible(); await user.click(screen.getByRole("button", { name: "确认修改关联" })); expect(mocks.review.mock.calls[1]?.[0]).toEqual(mocks.review.mock.calls[0]?.[0]); expect(mocks.review.mock.calls[1]?.[0].review.expectedSourceRevision).toBe(7);
});
it("retries an uncertain move without adopting new target or source revisions", async () => {
  mocks.review.mockRejectedValueOnce(new Error("lost move response")); show(); const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "选择人脸 2" })); await user.click(screen.getByRole("button", { name: "移动到人物" })); await user.selectOptions(screen.getByRole("combobox", { name: "目标人物" }), target.id); await user.click(screen.getByRole("button", { name: "核对移动" })); await user.click(screen.getByRole("button", { name: "确认修改关联" })); await waitFor(() => expect(screen.getByText(/保存结果尚未确认/)).toBeVisible()); mocks.people.mockReturnValue(snapshot({ items: [{ ...target, revision: 99 }], nextCursor: null })); mocks.person.mockReturnValue(snapshot({ ...source, revision: 88 })); await user.click(screen.getByRole("button", { name: "确认修改关联" })); expect(mocks.review.mock.calls[1]?.[0]).toEqual(mocks.review.mock.calls[0]?.[0]); expect(mocks.review.mock.calls[1]?.[0].review).toMatchObject({ expectedSourceRevision: 7, expectedPersonRevision: 4 });
});
it("discarding local recovery does not submit, cancel or claim a server result", async () => {
  mocks.review.mockRejectedValue(new Error("unknown")); mocks.faces.mockReturnValue(snapshot({ items: [face("face_1", 1)] })); show(); const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "从人物中移除" })); await user.click(screen.getByRole("button", { name: "确认修改关联" })); await waitFor(() => expect(screen.getByText(/保存结果尚未确认/)).toBeVisible()); await user.keyboard("{Escape}"); await user.click(screen.getByRole("button", { name: "收起本页恢复提示" })); expect(mocks.review).toHaveBeenCalledOnce(); expect(screen.queryByRole("button", { name: "继续核对上次操作" })).not.toBeInTheDocument(); expect(screen.getByRole("button", { name: "合并人物" })).toBeEnabled();
});
it.each(["person_not_found", "face_not_found", "idempotency_conflict"] as const)("ends a known %s without retaining an automatic retry", async code => {
  mocks.review.mockRejectedValue(new ApiError({ code, status: 409, message: "fixture", requestId: undefined })); mocks.faces.mockReturnValue(snapshot({ items: [face("face_1", 1)] })); show(); const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "从人物中移除" })); await user.click(screen.getByRole("button", { name: "确认修改关联" })); await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument()); expect(mocks.review).toHaveBeenCalledOnce(); expect(screen.queryByRole("button", { name: "继续核对上次操作" })).not.toBeInTheDocument();
});
const allowedCheck={sourcePersonId:source.id,targetPersonId:target.id,sourceRevision:7,targetRevision:4,canMerge:true,cannotLinkCount:0,duplicateAssignmentCount:0,sourceLimitExceeded:false,examples:[]};
async function openMerge(){const user=userEvent.setup();await user.click(screen.getByRole("button",{name:"合并人物"}));const dialog=screen.getByRole("dialog",{name:"合并人物"});await user.selectOptions(within(dialog).getByRole("combobox"),target.id);return{user,dialog};}
it("explains cannot-link constraints with bounded photo links and never submits",async()=>{
 mocks.check.mockReturnValue(snapshot({...allowedCheck,canMerge:false,cannotLinkCount:5,examples:[{sourceLibraryId:"lib_1",sourceAssetId:"asset_pair",targetLibraryId:"lib_2",targetAssetId:"asset_other"}]}));show();const{dialog}=await openMerge();expect(within(dialog).getByText("有 5 对人脸被标记为不同人物。")).toBeVisible();expect(within(dialog).getByText(/查看这 1 组照片/)).toBeVisible();expect(within(dialog).getByRole("link",{name:"核对目标照片（新页）"})).toHaveAttribute("href","/libraries/lib_2/media/asset_other");expect(within(dialog).getByRole("checkbox")).toBeDisabled();expect(within(dialog).getByRole("button",{name:"确认合并"})).toBeDisabled();expect(mocks.review).not.toHaveBeenCalled();
});
it.each([{...allowedCheck,canMerge:false,cannotLinkCount:1},{...allowedCheck,canMerge:false,duplicateAssignmentCount:2},{...allowedCheck,canMerge:false,sourceLimitExceeded:true},{...allowedCheck,sourceRevision:99}])("does not adopt changed versions or allow blocked preview %j",async value=>{
 mocks.check.mockReturnValue(snapshot(value));show();const{dialog}=await openMerge();expect(within(dialog).getByRole("checkbox")).toBeDisabled();expect(within(dialog).getByRole("button",{name:"确认合并"})).toBeDisabled();expect(mocks.review).not.toHaveBeenCalled();
});
it("pauses confirmation while checks are unavailable and retry is read-only",async()=>{
 mocks.check.mockReturnValue({...snapshot(undefined),isError:true,error:new ApiError({status:503,code:"internal_error",message:"synthetic",requestId:undefined})});show();const{user,dialog}=await openMerge();expect(within(dialog).getByText(/合并检查暂不可用/)).toBeVisible();expect(within(dialog).getByRole("button",{name:"确认合并"})).toBeDisabled();await user.click(within(dialog).getByRole("button",{name:"重新尝试"}));expect(mocks.refresh).toHaveBeenCalled();expect(mocks.review).not.toHaveBeenCalled();
});
