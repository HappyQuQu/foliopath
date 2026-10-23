import { useSyncExternalStore } from "react";
import type { AssetTag, Tag } from "../api/curation";
import { readTagLanguagePreference, subscribeTagLanguagePreference, writeTagLanguagePreference, type TagLanguagePreference } from "../storage/preferences";
import { useLocale } from "./LocaleProvider";

export function displayTagName(tag: Tag | AssetTag, language: TagLanguagePreference, locale: "zh-CN" | "en"): string {
  // A legacy/manual association never translates merely because its literal
  // text happens to look like an English model class.
  if (("source" in tag && tag.source !== "ai") || !tag.localization) return tag.name;
  const effective = language === "interface" ? locale : language;
  return effective === "bilingual" ? `${tag.localization.zhCN} / ${tag.localization.en}`
    : effective === "en" ? tag.localization.en : tag.localization.zhCN;
}

export function useTagDisplay() {
  const { locale } = useLocale();
  const language = useSyncExternalStore(subscribeTagLanguagePreference, readTagLanguagePreference, () => "interface" as const);
  return { language, setLanguage: writeTagLanguagePreference, tagName: (tag: Tag | AssetTag) => displayTagName(tag, language, locale) };
}
