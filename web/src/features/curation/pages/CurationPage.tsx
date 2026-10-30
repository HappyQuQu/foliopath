import { AssetDeletionAction, useDeletionFlow, useDeletionSelection } from "../../deletion";
import { AssetExifPanel } from "../../media";
import { useEffect, useMemo, useState, type CSSProperties } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

import { ContextSidebar } from "../../../components/patterns/ContextSidebar/ContextSidebar";
import { AppShell } from "../../../components/patterns/AppShell/AppShell";
import {
  MediaCollection,
  MediaCollectionSkeleton,
  type MediaCollectionItem,
} from "../../../components/patterns/MediaCollection/MediaCollection";
import {
  MediaPreview,
  type MediaPreviewItem,
} from "../../../components/patterns/MediaPreview/MediaPreview";
import { mediaPreviewDetails } from "../../../components/patterns/MediaPreview/mediaPreviewDetails";
import { useMediaPreviewController } from "../../../components/patterns/MediaPreview/useMediaPreviewController";
import { Button, EmptyState, ErrorState, LoadingState, Select } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import {
  assetContentUrl,
  type Asset,
  type AssetKind,
  type AssetSort,
} from "../../../lib/api/catalog";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { useTagDisplay } from "../../../lib/i18n/tagDisplay";
import {
  mediaAvailability,
  mediaAvailabilityPresentation,
  mediaPosterUrl,
  mediaStoryboard,
} from "../../../lib/media/availability";
import {
  createViewerLocationState,
  readViewerReturnState,
} from "../../../lib/navigation/viewer";
import { retryInfiniteNextPage } from "../../../lib/query/retryInfiniteNextPage";
import { readMediaLayoutPreference } from "../../../lib/storage/preferences";
import { paths } from "../../../routes/paths";
import {
  defaultBrowseUrlState,
  DirectoryNavigation,
} from "../../browse";
import { useLibrariesQuery, useLibraryQuery } from "../../libraries";
import { AssetCurationControls } from "../components/AssetCurationControls";
import { LibraryTagCloud } from "../components/LibraryTagCloud";
import {
  useFavoriteMutation,
  useFavoritesQuery,
  useTagAssetsQuery,
} from "../queries";
import styles from "./CurationPage.module.css";

