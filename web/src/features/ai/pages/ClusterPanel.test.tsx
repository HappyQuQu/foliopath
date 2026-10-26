import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ApiError } from "../../../lib/api/errors";
import { ClusterPanel } from "./ClusterPanel";

const mocks = vi.hoisted(() => ({ states: vi.fn(), cluster: vi.fn(), people: vi.fn(), person: vi.fn(), create: vi.fn(), review: vi.fn(), refresh: vi.fn().mockResolvedValue({}), back: vi.fn() }));
const target = { id: "person_target", name: "青岚", revision: 4, confirmedFaceCount: 0, assetCount: 0 };
const members = [1, 2, 3].map(n => ({ faceId: `face_${n}`, assetId: n === 3 ? "asset_2" : "asset_1", revision: n + 2, kind: "core", region: { xPercent: n * 10, yPercent: 20, widthPercent: 20, heightPercent: 30 } }));
const data = { cluster: { id: "cluster_1", kind: "core", memberCount: 150 }, items: members, nextCursor: "next" };
const snapshot = (data: unknown) => ({ data, isPending: false, isError: false, isFetching: false, refetch: mocks.refresh });
vi.mock("../queries", () => ({ useFaceClusterQuery: mocks.cluster, useAssetFacesQueries: mocks.states, usePeopleQuery: mocks.people, usePersonQuery: mocks.person, useCreatePersonMutation: () => ({ mutateAsync: mocks.create }), useApplyFaceReviewMutation: () => ({ mutateAsync: mocks.review }) }));
function panel() { return <LocaleProvider><MemoryRouter><ClusterPanel libraryId="lib_1" clusterId="cluster_1" session={{ csrfToken: "synthetic", expiresAt: "2099-01-01T00:00:00Z", administrator: { id: "admin", username: "fixture", displayName: "Synthetic" } }} onBack={mocks.back} /></MemoryRouter></LocaleProvider>; }
function show() { return render(panel()); }
beforeEach(() => { vi.clearAllMocks(); window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}'); mocks.states.mockImplementation((ids: string[]) => ids.map(id => snapshot({ items: members.filter(m => m.assetId === id).map(m => ({ ...m, state: "anonymous_core", personId: null })) }))); mocks.cluster.mockReturnValue(snapshot(data)); mocks.people.mockReturnValue(snapshot({ items: [target], nextCursor: null })); mocks.person.mockImplementation(id => snapshot({ ...target, id })); mocks.review.mockResolvedValue({ revision: 5 }); mocks.create.mockResolvedValue(target); });

it("shows each photo once, counts faces truthfully and requires an explicit single choice", async () => {
  show(); expect(screen.getAllByRole("article")).toHaveLength(2); expect(screen.getAllByRole("img")).toHaveLength(2); expect(screen.getAllByRole("radio")).toHaveLength(3);
  expect(screen.getByText("此组共 150 张人脸")).toBeVisible(); expect(screen.getByText("本页 3 张人脸 · 2 张照片")).toBeVisible();
  expect(screen.getByRole("button", { name: "确认归入人物" })).toBeDisabled(); expect(screen.getByRole("button", { name: "排除所选人脸" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "照片 1 · 标记 2" })); expect(screen.getByRole("radio", { name: "照片 1 · 标记 2" })).toBeChecked();
  await userEvent.click(screen.getByRole("radio", { name: "照片 2 · 标记 1" })); expect(screen.getByRole("radio", { name: "照片 1 · 标记 2" })).not.toBeChecked();
});
it("cancels without writing and freezes exact revisions before a confirmed assignment", async () => {
  show(); const user = userEvent.setup(); await user.click(screen.getByRole("radio", { name: "照片 1 · 标记 2" })); await user.selectOptions(screen.getByRole("combobox", { name: "目标人物" }), target.id);
  const trigger = screen.getByRole("button", { name: "确认归入人物" }); await user.click(trigger);
  const dialog = screen.getByRole("dialog", { name: "确认这张人脸属于所选人物？" }); await user.click(within(dialog).getByRole("button", { name: "取消" })); expect(mocks.review).not.toHaveBeenCalled(); expect(trigger).toHaveFocus();
  await user.click(trigger); mocks.person.mockReturnValue(snapshot({ ...target, revision: 99 })); await user.click(within(dialog).getByRole("button", { name: "确认保存" }));
  expect(mocks.review).toHaveBeenCalledWith(expect.objectContaining({ review: { action: "assign_face", faceId: "face_2", expectedFaceRevision: 4, personId: target.id, expectedPersonRevision: 4 } }));
  await waitFor(() => expect(screen.getByText(/所选人脸已保存/)).toBeVisible()); expect(screen.getAllByRole("radio").every(r => !(r as HTMLInputElement).checked)).toBe(true);
});
it("excludes only one face, without looping through other members", async () => {
  show(); const user = userEvent.setup(); await user.click(screen.getByRole("radio", { name: "照片 2 · 标记 1" })); await user.click(screen.getByRole("button", { name: "排除所选人脸" })); await user.click(screen.getByRole("button", { name: "确认保存" }));
  expect(mocks.review).toHaveBeenCalledOnce(); expect(mocks.review).toHaveBeenCalledWith(expect.objectContaining({ review: { action: "exclude_face", faceId: "face_3", expectedFaceRevision: 5 } }));
});
it("creates an empty person separately and retries uncertain creation with the same intent key", async () => {
  mocks.create.mockRejectedValueOnce(new Error("unknown response")); show(); const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "新建人物" })); const dialog = screen.getByRole("dialog", { name: "新建人物" });
  await user.type(within(dialog).getByRole("textbox", { name: "管理员填写的名称" }), "青岚"); const submit = within(dialog).getByRole("button", { name: "新建人物" }); await user.click(submit);
  await waitFor(() => expect(submit).toBeEnabled()); expect(mocks.create).toHaveBeenCalledOnce(); expect(mocks.review).not.toHaveBeenCalled();
  await user.click(submit); expect(mocks.create.mock.calls[0]![0]).toEqual(mocks.create.mock.calls[1]![0]); expect(mocks.create.mock.calls[0]![0]).not.toHaveProperty("sourceCluster");
  await waitFor(() => expect(screen.getByText(/尚未归类人脸/)).toBeVisible()); expect(mocks.review).not.toHaveBeenCalled();
});
it("keeps an unknown response retry explicit and idempotent", async () => {
  mocks.review.mockRejectedValueOnce(new Error("response lost")); show(); const user = userEvent.setup(); await user.click(screen.getByRole("radio", { name: "照片 1 · 标记 1" })); await user.click(screen.getByRole("button", { name: "排除所选人脸" })); const dialog = screen.getByRole("dialog", { name: "排除这张人脸？" }); const submit = within(dialog).getByRole("button", { name: "确认保存" });
  await user.click(submit); await waitFor(() => expect(within(dialog).getByText(/未确认操作是否保存/)).toBeVisible()); expect(mocks.review).toHaveBeenCalledOnce(); await user.click(submit);
  expect(mocks.review.mock.calls[0]![0]).toEqual(mocks.review.mock.calls[1]![0]);
});
it("does not retry conflicts and preserves a saved result when the group subsequently disappears", async () => {
  mocks.review.mockRejectedValueOnce(new ApiError({ status: 412, code: "precondition_failed", message: "fixture", requestId: undefined })); const mounted = show(); const user = userEvent.setup(); await user.click(screen.getByRole("radio", { name: "照片 1 · 标记 1" })); await user.click(screen.getByRole("button", { name: "排除所选人脸" })); await user.click(screen.getByRole("button", { name: "确认保存" }));
  await waitFor(() => expect(screen.getByText(/状态已变化，未重新提交/)).toBeVisible()); expect(mocks.review).toHaveBeenCalledOnce();
  await user.click(screen.getByRole("button", { name: "排除所选人脸" })); await user.click(screen.getByRole("button", { name: "确认保存" }));
  await waitFor(() => expect(screen.getByText(/所选人脸已保存/)).toBeVisible());
  mocks.cluster.mockReturnValue({ ...snapshot(undefined), isError: true, error: new ApiError({ status: 409, code: "face_cursor_stale", message: "fixture", requestId: undefined }) });
  mounted.rerender(panel());
  expect(screen.getByText(/所选人脸已保存/)).toBeVisible(); expect(screen.getByRole("button", { name: "返回列表" })).toBeVisible();
});

