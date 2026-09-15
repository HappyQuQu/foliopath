import { expect, it, vi } from "vitest";
import { useAIModelsQuery, useSemanticSettingsQueries, useVideoSemanticStatusQuery } from "./queries";

vi.mock("@tanstack/react-query", () => ({
  useQuery: (options: unknown) => options,
  useQueries: (options: unknown) => options,
  useMutation: vi.fn(),
  useQueryClient: vi.fn(),
}));

it("refreshes model availability without requiring a page reload", () => {
  expect(useAIModelsQuery()).toMatchObject({ queryKey: ["ai", "models"], refetchInterval: 10_000 });
});

it("refreshes idle, missing and active image status at bounded intervals", () => {
  const options = useSemanticSettingsQueries(["lib_1"]) as unknown as {
    queries: Array<{ queryKey: string[]; refetchInterval: (query: { state: { data?: { state: string } } }) => number }>;
  };
  expect(options.queries[0]!.queryKey).toEqual(["ai", "semantic", "lib_1"]);
  const interval = options.queries[0]!.refetchInterval;
  for (const state of ["ready", "disabled", "awaiting_model", "degraded"]) {
    expect(interval({ state: { data: { state } } })).toBe(10_000);
  }
  expect(interval({ state: {} })).toBe(10_000);
  expect(interval({ state: { data: { state: "building" } } })).toBe(1_500);
  expect(interval({ state: { data: { state: "clearing" } } })).toBe(1_500);
});

it("scopes video polling to the selected library", () => {
  expect(useVideoSemanticStatusQuery("lib_2")).toMatchObject({ queryKey: ["ai", "video-status", "lib_2"], refetchInterval: 10_000 });
});
