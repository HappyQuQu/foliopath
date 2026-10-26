import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it } from "vitest";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ReviewPhotoPreview } from "./ReviewPhotoPreview";

beforeEach(()=>window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}'));

it("falls back when all photo thumbnails fail and resets errors for a new source", () => {
 const {container,rerender}=render(<LocaleProvider><ReviewPhotoPreview assetIds={["asset_a","asset_b"]}/></LocaleProvider>);
 const images=container.querySelectorAll("img");fireEvent.error(images[0]!);expect(container.querySelectorAll("img")).toHaveLength(1);fireEvent.error(images[1]!);
 expect(screen.getByText("暂无可用照片预览")).toBeVisible();
 rerender(<LocaleProvider><ReviewPhotoPreview assetIds={["asset_c"]}/></LocaleProvider>);
 expect(container.querySelector("img")).toHaveAttribute("src","/api/v1/assets/asset_c/thumbnail?variant=grid");expect(screen.queryByText("暂无可用照片预览")).not.toBeInTheDocument();
});
it("bounds and deduplicates a preview rather than loading a whole collection", () => {
 const {container}=render(<LocaleProvider><ReviewPhotoPreview assetIds={["a","a","b","c","d","e"]}/></LocaleProvider>);
 expect(container.querySelectorAll("img")).toHaveLength(4);expect(container.querySelector('img[src*="/e/"]')).not.toBeInTheDocument();
});
