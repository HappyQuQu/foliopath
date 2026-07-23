import createClient from "openapi-fetch";

import type { paths } from "./generated/schema";

// This is the only raw HTTP client boundary. Product features must consume
// reviewed domain adapters built on top of it rather than importing it directly.
export const apiClient = createClient<paths>({
  baseUrl: "",
  credentials: "same-origin",
});
