import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "vitest";

import { MediaViewer } from "./MediaViewer";

const labels = {
  close: "Close",
  closeInformation: "Close information panel",
  exitFullscreen: "Exit fullscreen",
  fit: "Fit to window",
  fullscreen: "Fullscreen",
  imageFailed: "Image failed",
  info: "Show basic information",
  information: "Basic information",
  loadFailedDescription: "The original was not modified.",
  next: "Next item",
  originalSize: "Show at 1:1",
  previous: "Previous item",
  retry: "Try again",
  shortcutHint: "Drag to pan · Esc to exit",
  videoFailed: "Video failed",
  zoomIn: "Zoom in",
  zoomOut: "Zoom out",
};

const item = {
  contentUrl: "/api/v1/assets/photo/content",
  details: [
    { label: "Location", value: "Travel/Kyoto/photo.jpg" },
    { label: "Dimensions", value: "1200 × 800 px" },
  ],
  id: "photo",
  kind: "image" as const,
  name: "photo.jpg",
  summary: "Image · JPEG",
};

beforeEach(() => {
  Object.defineProperty(document, "fullscreenElement", {
    configurable: true,
    value: null,
  });
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn().mockReturnValue({ matches: false }),
  });
});

it("provides fit, 1:1, zoom, information, and close controls", async () => {
  const user = userEvent.setup();
  const close = vi.fn();
  render(
    <MediaViewer
      canGoNext
      canGoPrevious
      item={item}
      labels={labels}
      onClose={close}
      onNext={vi.fn()}
      onPrevious={vi.fn()}
      position="Item 2 of 4"
    />,
  );

  expect(screen.getByRole("img", { name: "photo.jpg" })).toHaveAttribute(
    "src",
    item.contentUrl,
  );
  expect(screen.getByRole("main", { name: "photo.jpg" })).toHaveFocus();
  expect(screen.queryByRole("complementary")).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Fit to window" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );

  await user.click(screen.getByRole("button", { name: "Show at 1:1" }));
  expect(screen.getByRole("button", { name: "Show at 1:1" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  fireEvent.wheel(screen.getByRole("img", { name: "photo.jpg" }).parentElement!, {
    deltaY: -1,
  });
  expect(screen.getByText("125%")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "Show basic information" }));
  expect(screen.getByRole("complementary", { name: "Basic information" })).toBeVisible();
  expect(screen.getByText("Travel/Kyoto/photo.jpg")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "Close information panel" }));
  expect(screen.queryByRole("complementary")).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Close" }));
  expect(close).toHaveBeenCalledOnce();
});

it("supports keyboard navigation from viewer buttons without hijacking media controls", () => {
  const close = vi.fn();
  const next = vi.fn();
  const previous = vi.fn();
  const { container, rerender } = render(
    <MediaViewer
      canGoNext
      canGoPrevious
      item={item}
      labels={labels}
      onClose={close}
      onNext={next}
      onPrevious={previous}
      position="Item 2 of 4"
    />,
  );

  fireEvent.keyDown(window, { key: "ArrowLeft" });
  fireEvent.keyDown(window, { key: "ArrowRight" });
  fireEvent.keyDown(window, { key: "Escape" });
  expect(previous).toHaveBeenCalledOnce();
  expect(next).toHaveBeenCalledOnce();
  expect(close).toHaveBeenCalledOnce();

  screen.getByRole("button", { name: "Close" }).focus();
  fireEvent.keyDown(screen.getByRole("button", { name: "Close" }), {
    key: "ArrowRight",
  });
  expect(next).toHaveBeenCalledTimes(2);

  fireEvent.keyDown(window, { key: "i" });
  expect(screen.getByRole("complementary", { name: "Basic information" })).toBeVisible();
  fireEvent.keyDown(window, { key: "I" });
  expect(screen.queryByRole("complementary")).not.toBeInTheDocument();

  rerender(
    <MediaViewer
      canGoNext
      canGoPrevious
      item={{ ...item, id: "clip", kind: "video", name: "clip.mp4" }}
      labels={labels}
      onClose={close}
      onNext={next}
      onPrevious={previous}
      position="Item 2 of 4"
    />,
  );
  const video = container.querySelector("video")!;
  video.focus();
  fireEvent.keyDown(video, { key: "ArrowRight" });
  expect(next).toHaveBeenCalledTimes(2);

  const dialog = document.createElement("div");
  const dialogButton = document.createElement("button");
  dialog.setAttribute("role", "dialog");
  dialog.append(dialogButton);
  document.body.append(dialog);
  dialogButton.focus();
  fireEvent.keyDown(dialogButton, { key: "ArrowRight" });
  expect(next).toHaveBeenCalledTimes(2);
  dialog.remove();
});

it("uses the fullscreen API and native video controls", async () => {
  const user = userEvent.setup();
  const requestFullscreen = vi.fn().mockResolvedValue(undefined);
  Object.defineProperty(HTMLElement.prototype, "requestFullscreen", {
    configurable: true,
    value: requestFullscreen,
  });
  const { container, rerender } = render(
    <MediaViewer
      canGoNext={false}
      canGoPrevious={false}
      item={item}
      labels={labels}
      onClose={vi.fn()}
      onNext={vi.fn()}
      onPrevious={vi.fn()}
      position="Current media"
    />,
  );

  await user.click(screen.getByRole("button", { name: "Fullscreen" }));
  expect(requestFullscreen).toHaveBeenCalledOnce();

  rerender(
    <MediaViewer
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
      onPrevious={vi.fn()}
      position="Current media"
    />,
  );
  expect(container.querySelector("video")).toHaveAttribute("controls");
  expect(container.querySelector("video")).toHaveAttribute(
    "poster",
    "/api/v1/assets/clip/thumbnail",
  );
  expect(screen.queryByRole("button", { name: "Zoom in" })).not.toBeInTheDocument();
});

it("keeps viewer navigation available while media is unavailable", () => {
  const next = vi.fn();
  render(
    <MediaViewer
      availability={{
        description: "The library mount is unavailable.",
        kind: "offline",
        title: "Library is offline",
      }}
      canGoNext
      canGoPrevious={false}
      item={item}
      labels={labels}
      onClose={vi.fn()}
      onNext={next}
      onPrevious={vi.fn()}
      position="Item 1 of 2"
    />,
  );

  expect(screen.getByRole("status")).toHaveTextContent("Library is offline");
  screen.getByRole("button", { name: "Next item" }).click();
  expect(next).toHaveBeenCalledOnce();
  expect(screen.queryByRole("img", { name: "photo.jpg" })).not.toBeInTheDocument();
});

it("starts with indexed information collapsed on a narrow viewport", () => {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn().mockReturnValue({ matches: true }),
  });
  render(
    <MediaViewer
      canGoNext={false}
      canGoPrevious={false}
      item={item}
      labels={labels}
      onClose={vi.fn()}
      onNext={vi.fn()}
      onPrevious={vi.fn()}
      position="Current media"
    />,
  );

  expect(screen.queryByRole("complementary")).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Show basic information" })).toHaveAttribute(
    "aria-pressed",
    "false",
  );
});
