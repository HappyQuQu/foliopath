import type { StorybookConfig } from "@storybook/react-vite";

const config: StorybookConfig = {
  stories: ["../src/**/*.stories.@(ts|tsx)"],
  staticDirs: [
    { from: "../../tests/fixtures/media", to: "/test-media" },
    // Binary-only, synthetic content fixtures for the production group-review
    // component. Storybook has no authentication or real media API backend.
    { from: "../../tests/fixtures/storybook-assets", to: "/api/v1/assets" },
  ],
  addons: ["@storybook/addon-a11y"],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
};

export default config;
