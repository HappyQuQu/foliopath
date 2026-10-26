import { render, screen, within } from "@testing-library/react";
import { expect, it } from "vitest";
import { Input } from "../../ui";
import { ContextSidebar } from "./ContextSidebar";

it("names each contextual region independently and preserves native field labels", () => {
  render(<><ContextSidebar title="Search tools" description="Current library"><label>Find people<Input type="search" /></label></ContextSidebar><ContextSidebar title="Favorite filters"><p>Images</p></ContextSidebar></>);
  const search = screen.getByRole("region", { name: "Search tools" });
  expect(within(search).getByRole("searchbox", { name: "Find people" })).toBeVisible();
  expect(within(search).getByText("Current library")).toBeVisible();
  expect(screen.getByRole("region", { name: "Favorite filters" })).toBeVisible();
});
