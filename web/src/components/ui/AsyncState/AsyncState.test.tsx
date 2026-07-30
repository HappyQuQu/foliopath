import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";

import { Button } from "../Button/Button";
import {
  EmptyState,
  ErrorState,
  LoadingState,
  OfflineState,
} from "./AsyncState";

it("announces loading progress without an urgent alert", () => {
  render(<LoadingState label="正在载入媒体" />);

  expect(screen.getByRole("status")).toHaveTextContent("正在载入媒体");
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
});

it("announces a blocking error and exposes its recovery action", async () => {
  const user = userEvent.setup();
  const onRetry = vi.fn();
  render(<ErrorState message="连接暂时不可用" onRetry={onRetry} />);

  expect(screen.getByRole("alert")).toHaveTextContent("连接暂时不可用");
  await user.click(screen.getByRole("button", { name: "重新尝试" }));
  expect(onRetry).toHaveBeenCalledOnce();
});

it("renders an empty-state recovery action without an urgent announcement", () => {
  render(
    <EmptyState
      action={<Button>Include subdirectories</Button>}
      description="Subdirectories contain 18 media items."
      title="No media in this directory"
    />,
  );

  expect(screen.getByText("No media in this directory")).toBeVisible();
  expect(
    screen.getByRole("button", { name: "Include subdirectories" }),
  ).toBeVisible();
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
});

it("announces a persistent offline state without treating it as empty", () => {
  render(
    <OfflineState
      description="The preserved index does not prove the source is empty."
      title="This library is offline"
    />,
  );

  expect(screen.getByRole("status")).toHaveTextContent(
    "This library is offline",
  );
  expect(screen.getByRole("status")).toHaveTextContent(
    "does not prove the source is empty",
  );
});
