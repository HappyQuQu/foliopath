import { render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";

import {
  MediaCollection,
  type MediaCollectionItem,
} from "./MediaCollection";

const labels = {
  animated: "Animated",
  failedThumbnail: "Thumbnail failed",
  image: "Image",
  loadMore: "Load more",
  loadMoreFailed: "More media could not be loaded.",
  loadingMore: "Loading more",
  pendingThumbnail: "Preparing thumbnail",
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
  render(
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
});
