import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "vitest";

import { MediaViewer } from "./MediaViewer";

const labels = {
  close: "Close",
  exitFullscreen: "Exit fullscreen",
  fit: "Fit to window",
  fullscreen: "Fullscreen",
  imageFailed: "Image failed",
  info: "Show basic information",
  information: "Basic information",
  next: "Next item",
  originalSize: "Show at 1:1",
  previous: "Previous item",
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
};

beforeEach(() => {
  Object.defineProperty(document, "fullscreenElement", {
    configurable: true,
    value: null,
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
  expect(screen.getByRole("complementary", { name: "Basic information" })).toBeVisible();
  expect(screen.getByText("Travel/Kyoto/photo.jpg")).toBeVisible();
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
  expect(screen.queryByRole("complementary")).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Close" }));
  expect(close).toHaveBeenCalledOnce();
});

it("supports keyboard previous, next, and Escape without hijacking controls", () => {
  const close = vi.fn();
  const next = vi.fn();
  const previous = vi.fn();
  render(
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
  expect(next).toHaveBeenCalledOnce();
});

it("uses the fullscreen API and native video controls", async () => {
  const user = userEvent.setup();
  const requestFullscreen = vi.fn().mockResolvedValue(undefined);
  Object.defineProperty(HTMLElement.prototype, "requestFullscreen", {
    configurable: true,
    value: requestFullscreen,
  });
  const { rerender } = render(
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
      }}
      labels={labels}
      onClose={vi.fn()}
      onNext={vi.fn()}
      onPrevious={vi.fn()}
      position="Current media"
    />,
  );
  expect(screen.getByLabelText("clip.mp4")).toHaveAttribute("controls");
  expect(screen.queryByRole("button", { name: "Zoom in" })).not.toBeInTheDocument();
});
