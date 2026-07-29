import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useParams,
} from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import type { Asset } from "../../../lib/api/catalog";
import { getAsset } from "../../../lib/api/catalog";
import { ApiError } from "../../../lib/api/errors";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { MediaViewerPage } from "./MediaViewerPage";

vi.mock("../../../lib/api/catalog", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/catalog")>();
  return { ...actual, getAsset: vi.fn() };
});

const photo = (id: string, libraryId = "lib_family"): Asset => ({
  directoryId: "dir_kyoto",
  durationMs: null,
  height: 800,
  id,
  kind: "image",
  libraryId,
  libraryName: "家庭影像",
  mimeType: "image/jpeg",
  modifiedAt: "2026-07-28T00:00:00Z",
  name: `${id}.jpg`,
  playbackStatus: "not_applicable",
  probeStatus: "ready",
  relativePath: `旅行/${id}.jpg`,
  sizeBytes: 1024,
  sourceAvailability: "available",
  storyboard: {
    cellHeight: null,
    cellWidth: null,
    columns: null,
    errorCode: null,
    frameCount: null,
    rows: null,
    status: "not_applicable",
    url: null,
  },
  thumbnail: { errorCode: null, status: "ready", url: "/thumbnail" },
  width: 1200,
});

beforeEach(() => {
  vi.mocked(getAsset).mockReset();
  vi.mocked(getAsset).mockImplementation(async (id) => photo(id));
});

it("moves through the loaded source sequence and returns focus context", async () => {
  const user = userEvent.setup();
  renderViewer({
    pathname: "/libraries/lib_family/media/first",
    search: "?from=%2Fsearch%3Fq%3Dkyoto",
    state: {
      returnTo: "/search?q=kyoto",
      sequence: [
        { id: "first", libraryId: "lib_family" },
        { id: "second", libraryId: "lib_family" },
      ],
    },
  });

  expect(await screen.findByRole("img", { name: "first.jpg" })).toBeVisible();
  expect(screen.getByText("Item 1 of 2")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "Next item" }));
  expect(await screen.findByRole("img", { name: "second.jpg" })).toBeVisible();
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/media/second?from=%2Fsearch%3Fq%3Dkyoto",
  );

  await user.click(screen.getByRole("button", { name: "Close" }));
  await waitFor(() =>
    expect(screen.getByTestId("location")).toHaveTextContent("/search?q=kyoto"),
  );
  expect(screen.getByTestId("location")).toHaveTextContent(
    '"restoreFocusAssetId":"second"',
  );
});

it("falls back to the asset library browse route for an unsafe direct return", async () => {
  const user = userEvent.setup();
  renderViewer({
    pathname: "/libraries/lib_family/media/first",
    search: "?from=%2F%2Fevil.example%2Fsearch",
  });

  expect(await screen.findByRole("img", { name: "first.jpg" })).toBeVisible();
  expect(screen.getByRole("button", { name: "Previous item" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Next item" })).toBeDisabled();
  await user.click(screen.getByRole("button", { name: "Close" }));
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/browse",
  );
});

it("renders source-offline metadata as a recoverable viewer state", async () => {
  vi.mocked(getAsset).mockResolvedValue({
    ...photo("offline"),
    sourceAvailability: "offline",
  });
  renderViewer({
    pathname: "/libraries/lib_family/media/offline",
  });

  expect(
    await screen.findByRole("heading", { name: "Library is offline" }),
  ).toBeVisible();
  expect(screen.getByRole("button", { name: "Check again" })).toBeVisible();
  expect(screen.getByRole("button", { name: "Close" })).toBeVisible();
});

it("keeps viewer chrome for an asset removed from the index", async () => {
  vi.mocked(getAsset).mockRejectedValue(
    new ApiError({
      code: "asset_not_found",
      message: "not found",
      requestId: undefined,
      status: 404,
    }),
  );
  renderViewer({
    pathname: "/libraries/lib_family/media/deleted",
  });

  expect(
    await screen.findByRole("heading", {
      name: "Media removed from the index",
    }),
  ).toBeVisible();
  expect(screen.queryByRole("button", { name: "Check again" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Close" })).toBeVisible();
});

function renderViewer(initialEntry: {
  pathname: string;
  search?: string;
  state?: unknown;
}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <LocaleProvider>
        <MemoryRouter initialEntries={[initialEntry]}>
          <Routes>
            <Route
              path="/libraries/:libraryId/media/:assetId"
              element={<ViewerRoute />}
            />
            <Route path="*" element={null} />
          </Routes>
          <LocationProbe />
        </MemoryRouter>
      </LocaleProvider>
    </QueryClientProvider>,
  );
}

function ViewerRoute() {
  const { assetId = "", libraryId = "" } = useParams();
  return <MediaViewerPage assetId={assetId} libraryId={libraryId} />;
}

function LocationProbe() {
  const location = useLocation();
  return (
    <output data-testid="location">
      {location.pathname}
      {location.search}
      {JSON.stringify(location.state)}
    </output>
  );
}
