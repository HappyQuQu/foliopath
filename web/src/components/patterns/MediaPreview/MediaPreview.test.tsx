import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";

import { MediaPreview } from "./MediaPreview";

const labels = {
  close: "Close preview",
  followingDescription: "Click another item to update the preview.",
  followingTitle: "Preview follows selection",
  imageFailed: "Image failed",
  loadFailedDescription: "The original was not modified.",
  next: "Next item",
  openViewer: "Open full viewer",
  pin: "Pin preview",
  pinnedDescription: "Double-click to switch.",
  pinnedTitle: "Preview pinned",
  position: "Item 2 of 3",
  previous: "Previous item",
  preview: "Preview",
  resize: "Resize preview",
  retry: "Try again",
  unpin: "Unpin preview",
  videoFailed: "Video failed",
};

const item = {
  contentUrl: "/api/v1/assets/photo/content",
  details: [
    { label: "Type", value: "Image · image/jpeg" },
    {
      label: "Location",
      layout: "path" as const,
      value:
        "2026年7月精品园子视频合集5/黑长发学生妹格外长且没有空格的文件路径/photo.jpg",
    },
    { label: "Size", value: "2.4 MB" },
  ],
  id: "photo",
  kind: "image" as const,
  name: "photo.jpg",
};

it("renders non-modal media, metadata, navigation, and close controls", () => {
  const close = vi.fn();
  const next = vi.fn();
  const openViewer = vi.fn();
  const previous = vi.fn();
  render(
    <MediaPreview
      canGoNext
      canGoPrevious
      item={item}
      labels={labels}
      onClose={close}
      onNext={next}
      onOpenViewer={openViewer}
      onPinnedChange={vi.fn()}
      onPrevious={previous}
      onWidthChange={vi.fn()}
      pinned={false}
      width={406}
    />,
  );

  expect(
    screen.getByRole("complementary", { name: "Preview: photo.jpg" }),
  ).toBeVisible();
  expect(screen.getByRole("img", { name: "photo.jpg" })).toHaveAttribute(
    "src",
    item.contentUrl,
  );
  expect(screen.getByText("2.4 MB")).toBeVisible();
  const location = screen.getByText(/2026年7月精品园子视频合集5/);
  expect(location.parentElement).toHaveAttribute("data-layout", "path");
  expect(location).toHaveStyle({
    overflowWrap: "anywhere",
    whiteSpace: "normal",
  });
  screen.getByRole("button", { name: "Previous item" }).click();
  screen.getByRole("button", { name: "Next item" }).click();
  screen.getByRole("button", { name: "Open full viewer" }).click();
  screen.getByRole("button", { name: "Close preview" }).click();
  expect(previous).toHaveBeenCalledOnce();
  expect(next).toHaveBeenCalledOnce();
  expect(openViewer).toHaveBeenCalledOnce();
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
      onOpenViewer={vi.fn()}
      onPinnedChange={vi.fn()}
      onPrevious={vi.fn()}
      onWidthChange={resize}
      pinned={false}
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
      onOpenViewer={vi.fn()}
      onPinnedChange={vi.fn()}
      onPrevious={vi.fn()}
      onWidthChange={resize}
      pinned={false}
      width={610}
    />,
  );
  fireEvent.keyDown(screen.getByRole("separator", { name: "Resize preview" }), {
    key: "ArrowLeft",
  });
  expect(resize).toHaveBeenLastCalledWith(620);
});

