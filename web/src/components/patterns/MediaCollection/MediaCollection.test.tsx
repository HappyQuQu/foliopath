import { createRef } from "react";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import {
  MediaCollection,
  mediaCollectionCapacityBudget,
  shouldLoadNextMediaPage,
  storyboardPlaybackTiming,
  type MediaCollectionHandle,
  type MediaCollectionItem,
} from "./MediaCollection";

const labels = {
  activatePreview: "Preview {name}",
  animated: "Animated",
  failedThumbnail: "Thumbnail failed",
  image: "Image",
  loadMore: "Load more",
  loadMoreFailed: "More media could not be loaded.",
  loadingMore: "Loading more",
  pendingThumbnail: "Preparing thumbnail",
  previewing: "Currently previewing",
  retryLoadMore: "Retry loading more",
  unavailableThumbnail: "Thumbnail unavailable",
  video: "Video",
};

const storyboardItem: MediaCollectionItem = {
  height: 1080,
  id: "video-storyboard",
  kind: "video",
  modifiedLabel: "Today",
  name: "video-storyboard.mp4",
  storyboard: {
    cellHeight: 180,
    cellWidth: 320,
    columns: 5,
    frameCount: 10,
    rows: 2,
    url: "/storyboard.webp",
  },
  thumbnailStatus: "ready",
  thumbnailUrl: "/poster.webp",
  width: 1920,
};

beforeEach(() => {
  Object.defineProperty(document, "visibilityState", {
    configurable: true,
    value: "visible",
  });
  Object.defineProperty(HTMLImageElement.prototype, "decode", {
    configurable: true,
    value: vi.fn().mockResolvedValue(undefined),
  });
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  Reflect.deleteProperty(HTMLImageElement.prototype, "decode");
});

it("renders a bounded virtual window in query order with stable media identity", () => {
  const items: MediaCollectionItem[] = Array.from({ length: 200 }, (_, index) => ({
    height: 900,
    id: `asset-${index}`,
    kind: index % 3 === 0 ? "video" : "image",
    modifiedLabel: `July ${index + 1}`,
    name: `asset-${index}.jpg`,
    thumbnailStatus: index === 0 ? "ready" : "pending",
    thumbnailUrl: index === 0 ? "/thumbnail-0.webp" : null,
    width: 1200,
  }));

  render(
    <MediaCollection
      hasNextPage
      isFetchingNextPage={false}
      items={items}
      labels={labels}
      layout="grid"
      onLoadMore={vi.fn()}
    />,
  );

  const renderedItems = screen.getAllByRole("listitem");
  expect(renderedItems.length).toBeGreaterThan(0);
  expect(renderedItems.length).toBeLessThan(200);
  expect(renderedItems[0]).toHaveAttribute("aria-posinset", "1");
  expect(
    screen
      .getByRole("article", { name: "asset-0.jpg · Video" })
      .querySelector("img"),
  ).toHaveAttribute("src", "/thumbnail-0.webp");
  expect(screen.getByRole("button", { name: "Load more" })).toBeVisible();
});

it(
  "keeps the primary 100k tier in a bounded DOM and does not prefetch from the first window",
  () => {
    const loadMore = vi.fn();
    const items: MediaCollectionItem[] = Array.from(
      { length: mediaCollectionCapacityBudget.primaryTierItems },
      (_, index) => ({
        height: 900,
        id: `capacity-${index}`,
        kind: index % 7 === 0 ? "video" : "image",
        modifiedLabel: "Today",
        name: `capacity-${index}.jpg`,
        thumbnailStatus: "pending",
        thumbnailUrl: null,
        width: 1200,
      }),
    );

    render(
      <MediaCollection
        hasNextPage
        isFetchingNextPage={false}
        items={items}
        labels={labels}
        layout="grid"
        onLoadMore={loadMore}
      />,
    );

    const mountedItems = screen.getAllByRole("listitem");
    expect(mountedItems.length).toBeLessThanOrEqual(64);
    expect(mountedItems.length).toBeLessThan(items.length / 1_000);
    expect(mountedItems[0]).toHaveAttribute("aria-setsize", "100000");
    expect(loadMore).not.toHaveBeenCalled();
  },
  10_000,
);