export function CurationPage({
  logoutPending,
  onLogout,
  session,
  tagId,
}: {
  logoutPending?: boolean;
  onLogout?: () => Promise<void>;
  session: AuthenticatedSession;
  tagId?: string;
}) {
  const { locale, t } = useLocale();
  const deletion = useDeletionFlow(session);
  const { tagName, language } = useTagDisplay();
  const location = useLocation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [layout] = useState(readMediaLayoutPreference);
  const [sizeControlContainer, setSizeControlContainer] = useState<HTMLElement | null>(null);
  const showingTags = location.pathname.startsWith(paths.tags);
  const librariesQuery = useLibrariesQuery();
  const libraries = useMemo(
    () => librariesQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [librariesQuery.data],
  );
  const requestedLibraryId = searchParams.get("libraryId");
  const hasRequestedLibrary = searchParams.has("libraryId");
  const listedLibrary = libraries.find(library => library.id === requestedLibraryId);
  // Resolve deep links beyond the picker page without falling back to another library.
  const requestedLibraryQuery = useLibraryQuery(
    requestedLibraryId && !listedLibrary && !librariesQuery.isPending ? requestedLibraryId : "",
  );
  const currentLibrary = hasRequestedLibrary
    ? listedLibrary ?? requestedLibraryQuery.data?.library
    : libraries.find(library => library.status === "ready") ?? libraries[0];
  const selectableLibraries = currentLibrary && !libraries.some(library => library.id === currentLibrary.id)
    ? [...libraries, currentLibrary] : libraries;
  const libraryId = currentLibrary?.id;
  const kind = searchParams.get("kind") === "image" || searchParams.get("kind") === "video"
    ? searchParams.get("kind") as "image" | "video"
    : "all";
  const requestedSort = searchParams.get("sort");
  const sort: AssetSort | "favoritedAt" = showingTags
    ? requestedSort === "name" || requestedSort === "size" ? requestedSort : "modifiedAt"
    : requestedSort === "modifiedAt" || requestedSort === "name" || requestedSort === "size"
      ? requestedSort
      : "favoritedAt";
  const kinds: AssetKind[] | undefined = kind === "all"
    ? undefined
    : kind === "image" ? ["image", "animated"] : ["video"];
  const queryOptions = {
    ...(kinds ? { kinds } : {}),
    ...(libraryId ? { libraryId } : {}),
    order: "desc" as const,
    sort,
  };
  const favoritesQuery = useFavoritesQuery(queryOptions, Boolean(libraryId) && !showingTags);
  const favoriteCount = favoritesQuery.data?.pages[0]?.counts.all;
  const tagAssetsQuery = useTagAssetsQuery(tagId, queryOptions, Boolean(libraryId) && showingTags);
  const selectedTagName = useMemo(() => {
    if (!libraryId || !tagId) return undefined;
    for (const page of tagAssetsQuery.data?.pages ?? []) {
      for (const item of page.items) {
        const tag = item.curation.tags.find(tag => tag.id === tagId);
        if (tag) return tagName(tag);
      }
    }
    return undefined;
  }, [libraryId, tagId, tagAssetsQuery.data, language, locale]);
  const favoriteMutation = useFavoriteMutation();
  const activeQuery = tagId ? tagAssetsQuery : favoritesQuery;
  const assets = useMemo(
    () => !libraryId || (showingTags && !tagId) ? [] : activeQuery.data?.pages.flatMap((page) => page.items.map((item) => item.asset)) ?? [],
    [activeQuery.data, libraryId, showingTags, tagId],
  );
  const mediaItems = useMemo(
    () => mapItems(assets, locale),
    [assets, locale],
  );
  const selectedPathIds = useMemo(() => new Set<string>(), []);
  const preview = useMediaPreviewController({
    items: assets,
    retainPinnedOutsideResults: false,
    resetKey: `${libraryId ?? "none"}:${kind}:${showingTags ? `tag:${tagId ?? "index"}` : "favorites"}`,
  });
  const selection = useDeletionSelection({ flow: deletion, scope: `${location.pathname}:${searchParams}`, loadedTargets: assets.map(item => ({ type: "asset" as const, id: item.id })), assets, onEnter: preview.close, onAccepted: preview.close });
  const previewAsset = preview.previewItem;
  const previewItem = useMemo<MediaPreviewItem | undefined>(
    () =>
      previewAsset
        ? {
            contentUrl: assetContentUrl(previewAsset.id),
            details: mediaPreviewDetails(previewAsset, locale, {
              animated: t("browse.kindAnimated"),
              dimensions: t("browse.detailDimensions"),
              duration: t("browse.detailDuration"),
              image: t("browse.kindImage"),
              modified: t("browse.detailModified"),
              path: t("browse.detailPath"),
              size: t("browse.detailSize"),
              type: t("browse.detailType"),
              unknown: t("browse.detailUnknown"),
              video: t("browse.kindVideo"),
            }),
            id: previewAsset.id,
            kind: previewAsset.kind,
            name: previewAsset.name,
            mimeType: previewAsset.mimeType,
            ...(mediaPosterUrl(previewAsset)
              ? { posterUrl: mediaPosterUrl(previewAsset) }
              : {}),
          }
        : undefined,
    [locale, previewAsset, t],
  );
  const heading = showingTags
    ? tagId ? selectedTagName ?? t("tagCloud.filtered") : t("curation.tags")
    : t("curation.favorites");

  useEffect(() => {
    if (!libraryId || hasRequestedLibrary) return;
    const next = new URLSearchParams(searchParams);
    next.set("libraryId", libraryId);
    setSearchParams(next, { replace: true });
  }, [hasRequestedLibrary, libraryId, searchParams, setSearchParams]);

  useEffect(() => {
    const returnState = readViewerReturnState(location.state);
    if (
      !returnState ||
      !assets.some((asset) => asset.id === returnState.restoreFocusAssetId)
    ) {
      return;
    }
    preview.collectionRef.current?.restoreItem(returnState.restoreFocusAssetId);
    navigate(`${location.pathname}${location.search}`, {
      replace: true,
      state: null,
    });
  }, [
    assets,
    location.pathname,
    location.search,
    location.state,
    navigate,
    preview.collectionRef,
  ]);

  const mediaFilters = (
    <div className={styles.filters}>
      <nav className={styles.kindGroup} aria-label={t("curation.filters")}>
        {(["all", "image", "video"] as const).map((value) => (
          <Button
            key={value}
            onClick={() => {
              const next = new URLSearchParams(searchParams);
              if (value === "all") next.delete("kind"); else next.set("kind", value);
              setSearchParams(next);
            }}
            aria-pressed={kind === value}
            size="small"
            variant={kind === value ? "secondary" : "quiet"}
          >
            {t(`curation.kind.${value}`)}
            {!showingTags && <small>{favoritesQuery.data?.pages[0]?.counts[value === "all" ? "all" : value === "image" ? "images" : "videos"] ?? "—"}</small>}
          </Button>
        ))}
      </nav>
      {showingTags && <div className={styles.densitySlot} ref={setSizeControlContainer} />}
      <label>
        <span>{t("curation.sort")}</span>
        <Select
          onChange={(event) => {
            const next = new URLSearchParams(searchParams);
            next.set("sort", event.target.value);
            setSearchParams(next);
          }}
          value={sort}
        >
          {!showingTags && <option value="favoritedAt">{t("curation.sortFavorited")}</option>}
          <option value="modifiedAt">{t("curation.sortModified")}</option>
          <option value="name">{t("curation.sortName")}</option>
          <option value="size">{t("curation.sortSize")}</option>
        </Select>
      </label>
    </div>
  );

  return (
    <AppShell
      autoHideChrome={layout === "seamless" && mediaItems.length > 0}
      active="browse"
      browseHref={paths.root}
      homeHref={paths.root}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      logoutPending={logoutPending}
      navigationKey={`${libraryId ?? "none"}:${showingTags ? tagId ?? "tags" : "favorites"}`}
      onLogout={onLogout}
      navigationLabel={t(showingTags ? "context.openTags" : "context.openFavorites")}
      sidebarLabel={t(showingTags ? "context.tags" : "context.favorites")}
      searchHref={paths.search}
      settingsHref={paths.generalSettings}
      topbarContent={<span>{currentLibrary?.name ?? heading}</span>}
      sidebarContent={
        libraryId ? (
          <DirectoryNavigation
            activeQuickAccess={showingTags ? "tags" : "favorites"}
            browseState={defaultBrowseUrlState()}
            currentLibraryName={currentLibrary?.name}
            {...(favoriteCount !== undefined ? { favoriteCount } : {})}
            hideDirectoryTree
            detailLayout={showingTags ? "fill" : "scroll"}
            libraries={selectableLibraries}
            libraryId={libraryId}
            onLibraryChange={(nextLibraryId) => {
              const next = new URLSearchParams({ libraryId: nextLibraryId });
              if (showingTags) next.set("kind", "image");
              navigate(`${showingTags ? paths.tags : paths.favorites}?${next.toString()}`);
            }}
            quickAccessDetail={<div className={showingTags ? styles.tagTools : undefined}>
              {showingTags && <LibraryTagCloud key={libraryId} libraryId={libraryId} {...(tagId ? { selectedTagId: tagId } : {})} />}
              {!showingTags && <ContextSidebar title={t("context.favorites")} description={t("context.favoriteScope")}>
                {mediaFilters}
              </ContextSidebar>}
            </div>}
            selectedPathIds={selectedPathIds}
          />
        ) : undefined
      }
      title={heading}
    >
      {deletion.dialog}
      <section className={styles.page} aria-labelledby="curation-heading">
        <div
          className={styles.workspace}
          data-has-preview={previewItem ? "" : undefined}
          style={{ "--preview-width": `${preview.width}px` } as CSSProperties}
        >
        <div className={styles.results}>
        <header className={styles.header}>
          <p>{t("curation.quickAccess")}</p>
          <h1 id="curation-heading">{heading}</h1>
          <span>{t("curation.loadedCount").replace("{count}", String(assets.length))}</span>
        {selection.toolbar}
        </header>
        {showingTags && <section className={styles.tagResultTools} aria-label={t("context.mediaFilters")}>
          {mediaFilters}
        </section>}
        <p className={styles.applied} aria-label={t("context.resultsFilters")}>
          {t(`curation.kind.${kind}`)} · {t(sort === "favoritedAt" ? "curation.sortFavorited" : sort === "name" ? "curation.sortName" : sort === "size" ? "curation.sortSize" : "curation.sortModified")}
        </p>

          {!libraryId ? (
            librariesQuery.isPending || (Boolean(requestedLibraryId) && requestedLibraryQuery.isPending) ? <LoadingState /> :
            librariesQuery.isError || hasRequestedLibrary ? <>
              <ErrorState message={t("tagCloud.libraryFailed")} onRetry={() => {
                void librariesQuery.refetch();
                if (requestedLibraryId) void requestedLibraryQuery.refetch();
              }} />
              <Button variant="secondary" onClick={() => navigate(paths.libraries)}>{t("tagCloud.openLibraries")}</Button>
            </> : <EmptyState title={t("tagCloud.noLibrary")} description={t("tagCloud.chooseDescription")}
              action={<Button onClick={() => navigate(paths.libraries)}>{t("tagCloud.openLibraries")}</Button>} />
          ) : showingTags && !tagId ? (
            <EmptyState
              description={t("tagCloud.chooseDescription")}
              title={t("tagCloud.choose")}
            />
          ) : activeQuery.isPending ? (
            <MediaCollectionSkeleton label={t("curation.loadingAssets")} />
          ) : activeQuery.isError && assets.length === 0 ? (
            <ErrorState
              message={t("curation.assetsFailed")}
              onRetry={() => void activeQuery.refetch()}
            />
          ) : assets.length === 0 ? (
            <EmptyState
              description={showingTags ? t("curation.emptyTagDescription") : t("curation.emptyFavoritesDescription")}
              title={showingTags ? t("curation.emptyTag") : t("curation.emptyFavorites")}
            />
          ) : (
            <MediaCollection
              ref={preview.collectionRef}
              {...(selection.mediaSelection ? { selection: selection.mediaSelection } : {})}
              {...(showingTags ? { sizeControlContainer } : {})}
              {...(favoriteMutation.variables?.assetId
                ? { favoritePendingId: favoriteMutation.variables.assetId }
                : {})}
              hasNextPage={activeQuery.hasNextPage}
              isFetchingNextPage={activeQuery.isFetching}
              items={mediaItems}
              labels={{
                activatePreview: preview.pinned
                  ? t("browse.selectPinnedPreview")
                  : t("browse.activatePreview"),
                addFavorite: t("curation.addFavorite"),
                animated: t("browse.kindAnimated"),
                failedThumbnail: t("browse.thumbnailFailed"),
                image: t("browse.kindImage"),
                loadMore: t("browse.loadMoreMedia"),
                loadMoreFailed: t("browse.loadMoreMediaFailed"),
                loadingMore: t("browse.loadingMoreMedia"),
                pendingThumbnail: t("browse.thumbnailPending"),
                tileSize: t("browse.tileSize"),
                previewing: t("browse.currentlyPreviewing"),
                removeFavorite: t("curation.removeFavorite"),
                retryLoadMore: t("browse.retryLoadMoreMedia"),
                unavailableThumbnail: t("browse.thumbnailUnavailable"),
                video: t("browse.kindVideo"),
              }}
              layout={layout}
              onFavoriteToggle={(assetId, favorite) =>
                favoriteMutation.mutate({ assetId, csrfToken: session.csrfToken, favorite })
              }
              onItemActivate={(assetId, activation) =>
                preview.activate(assetId, activation)
              }
              onLoadMore={() => void activeQuery.fetchNextPage()}
              onRetryLoadMore={() =>
                void retryInfiniteNextPage({
                  error: activeQuery.error,
                  loadNextPage: activeQuery.fetchNextPage,
                  refresh: activeQuery.refetch,
                })
              }
              paginationError={activeQuery.isFetchNextPageError}
              {...(previewAsset
                ? { previewItemId: previewAsset.id }
                : {})}
              {...(preview.selectedItemId
                ? { selectedItemId: preview.selectedItemId }
                : {})}
            />
          )}
        </div>
        {previewItem && (
          <MediaPreview
            autoPlayVideo={preview.autoPlayVideo}
            muteVideo={preview.muteVideo}
            availability={
              previewAsset && mediaAvailability(previewAsset)
                ? mediaAvailabilityPresentation(
                    mediaAvailability(previewAsset)!,
                    t,
                    () => void activeQuery.refetch(),
                  )
                : undefined
            }
            canGoNext={
              preview.previewIndex >= 0 &&
              preview.previewIndex < assets.length - 1
            }
            canGoPrevious={preview.previewIndex > 0}
            metadataContent={<AssetExifPanel assetId={previewItem.id} kind={previewItem.kind} />}
              curationContent={
              previewAsset ? (
                <><AssetCurationControls
                  assetId={previewAsset.id}
                  csrfToken={session.csrfToken}
                /><AssetDeletionAction flow={deletion} asset={previewAsset} onAccepted={preview.close} /></>
              ) : undefined
            }
            item={previewItem}
            labels={{
              close: t("browse.closePreview"),
              followingDescription: t("browse.previewFollowingDescription"),
              followingTitle: t("browse.previewFollowingTitle"),
              imageFailed: t("browse.previewImageFailed"),
              loadFailedDescription: t("mediaState.loadFailedDescription"),
              next: t("browse.nextMedia"),
              openViewer: t("browse.openFullViewer"),
              pin: t("browse.pinPreview"),
              pinnedDescription: t("browse.previewPinnedDescription"),
              pinnedTitle: t("browse.previewPinnedTitle"),
              position: t("browse.previewPosition")
                .replace("{current}", String(preview.previewIndex + 1))
                .replace("{total}", String(assets.length)),
              previous: t("browse.previousMedia"),
              preview: t("browse.preview"),
              resize: t("browse.resizePreview"),
              retry: t("mediaState.retry"),
              unpin: t("browse.unpinPreview"),
              videoFailed: t("browse.previewVideoFailed"),
            }}
            maxWidth={preview.maxWidth}
            onClose={preview.close}
            onNext={() => preview.moveTo(assets[preview.previewIndex + 1])}
            onOpenViewer={() => {
              if (!previewAsset) return;
              const returnTo = `${location.pathname}${location.search}`;
              navigate(
                paths.media(previewAsset.libraryId, previewAsset.id, returnTo),
                {
                  state: createViewerLocationState(assets, returnTo),
                },
              );
            }}
            onPinnedChange={preview.updatePinned}
            onPrevious={() => preview.moveTo(assets[preview.previewIndex - 1])}
            onWidthChange={preview.setWidth}
            pinned={preview.pinned}
            width={preview.width}
          />
        )}
        </div>
      </section>
    </AppShell>
  );
}

function mapItems(assets: Asset[], locale: string): MediaCollectionItem[] {
  const formatter = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" });
  return assets.map((asset) => {
    const storyboard = mediaStoryboard(asset);
    return {
      favorite: asset.favorite ?? false,
      height: asset.height,
      id: asset.id,
      kind: asset.kind,
      modifiedLabel: formatter.format(new Date(asset.modifiedAt)),
      name: asset.name,
      ...(storyboard ? { storyboard } : {}),
      thumbnailStatus: asset.thumbnail.status,
      thumbnailUrl: asset.thumbnail.url,
      width: asset.width,
    };
  });
}
