import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";

import { MediaPreview } from "./MediaPreview";

const labels = {
  close: "Close preview",
  imageFailed: "Image failed",
  next: "Next item",
  position: "Item 2 of 3",
  previous: "Previous item",
  preview: "Preview",
  resize: "Resize preview",
  videoFailed: "Video failed",
};

const item = {
  contentUrl: "/api/v1/assets/photo/content",
  details: [
    { label: "Type", value: "Image · image/jpeg" },
    { label: "Size", value: "2.4 MB" },
  ],
  id: "photo",
  kind: "image" as const,
  name: "photo.jpg",
};

it("renders non-modal media, metadata, navigation, and close controls", () => {
  const close = vi.fn();
  const next = vi.fn();
  const previous = vi.fn();
  render(
    <MediaPreview
      canGoNext
      canGoPrevious
      item={item}
      labels={labels}
      onClose={close}
      onNext={next}
      onPrevious={previous}
      onWidthChange={vi.fn()}
      width={406}
    />,
  );

  expect(screen.getByRole("complementary", { name: "Preview: photo.jpg" })).toBeVisible();
  expect(screen.getByRole("img", { name: "photo.jpg" })).toHaveAttribute(
    "src",
    item.contentUrl,
  );
  expect(screen.getByText("2.4 MB")).toBeVisible();
  screen.getByRole("button", { name: "Previous item" }).click();
  screen.getByRole("button", { name: "Next item" }).click();
  screen.getByRole("button", { name: "Close preview" }).click();
  expect(previous).toHaveBeenCalledOnce();
  expect(next).toHaveBeenCalledOnce();
  expect(close).toHaveBeenCalledOnce();
});

it("supports keyboard width adjustment and clamps the shared panel", () => {
  const resize = vi.fn();
  const { rerender } = render(
    <MediaPreview
      canGoNext={false}
      canGoPrevious={false}
      item={item}
      labels={labels}
      onClose={vi.fn()}
      onNext={vi.fn()}
      onPrevious={vi.fn()}
      onWidthChange={resize}
      width={406}
    />,
  );

  fireEvent.keyDown(screen.getByRole("separator", { name: "Resize preview" }), {
    key: "ArrowLeft",
  });
  expect(resize).toHaveBeenLastCalledWith(430);

  rerender(
    <MediaPreview
      canGoNext={false}
      canGoPrevious={false}
      item={item}
      labels={labels}
      onClose={vi.fn()}
      onNext={vi.fn()}
      onPrevious={vi.fn()}
      onWidthChange={resize}
      width={610}
    />,
  );
  fireEvent.keyDown(screen.getByRole("separator", { name: "Resize preview" }), {
    key: "ArrowLeft",
  });
  expect(resize).toHaveBeenLastCalledWith(620);
});

it("uses native inline controls for video content", () => {
  render(
    <MediaPreview
      canGoNext={false}
      canGoPrevious={false}
      item={{
        ...item,
        contentUrl: "/api/v1/assets/clip/content",
        id: "clip",
        kind: "video",
        name: "clip.mp4",
      }}
      labels={labels}
      onClose={vi.fn()}
      onNext={vi.fn()}
      onPrevious={vi.fn()}
      onWidthChange={vi.fn()}
      width={406}
    />,
  );

  const video = screen.getByLabelText("clip.mp4");
  expect(video.tagName).toBe("VIDEO");
  expect(video).toHaveAttribute("controls");
  expect(video).toHaveAttribute("playsinline");
  expect(video).toHaveAttribute("preload", "metadata");
  expect(video).toHaveAttribute("src", "/api/v1/assets/clip/content");
});