it("allows one cursor request near the edge and suppresses it while one is in flight", () => {
  expect(
    shouldLoadNextMediaPage({
      columns: 6,
      hasNextPage: true,
      isFetchingNextPage: false,
      itemCount: 100_000,
      lastVirtualIndex: 99_988,
      paginationError: false,
    }),
  ).toBe(true);
  expect(
    shouldLoadNextMediaPage({
      columns: 6,
      hasNextPage: true,
      isFetchingNextPage: true,
      itemCount: 100_000,
      lastVirtualIndex: 99_999,
      paginationError: false,
    }),
  ).toBe(false);
  expect(
    shouldLoadNextMediaPage({
      columns: 6,
      hasNextPage: true,
      isFetchingNextPage: false,
      itemCount: 100_000,
      lastVirtualIndex: 99_999,
      paginationError: true,
    }),
  ).toBe(false);
});

it("uses the virtualizer to restore the anchor for a distant capacity-tier item", () => {
  const scrollTo = vi
    .spyOn(window, "scrollTo")
    .mockImplementation(() => undefined);
  const ref = createRef<MediaCollectionHandle>();
  const items: MediaCollectionItem[] = Array.from(
    { length: mediaCollectionCapacityBudget.primaryTierItems },
    (_, index) => ({
      height: 900,
      id: `restore-${index}`,
      kind: "image",
      modifiedLabel: "Today",
      name: `restore-${index}.jpg`,
      thumbnailStatus: "pending",
      thumbnailUrl: null,
      width: 1200,
    }),
  );

  render(
    <MediaCollection
      ref={ref}
      hasNextPage={false}
      isFetchingNextPage={false}
      items={items}
      labels={labels}
      layout="grid"
      onItemActivate={vi.fn()}
      onLoadMore={vi.fn()}
    />,
  );

  act(() => ref.current?.restoreItem("restore-99999"));
  expect(scrollTo).toHaveBeenCalled();
  scrollTo.mockRestore();
});

it("preserves an original aspect ratio in masonry and exposes placeholder status", () => {
  render(
    <MediaCollection
      hasNextPage={false}
      isFetchingNextPage={false}
      items={[
        {
          height: 1200,
          id: "portrait",
          kind: "image",
          modifiedLabel: "Today",
          name: "portrait.jpg",
          thumbnailStatus: "failed",
          thumbnailUrl: null,
          width: 800,
        },
      ]}
      labels={labels}
      layout="masonry"
      onLoadMore={vi.fn()}
    />,
  );

  expect(screen.getByText("Thumbnail failed")).toBeVisible();
  expect(screen.getByRole("article", { name: "portrait.jpg · Image" })).toBeVisible();
});

it("preserves loaded items and exposes a retry when the next page fails", () => {
  const retry = vi.fn();
  const view = render(
    <MediaCollection
      hasNextPage
      isFetchingNextPage={false}
      items={[
        {
          height: 800,
          id: "ready",
          kind: "image",
          modifiedLabel: "Today",
          name: "ready.jpg",
          thumbnailStatus: "ready",
          thumbnailUrl: "/ready.webp",
          width: 1200,
        },
      ]}
      labels={labels}
      layout="grid"
      onLoadMore={vi.fn()}
      onRetryLoadMore={retry}
      paginationError
    />,
  );

  expect(screen.getByText("ready.jpg")).toBeVisible();
  expect(screen.getByText("More media could not be loaded.")).toBeVisible();
  screen.getByRole("button", { name: "Retry loading more" }).click();
  expect(retry).toHaveBeenCalledOnce();

  view.rerender(
    <MediaCollection
      hasNextPage
      isFetchingNextPage
      items={[
        {
          height: 800,
          id: "ready",
          kind: "image",
          modifiedLabel: "Today",
          name: "ready.jpg",
          thumbnailStatus: "ready",
          thumbnailUrl: "/ready.webp",
          width: 1200,
        },
      ]}
      labels={labels}
      layout="grid"
      onLoadMore={vi.fn()}
      onRetryLoadMore={retry}
      paginationError
    />,
  );
  expect(screen.getByRole("button", { name: "Loading more" })).toBeDisabled();
});