it("announces pin state and closes with Escape", () => {
  const close = vi.fn();
  const changePin = vi.fn();
  const { rerender } = render(
    <MediaPreview
      canGoNext={false}
      canGoPrevious={false}
      item={item}
      labels={labels}
      onClose={close}
      onNext={vi.fn()}
      onOpenViewer={vi.fn()}
      onPinnedChange={changePin}
      onPrevious={vi.fn()}
      onWidthChange={vi.fn()}
      pinned={false}
      width={406}
    />,
  );

  expect(screen.getByText("Preview follows selection")).toBeVisible();
  screen.getByRole("button", { name: "Pin preview" }).click();
  expect(changePin).toHaveBeenCalledWith(true);

  rerender(
    <MediaPreview
      canGoNext={false}
      canGoPrevious={false}
      item={item}
      labels={labels}
      onClose={close}
      onNext={vi.fn()}
      onOpenViewer={vi.fn()}
      onPinnedChange={changePin}
      onPrevious={vi.fn()}
      onWidthChange={vi.fn()}
      pinned
      width={406}
    />,
  );
  expect(screen.getByText("Preview pinned")).toBeVisible();
  expect(screen.getByRole("button", { name: "Unpin preview" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  fireEvent.keyDown(window, { key: "Escape" });
  expect(close).toHaveBeenCalledOnce();
});

it("uses native inline controls for video content", () => {
  const { rerender } = render(
    <MediaPreview
      canGoNext={false}
      canGoPrevious={false}
      item={{
        ...item,
        contentUrl: "/api/v1/assets/clip/content",
        id: "clip",
        kind: "video",
        name: "clip.mp4",
        posterUrl: "/api/v1/assets/clip/thumbnail",
      }}
      labels={labels}
      onClose={vi.fn()}
      onNext={vi.fn()}
      onOpenViewer={vi.fn()}
      onPinnedChange={vi.fn()}
      onPrevious={vi.fn()}
      onWidthChange={vi.fn()}
      pinned={false}
      width={406}
    />,
  );

  const video = screen.getByLabelText("clip.mp4");
  expect(video.tagName).toBe("VIDEO");
  expect(video).toHaveProperty("autoplay", true);
  expect(video).toHaveProperty("muted", true);
  expect(video).toHaveAttribute("controls");
  expect(video).toHaveAttribute("playsinline");
  expect(video).toHaveAttribute("preload", "metadata");
  expect(video).toHaveAttribute("src", "/api/v1/assets/clip/content");
  expect(video).toHaveAttribute("poster", "/api/v1/assets/clip/thumbnail");

  fireEvent.error(video);
  expect(screen.getByRole("heading", { name: "Video failed" })).toBeVisible();
  expect(screen.queryByLabelText("clip.mp4")).not.toBeInTheDocument();

  rerender(
    <MediaPreview
      canGoNext={false}
      canGoPrevious={false}
      item={{
        ...item,
        contentUrl: "/api/v1/assets/clip-next/content",
        id: "clip-next",
        kind: "video",
        name: "clip-next.mp4",
      }}
      labels={labels}
      onClose={vi.fn()}
      onNext={vi.fn()}
      onOpenViewer={vi.fn()}
      onPinnedChange={vi.fn()}
      onPrevious={vi.fn()}
      onWidthChange={vi.fn()}
      pinned={false}
      width={406}
    />,
  );

  expect(video.isConnected).toBe(false);
  expect(document.querySelectorAll("video")).toHaveLength(1);
  expect(screen.getByLabelText("clip-next.mp4")).toHaveAttribute(
    "src",
    "/api/v1/assets/clip-next/content",
  );
});

it("keeps autoplay and default mute independent", () => {
  render(
    <MediaPreview
      autoPlayVideo={false}
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
      onOpenViewer={vi.fn()}
      onPinnedChange={vi.fn()}
      onPrevious={vi.fn()}
      onWidthChange={vi.fn()}
      pinned={false}
      width={406}
    />,
  );

  const video = screen.getByLabelText("clip.mp4");
  expect(video).toHaveProperty("autoplay", false);
  expect(video).toHaveProperty("muted", true);
});

it("allows an audible video preview preference", () => {
  render(
    <MediaPreview
      autoPlayVideo
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
      muteVideo={false}
      onClose={vi.fn()}
      onNext={vi.fn()}
      onOpenViewer={vi.fn()}
      onPinnedChange={vi.fn()}
      onPrevious={vi.fn()}
      onWidthChange={vi.fn()}
      pinned={false}
      width={406}
    />,
  );

  const video = screen.getByLabelText("clip.mp4");
  expect(video).toHaveProperty("autoplay", true);
  expect(video).toHaveProperty("muted", false);
});
