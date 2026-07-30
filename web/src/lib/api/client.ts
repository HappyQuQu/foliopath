import createClient, {
  createQuerySerializer,
  type QuerySerializerOptions,
} from "openapi-fetch";

import type { paths } from "./generated/schema";

// This is the only raw HTTP client boundary. Product features must consume
// reviewed domain adapters built on top of it rather than importing it directly.
const apiQuerySerializerOptions = {
  array: {
    explode: false,
    style: "form",
  },
} satisfies QuerySerializerOptions;

const serializeQuery = createQuerySerializer(apiQuerySerializerOptions);

export function serializeAPIQuery(query: Record<string, unknown>): string {
  return serializeQuery(query);
}

export const apiClient = createClient<paths>({
  baseUrl: "",
  credentials: "same-origin",
  fetch: (...args) => globalThis.fetch(...args),
  querySerializer: serializeAPIQuery,
});