it("activates a media preview without hiding the source-directory link", () => {
  const activate = vi.fn();
  render(
    <MediaCollection
      hasNextPage={false}
      isFetchingNextPage={false}
      items={[
        {
          height: 800,
          id: "ready",
          kind: "image",
          modifiedLabel: "Today",
          name: "ready.jpg",
          sourceHref: "/source",
          sourceLabel: "Source: photos",
          thumbnailStatus: "ready",
          thumbnailUrl: "/ready.webp",
          width: 1200,
        },
      ]}
      labels={labels}
      layout="grid"
      onItemActivate={activate}
      onLoadMore={vi.fn()}
      previewItemId="ready"
      selectedItemId="ready"
    />,
  );

  const trigger = screen.getByRole("button", { name: "Preview ready.jpg" });
  expect(trigger).toHaveAttribute("aria-pressed", "true");
  trigger.click();
  expect(activate).toHaveBeenCalledWith(
    "ready",
    "single",
    expect.any(HTMLButtonElement),
  );
  fireEvent.doubleClick(trigger);
  expect(activate).toHaveBeenLastCalledWith(
    "ready",
    "double",
    expect.any(HTMLButtonElement),
  );
  expect(screen.getByText("Currently previewing")).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Source: photos" })).toBeVisible();
});

it("waits for hover intent and sprite decode before starting the storyboard", async () => {
  vi.useFakeTimers();
  allowStoryboardMotion();
  const decode = vi.mocked(HTMLImageElement.prototype.decode);
  renderStoryboard();

  const card = screen.getByRole("article", {
    name: "video-storyboard.mp4 · Video",
  });
  fireEvent.pointerEnter(card);

  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs - 1),
  );
  expect(decode).not.toHaveBeenCalled();
  expect(card).not.toHaveAttribute("data-storyboard-playing");
  expect(card.querySelector('[src="/poster.webp"]')).toBeInTheDocument();

  await act(() => vi.advanceTimersByTimeAsync(1));
  expect(decode).toHaveBeenCalledOnce();
  expect(card).toHaveAttribute("data-storyboard-playing", "true");
  expect(card.querySelector('[src="/storyboard.webp"]')).toBeInTheDocument();
});

it("cycles sprite frames every 500ms and restores the poster on leave", async () => {
  vi.useFakeTimers();
  allowStoryboardMotion();
  renderStoryboard();

  const card = screen.getByRole("article", {
    name: "video-storyboard.mp4 · Video",
  });
  fireEvent.pointerEnter(card);
  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs),
  );

  const sprite = card.querySelector<HTMLImageElement>('[src="/storyboard.webp"]');
  expect(sprite?.style.getPropertyValue("--storyboard-x")).toBe("10%");
  expect(sprite?.style.getPropertyValue("--storyboard-y")).toBe("25%");
  expect(sprite).toHaveAttribute("data-cover-axis", "height");

  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.frameMs * 6),
  );
  expect(sprite?.style.getPropertyValue("--storyboard-x")).toBe("30%");
  expect(sprite?.style.getPropertyValue("--storyboard-y")).toBe("75%");

  fireEvent.pointerLeave(card);
  expect(card).not.toHaveAttribute("data-storyboard-playing");
  expect(card.querySelector('[src="/storyboard.webp"]')).not.toBeInTheDocument();
  expect(card.querySelector('[src="/poster.webp"]')).toBeInTheDocument();
});