it("does not allow already assigned, excluded, mismatched or unavailable faces to be submitted", async () => {
  mocks.states.mockReturnValue([{ ...snapshot({ items: [{ ...members[0], state: "assigned" }, { ...members[1], state: "excluded" }] }) }, { ...snapshot(undefined), isError: true, error: new Error("offline") }]);
  show(); expect(screen.getByText("已归类")).toBeVisible(); expect(screen.getByText("已排除")).toBeVisible(); expect(screen.getAllByRole("radio").every(r => (r as HTMLInputElement).disabled)).toBe(true);
  expect(screen.getByRole("button", { name: "排除所选人脸" })).toBeDisabled(); expect(mocks.review).not.toHaveBeenCalled();
});
it("blocks double submission while a single review is pending", async () => {
  let finish!: (v: object) => void; mocks.review.mockImplementation(() => new Promise(resolve => { finish = resolve; }));
  show(); const user = userEvent.setup(); await user.click(screen.getByRole("radio", { name: "照片 1 · 标记 1" })); await user.click(screen.getByRole("button", { name: "排除所选人脸" })); const submit = screen.getByRole("button", { name: "确认保存" });
  await user.dblClick(submit); expect(mocks.review).toHaveBeenCalledOnce(); expect(submit).toBeDisabled(); await act(async () => finish({ revision: 5 }));
});
it("uses a refreshed target revision for the next separately confirmed face", async () => {
  mocks.review.mockImplementation(async () => { mocks.person.mockReturnValue(snapshot({ ...target, revision: 5 })); return { revision: 5 }; });
  show(); const user = userEvent.setup(); await user.selectOptions(screen.getByRole("combobox", { name: "目标人物" }), target.id);
  await user.click(screen.getByRole("radio", { name: "照片 1 · 标记 1" })); await user.click(screen.getByRole("button", { name: "确认归入人物" })); await user.click(screen.getByRole("button", { name: "确认保存" })); await waitFor(() => expect(screen.getByText(/所选人脸已保存/)).toBeVisible());
  await user.click(screen.getByRole("radio", { name: "照片 1 · 标记 2" })); await user.click(screen.getByRole("button", { name: "确认归入人物" })); await user.click(screen.getByRole("button", { name: "确认保存" }));
  expect(mocks.review.mock.calls[0]![0].review.expectedPersonRevision).toBe(4); expect(mocks.review.mock.calls[1]![0].review.expectedPersonRevision).toBe(5);
});
