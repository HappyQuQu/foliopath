import { createRef } from "react";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";

import {
  MediaCollection,
  mediaCollectionCapacityBudget,
  shouldLoadNextMediaPage,
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
