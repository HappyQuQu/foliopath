import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";

import { PanelResizer } from "./PanelResizer";

it("resizes in the panel growth direction and clamps at its bounds", () => {
  const onChange = vi.fn();
  render(
    <PanelResizer
      ariaLabel="Resize panel"
      growDirection="right"
      max={420}
      min={224}
      onChange={onChange}
      value={272}
    />,
  );

  const separator = screen.getByRole("separator", { name: "Resize panel" });
  fireEvent.keyDown(separator, { key: "ArrowRight" });
  expect(onChange).toHaveBeenLastCalledWith(296);

  fireEvent.keyDown(separator, { key: "Home" });
  expect(onChange).toHaveBeenLastCalledWith(224);

  fireEvent.keyDown(separator, { key: "End" });
  expect(onChange).toHaveBeenLastCalledWith(420);
});

it("reverses horizontal keys for a right-side panel", () => {
  const onChange = vi.fn();
  render(
    <PanelResizer
      ariaLabel="Resize preview"
      growDirection="left"
      max={620}
      min={360}
      onChange={onChange}
      value={406}
    />,
  );

  const separator = screen.getByRole("separator", { name: "Resize preview" });
  fireEvent.keyDown(separator, { key: "ArrowLeft" });
  expect(onChange).toHaveBeenLastCalledWith(430);
  fireEvent.keyDown(separator, { key: "ArrowRight" });
  expect(onChange).toHaveBeenLastCalledWith(382);
});

it("tracks mouse movement outside the separator while dragging", () => {
  const onChange = vi.fn();
  render(
    <PanelResizer
      ariaLabel="Resize panel"
      growDirection="right"
      max={420}
      min={224}
      onChange={onChange}
      value={272}
    />,
  );

  fireEvent.mouseDown(
    screen.getByRole("separator", { name: "Resize panel" }),
    { button: 0, clientX: 272 },
  );
  fireEvent.mouseMove(window, { clientX: 332 });
  fireEvent.mouseUp(window);

  expect(onChange).toHaveBeenLastCalledWith(332);
  expect(document.body).not.toHaveStyle({ cursor: "col-resize" });
});
