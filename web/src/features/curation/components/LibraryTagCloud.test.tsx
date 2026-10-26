import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation, useNavigate } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { listTags, type Tag } from "../../../lib/api/curation";
import { ApiError } from "../../../lib/api/errors";
import { writeTagLanguagePreference } from "../../../lib/storage/preferences";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { LibraryTagCloud } from "./LibraryTagCloud";

vi.mock("../../../lib/api/curation", async importOriginal => ({
  ...await importOriginal<typeof import("../../../lib/api/curation")>(), listTags: vi.fn(),
}));

const tag = (id: string, assetCount = 1): Tag => ({ id, name: id, assetCount, createdAt: "2026-10-22T00:00:00Z", updatedAt: "2026-10-22T00:00:00Z" });
function Location() {
  const location = useLocation();
  const navigate = useNavigate();
  return <><output data-testid="location">{location.pathname}{location.search}</output><button onClick={() => navigate(-1)}>Browser back</button></>;
}
function setup(route = "/tags?libraryId=lib_one", selectedTagId?: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}><LocaleProvider><MemoryRouter initialEntries={[route]}>
    <LibraryTagCloud libraryId="lib_one" {...(selectedTagId ? { selectedTagId } : {})} /><Location />
  </MemoryRouter></LocaleProvider></QueryClientProvider>);
  return userEvent.setup();
}
beforeEach(() => { vi.mocked(listTags).mockReset(); });

it("shows distinct photo counts in a consistent list and image-scoped links", async () => {
  vi.mocked(listTags).mockResolvedValue({ items: [tag("small", 2), tag("medium", 10), tag("large", 200)], nextCursor: null });
  setup(undefined, "medium");
  expect(await screen.findByRole("link", { name: /large/ })).toHaveAttribute("data-weight", "5");
  expect(screen.getByRole("link", { name: /small/ })).toHaveAttribute("data-weight", "1");
  const medium = screen.getByRole("link", { name: /medium/ });
  expect(medium).toHaveAttribute("data-weight", "3");
  expect(medium).not.toHaveAttribute("data-frequency");
  expect(medium).toHaveAttribute("aria-current", "page");
  expect(medium).toHaveAttribute("href", "/tags/medium?libraryId=lib_one&kind=image");
  expect(screen.getByLabelText(/200 photos|200 张图片/)).toBeVisible();
  expect(listTags).toHaveBeenCalledWith(undefined, { libraryId: "lib_one", kinds: ["image", "animated"] });
});

it("renders only one bounded page and supports next/previous without mixing pages", async () => {
  vi.mocked(listTags).mockImplementation(async cursor => cursor
    ? { items: [tag("second", 50), tag("same", 10)], nextCursor: null }
    : { items: [tag("first", 200), tag("same", 10)], nextCursor: "next-token" });
  const user = setup();
  expect(await screen.findByRole("link", { name: /same/ })).toBeVisible();
  await user.click(screen.getByRole("button", { name: /Next page|下一页/ }));
  expect(await screen.findByRole("link", { name: /second/ })).toBeVisible();
  expect(screen.queryByRole("link", { name: /first/ })).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: /same/ })).toBeVisible();
  expect(screen.getByTestId("location")).toHaveTextContent("tagCursor=next-token");
  await user.click(screen.getByRole("button", { name: /Previous page|上一页/ }));
  expect(await screen.findByRole("link", { name: /first/ })).toBeVisible();
  expect(screen.queryByRole("link", { name: /second/ })).not.toBeInTheDocument();
});

it("submits bounded search and resets the cursor; clearing search returns to the library", async () => {
  vi.mocked(listTags).mockResolvedValue({ items: [tag("旅行", 3)], nextCursor: "another" });
  const user = setup("/tags?libraryId=lib_one&tagCursor=old");
  const input = screen.getByRole("searchbox", { name: /Search photo tags|搜索图片标签/ });
  expect(input).toHaveAttribute("maxLength", "32");
  await user.type(input, "旅行");
  await user.click(screen.getByRole("button", { name: /Find|查找/ }));
  await waitFor(() => expect(listTags).toHaveBeenLastCalledWith(undefined, { libraryId: "lib_one", kinds: ["image", "animated"], q: "旅行" }));
  expect(screen.getByTestId("location")).not.toHaveTextContent("tagCursor");
  await user.click(screen.getByRole("button", { name: /Clear search|清除搜索/ }));
  await waitFor(() => expect(listTags).toHaveBeenLastCalledWith(undefined, { libraryId: "lib_one", kinds: ["image", "animated"] }));
});

