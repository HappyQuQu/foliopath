import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";

import { Button } from "../Button/Button";
import { EmptyState, OfflineState } from "./AsyncState";

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