it.each([
  { columns: 4, frameCount: 4, rows: 1 },
  { columns: 5, frameCount: 10, rows: 2 },
] as const)(
  "uses server layout and loops a $frameCount-frame sprite",
  async ({ columns, frameCount, rows }) => {
    vi.useFakeTimers();
    allowStoryboardMotion();
    const item = {
      ...storyboardItem,
      storyboard: {
        ...storyboardItem.storyboard!,
        columns,
        frameCount,
        rows,
      },
    };
    renderStoryboard([item]);
    const card = screen.getByRole("article", {
      name: "video-storyboard.mp4 · Video",
    });
    fireEvent.pointerEnter(card);
    await act(() =>
      vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs),
    );
    const sprite = card.querySelector<HTMLImageElement>(
      '[src="/storyboard.webp"]',
    );

    await act(() =>
      vi.advanceTimersByTimeAsync(
        storyboardPlaybackTiming.frameMs * (frameCount - 1),
      ),
    );
    expect(sprite?.style.getPropertyValue("--storyboard-x")).toBe(
      `${(((frameCount - 1) % columns) + 0.5) / columns * 100}%`,
    );
    expect(sprite?.style.getPropertyValue("--storyboard-y")).toBe(
      `${
        (Math.floor((frameCount - 1) / columns) + 0.5) / rows * 100
      }%`,
    );

    await act(() =>
      vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.frameMs),
    );
    expect(sprite?.style.getPropertyValue("--storyboard-x")).toBe(
      `${0.5 / columns * 100}%`,
    );
    expect(sprite?.style.getPropertyValue("--storyboard-y")).toBe(
      `${0.5 / rows * 100}%`,
    );
  },
);

it("preserves portrait sprite cells when covering a grid card", async () => {
  vi.useFakeTimers();
  allowStoryboardMotion();
  renderStoryboard([
    {
      ...storyboardItem,
      storyboard: {
        ...storyboardItem.storyboard!,
        cellHeight: 320,
        cellWidth: 180,
      },
    },
  ]);
  const card = screen.getByRole("article", {
    name: "video-storyboard.mp4 · Video",
  });
  fireEvent.pointerEnter(card);
  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs),
  );

  expect(card.querySelector('[src="/storyboard.webp"]')).toHaveAttribute(
    "data-cover-axis",
    "width",
  );
});

it("keeps only one active storyboard and cancels playback when the page hides", async () => {
  vi.useFakeTimers();
  allowStoryboardMotion();
  renderStoryboard([
    storyboardItem,
    {
      ...storyboardItem,
      id: "video-storyboard-2",
      name: "video-storyboard-2.mp4",
      storyboard: {
        ...storyboardItem.storyboard!,
        url: "/storyboard-2.webp",
      },
    },
  ]);

  const first = screen.getByRole("article", {
    name: "video-storyboard.mp4 · Video",
  });
  const second = screen.getByRole("article", {
    name: "video-storyboard-2.mp4 · Video",
  });
  fireEvent.pointerEnter(first);
  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs),
  );
  expect(first).toHaveAttribute("data-storyboard-playing", "true");

  fireEvent.pointerEnter(second);
  expect(first).not.toHaveAttribute("data-storyboard-playing");
  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs),
  );
  expect(second).toHaveAttribute("data-storyboard-playing", "true");

  Object.defineProperty(document, "visibilityState", {
    configurable: true,
    value: "hidden",
  });
  fireEvent(document, new Event("visibilitychange"));
  expect(second).not.toHaveAttribute("data-storyboard-playing");
});

it("does not animate for reduced motion or a non-fine pointer", async () => {
  vi.useFakeTimers();
  const decode = vi.mocked(HTMLImageElement.prototype.decode);
  const { rerender } = renderStoryboard(undefined, {
    finePointer: true,
    reducedMotion: true,
  });
  const card = screen.getByRole("article", {
    name: "video-storyboard.mp4 · Video",
  });
  fireEvent.pointerEnter(card);
  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs),
  );
  expect(decode).not.toHaveBeenCalled();

  setStoryboardMotion({ finePointer: false, reducedMotion: false });
  rerender(storyboardCollection([storyboardItem]));
  fireEvent.pointerEnter(card);
  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs),
  );
  expect(decode).not.toHaveBeenCalled();
  expect(card.querySelector('[src="/storyboard.webp"]')).not.toBeInTheDocument();
});