it("keeps page predecessors aligned with URL history after browser back", async () => {
  vi.mocked(listTags).mockImplementation(async cursor => {
    if (cursor === "page3") return { items: [tag("third")], nextCursor: null };
    if (cursor === "page2") return { items: [tag("second")], nextCursor: "page3" };
    return { items: [tag("first")], nextCursor: "page2" };
  });
  const user = setup();
  await screen.findByRole("link", { name: /first/ });
  await user.click(screen.getByRole("button", { name: /Next page|下一页/ }));
  await screen.findByRole("link", { name: /second/ });
  await user.click(screen.getByRole("button", { name: /Next page|下一页/ }));
  await screen.findByRole("link", { name: /third/ });
  await user.click(screen.getByRole("button", { name: "Browser back" }));
  await screen.findByRole("link", { name: /second/ });
  await user.click(screen.getByRole("button", { name: /Previous page|上一页/ }));
  expect(await screen.findByRole("link", { name: /first/ })).toBeVisible();
  expect(screen.getByTestId("location")).not.toHaveTextContent("tagCursor");
});

it("offers first page instead of claiming a previous page for an unknown deep-link predecessor", async () => {
  vi.mocked(listTags).mockResolvedValue({ items: [tag("linked")], nextCursor: null });
  const user = setup("/tags?libraryId=lib_one&tagCursor=deep");
  await screen.findByRole("link", { name: /linked/ });
  expect(screen.queryByRole("button", { name: /Previous page|上一页/ })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: /First page|返回第一页/ }));
  expect(screen.getByTestId("location")).not.toHaveTextContent("tagCursor");
});

it("offers first-page recovery after revision invalidates a cursor without retry loops", async () => {
  vi.mocked(listTags).mockImplementation(async cursor => {
    if (cursor) throw new ApiError({ code: "invalid_cursor", message: "Expired", status: 400, requestId: undefined });
    return { items: [tag("current")], nextCursor: null };
  });
  const user = setup("/tags?libraryId=lib_one&tagCursor=expired");
  expect(await screen.findByRole("alert")).toHaveTextContent(/Tags changed|标签已|标签发生/);
  expect(listTags).toHaveBeenCalledTimes(1);
  await user.click(screen.getByRole("button", { name: /First page|返回第一页/ }));
  expect(await screen.findByRole("link", { name: /current/ })).toBeVisible();
  expect(listTags).toHaveBeenCalledTimes(2);
});

it("does not misrepresent a network failure as an empty library", async () => {
  vi.mocked(listTags).mockRejectedValue(new Error("offline"));
  setup();
  expect(await screen.findByRole("alert")).toHaveTextContent(/could not be loaded|无法载入/);
  expect(screen.queryByText(/No photo tags yet|暂无图片标签/)).not.toBeInTheDocument();
});

it("explains empty data with manual editing and opt-in automatic tagging", async () => {
  vi.mocked(listTags).mockResolvedValue({ items: [], nextCursor: null });
  setup();
  expect(await screen.findByText(/enable automatic tags|为此库开启自动标签/)).toBeVisible();
});

it("separates bilingual labels without translating manual names", async () => {
  writeTagLanguagePreference("bilingual");
  vi.mocked(listTags).mockResolvedValue({ items: [
    { ...tag("car", 200), localization: { key: "car", zhCN: "汽车", en: "Car" } },
    tag("manual / 原文", 1),
  ], nextCursor: null });
  setup();
  const car = await screen.findByRole("link", { name: /汽车.*Car/ });
  expect(car.querySelector('[lang="en"]')).toHaveTextContent("Car");
  expect(screen.getByRole("link", { name: /manual \/ 原文/ })).toHaveTextContent("manual / 原文");
});
