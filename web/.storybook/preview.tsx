import type { Preview } from "@storybook/react-vite";

import { LocaleProvider } from "../src/lib/i18n/LocaleProvider";

import "../src/styles/global.css";

const preferenceKey = "foliopath.preferences.v1";

const preview: Preview = {
  parameters: {
    a11y: {
      test: "error",
    },
    backgrounds: {
      disable: true,
    },
    layout: "centered",
  },
  globalTypes: {
    locale: {
      description: "FolioPath locale",
      defaultValue: "zh-CN",
      toolbar: {
        icon: "globe",
        items: [
          { value: "zh-CN", title: "简体中文" },
          { value: "en", title: "English" },
        ],
      },
    },
    theme: {
      description: "FolioPath theme",
      defaultValue: "light",
      toolbar: {
        icon: "circlehollow",
        items: ["light", "dark"],
      },
    },
  },
  decorators: [
    (Story, context) => {
      const locale = context.globals.locale === "en" ? "en" : "zh-CN";
      const theme = context.globals.theme === "dark" ? "dark" : "light";
      window.localStorage.setItem(
        preferenceKey,
        JSON.stringify({ locale, theme }),
      );
      document.documentElement.dataset.theme = theme;
      document.documentElement.lang = locale;
      return (
        <LocaleProvider key={locale}>
          <Story />
        </LocaleProvider>
      );
    },
  ],
};

export default preview;