it("does not start a storyboard from keyboard focus", async () => {
  vi.useFakeTimers();
  allowStoryboardMotion();
  const decode = vi.mocked(HTMLImageElement.prototype.decode);
  render(
    <MediaCollection
      hasNextPage={false}
      isFetchingNextPage={false}
      items={[storyboardItem]}
      labels={labels}
      layout="grid"
      onItemActivate={vi.fn()}
      onLoadMore={vi.fn()}
    />,
  );

  screen.getByRole("button", { name: "Preview video-storyboard.mp4" }).focus();
  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs),
  );
  expect(decode).not.toHaveBeenCalled();
  expect(
    screen.getByRole("article", {
      name: "video-storyboard.mp4 · Video",
    }),
  ).not.toHaveAttribute("data-storyboard-playing");
});

it("bounds rapid hover across a capacity window to one request and timer", async () => {
  vi.useFakeTimers();
  allowStoryboardMotion();
  const decode = vi.mocked(HTMLImageElement.prototype.decode);
  const items = Array.from({ length: 100 }, (_, index) => ({
    ...storyboardItem,
    id: `rapid-${index}`,
    name: `rapid-${index}.mp4`,
  }));
  const view = renderStoryboard(items);
  const cards = screen.getAllByRole("article");
  for (const card of cards) {
    fireEvent.pointerEnter(card);
  }
  expect(decode).not.toHaveBeenCalled();

  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs),
  );
  expect(decode).toHaveBeenCalledOnce();
  expect(
    cards.filter((card) => card.hasAttribute("data-storyboard-playing")),
  ).toHaveLength(1);
  expect(vi.getTimerCount()).toBe(1);

  view.unmount();
  expect(vi.getTimerCount()).toBe(0);
});

it("keeps the poster when storyboard decoding fails and clears timers on unmount", async () => {
  vi.useFakeTimers();
  allowStoryboardMotion();
  vi.mocked(HTMLImageElement.prototype.decode).mockRejectedValue(
    new Error("corrupt sprite"),
  );
  const view = renderStoryboard();
  const card = screen.getByRole("article", {
    name: "video-storyboard.mp4 · Video",
  });
  fireEvent.pointerEnter(card);
  await act(() =>
    vi.advanceTimersByTimeAsync(storyboardPlaybackTiming.hoverIntentMs),
  );

  expect(card).not.toHaveAttribute("data-storyboard-playing");
  expect(card.querySelector('[src="/poster.webp"]')).toBeInTheDocument();
  expect(card.querySelector('[src="/storyboard.webp"]')).not.toBeInTheDocument();
  view.unmount();
  expect(vi.getTimerCount()).toBe(0);
});

function renderStoryboard(
  items: MediaCollectionItem[] = [storyboardItem],
  motion: { finePointer: boolean; reducedMotion: boolean } = {
    finePointer: true,
    reducedMotion: false,
  },
) {
  setStoryboardMotion(motion);
  return render(storyboardCollection(items));
}

function storyboardCollection(items: MediaCollectionItem[]) {
  return (
    <MediaCollection
      hasNextPage={false}
      isFetchingNextPage={false}
      items={items}
      labels={labels}
      layout="grid"
      onLoadMore={vi.fn()}
    />
  );
}

function allowStoryboardMotion() {
  setStoryboardMotion({ finePointer: true, reducedMotion: false });
}

function setStoryboardMotion({
  finePointer,
  reducedMotion,
}: {
  finePointer: boolean;
  reducedMotion: boolean;
}) {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn((query: string) => ({
      matches: query.includes("prefers-reduced-motion")
        ? reducedMotion
        : finePointer,
      media: query,
    })),
  });
}
