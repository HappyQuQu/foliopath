import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { expect, it } from "vitest";
import { RadioOption } from "./RadioOption";

it("keeps native exclusive selection, associated labels and disabled choices", async () => {
  const user = userEvent.setup();
  function Group() { const [value, setValue] = useState("library"); return <fieldset><legend>Scope</legend>{["library", "directory", "all"].map(option => <RadioOption key={option} name="scope" value={option} checked={value === option} disabled={option === "directory"} onChange={() => setValue(option)}>{option}</RadioOption>)}</fieldset>; }
  render(<Group />);
  expect(screen.getByRole("group", { name: "Scope" })).toBeVisible();
  expect(screen.getByRole("radio", { name: "library" })).toBeChecked();
  await user.click(screen.getByText("all"));
  expect(screen.getByRole("radio", { name: "all" })).toBeChecked();
  expect(screen.getByRole("radio", { name: "library" })).not.toBeChecked();
  expect(screen.getByRole("radio", { name: "directory" })).toBeDisabled();
});
