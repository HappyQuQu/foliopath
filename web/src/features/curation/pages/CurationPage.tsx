import { useEffect, useMemo, useState, type CSSProperties } from "react";
import { Link, useLocation, useNavigate, useSearchParams } from "react-router-dom";

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
import { Button, EmptyState, ErrorState } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import {
  assetContentUrl,
  type Asset,
  type AssetKind,
  type AssetSort,
} from "../../../lib/api/catalog";
import type { Tag } from "../../../lib/api/curation";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
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
import { useLibrariesQuery } from "../../libraries";
import { AssetCurationControls } from "../components/AssetCurationControls";
import {
  useFavoriteMutation,
  useFavoritesQuery,
  useTagAssetsQuery,
  useTagsQuery,
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
  const location = useLocation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [layout] = useState(readMediaLayoutPreference);
  const showingTags = location.pathname.startsWith(paths.tags);
  const librariesQuery = useLibrariesQuery();
  const libraries = useMemo(
    () => librariesQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [librariesQuery.data],
  );
  const requestedLibraryId = searchParams.get("libraryId");
  const currentLibrary =
    libraries.find((library) => library.id === requestedLibraryId) ??
    libraries.find((library) => library.status === "ready") ??
    libraries[0];
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
  const tagsQuery = useTagsQuery();
  const favoritesQuery = useFavoritesQuery(queryOptions);
  const favoriteCount = favoritesQuery.data?.pages[0]?.counts.all;
  const tagAssetsQuery = useTagAssetsQuery(tagId, queryOptions);
  const favoriteMutation = useFavoriteMutation();
  const tags = useMemo(
    () => tagsQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [tagsQuery.data],
  );
  const selectedTag = tags.find((tag) => tag.id === tagId);
  const activeQuery = tagId ? tagAssetsQuery : favoritesQuery;
  const assets = useMemo(
    () => activeQuery.data?.pages.flatMap((page) => page.items.map((item) => item.asset)) ?? [],
    [activeQuery.data],
  );
  const mediaItems = useMemo(
    () => mapItems(assets, locale),
    [assets, locale],
  );
  const selectedPathIds = useMemo(() => new Set<string>(), []);
  const preview = useMediaPreviewController({
    items: assets,
    resetKey: showingTags ? `tag:${tagId ?? "index"}` : "favorites",
  });
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
            ...(mediaPosterUrl(previewAsset)
              ? { posterUrl: mediaPosterUrl(previewAsset) }
              : {}),
          }
        : undefined,
    [locale, previewAsset, t],
  );
  const heading = showingTags
    ? selectedTag?.name ?? t("curation.tags")
    : t("curation.favorites");

  useEffect(() => {
    if (!libraryId || requestedLibraryId === libraryId) return;
    const next = new URLSearchParams(searchParams);
    next.set("libraryId", libraryId);
    setSearchParams(next, { replace: true });
  }, [libraryId, requestedLibraryId, searchParams, setSearchParams]);

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

  return (
    <AppShell
      active="browse"
      browseHref={paths.root}
      homeHref={paths.root}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      logoutPending={logoutPending}
      onLogout={onLogout}
      searchHref={paths.search}
      settingsHref={paths.generalSettings}
      sidebarContent={
        libraryId ? (
          <DirectoryNavigation
            activeQuickAccess={showingTags ? "tags" : "favorites"}
            browseState={defaultBrowseUrlState()}
            currentLibraryName={currentLibrary?.name}
            {...(favoriteCount !== undefined ? { favoriteCount } : {})}
            hideDirectoryTree
            libraries={libraries}
            libraryId={libraryId}
            onLibraryChange={(nextLibraryId) => {
              const next = new URLSearchParams(searchParams);
              next.set("libraryId", nextLibraryId);
              setSearchParams(next);
            }}
            quickAccessDetail={
              showingTags ? (
                <TagWall
                  tags={tags}
                  hasMoreTags={tagsQuery.hasNextPage}
                  libraryId={libraryId}
                  loadingMoreTags={tagsQuery.isFetchingNextPage}
                  onLoadMoreTags={() => void tagsQuery.fetchNextPage()}
                  {...(tagId ? { selectedTagId: tagId } : {})}
                />
              ) : undefined
            }
            selectedPathIds={selectedPathIds}
          />
        ) : undefined
      }
      title={heading}
    >
      <section className={styles.page} aria-labelledby="curation-heading">
        <header className={styles.header}>
          <p>{t("curation.quickAccess")}</p>
          <h1 id="curation-heading">{heading}</h1>
          <span>{t("curation.loadedCount").replace("{count}", String(assets.length))}</span>
        </header>
        <div className={styles.filters} role="region" aria-label={t("curation.filters")}>
          <div className={styles.kindGroup}>
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
                variant="secondary"
              >
                {t(`curation.kind.${value}`)}
              </Button>
            ))}
          </div>
          <label>
            <span>{t("curation.sort")}</span>
            <select
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
            </select>
          </label>
        </div>
        <div
          className={styles.workspace}
          data-has-preview={previewItem ? "" : undefined}
          style={{ "--preview-width": `${preview.width}px` } as CSSProperties}
        >
        <div className={styles.results}>
          {showingTags && !tagId ? (
            <EmptyState
              description={t("curation.selectTagDescription")}
              title={t("curation.selectTag")}
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
              {...(favoriteMutation.variables?.assetId
                ? { favoritePendingId: favoriteMutation.variables.assetId }
                : {})}
              hasNextPage={activeQuery.hasNextPage}
              isFetchingNextPage={activeQuery.isFetchingNextPage}
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
            curationContent={
              previewAsset ? (
                <AssetCurationControls
                  assetId={previewAsset.id}
                  csrfToken={session.csrfToken}
                />
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

function TagWall({
  hasMoreTags,
  libraryId,
  loadingMoreTags,
  onLoadMoreTags,
  selectedTagId,
  tags,
}: {
  hasMoreTags: boolean;
  libraryId: string;
  loadingMoreTags: boolean;
  onLoadMoreTags: () => void;
  selectedTagId?: string;
  tags: Tag[];
}) {
  const { t } = useLocale();
  const maximumCount = Math.max(1, ...tags.map((tag) => tag.assetCount));
  return (
    <section className={styles.tagWall} aria-labelledby="tag-wall-heading">
      <h2 id="tag-wall-heading">{t("curation.allTags")}</h2>
      <nav aria-label={t("curation.allTags")}>
        {tags.map((tag) => (
          <Link
            aria-current={selectedTagId === tag.id ? "page" : undefined}
            className={styles.tagCloudLink}
            data-weight={tagWeight(tag.assetCount, maximumCount)}
            key={tag.id}
            to={`${paths.tag(tag.id)}?${new URLSearchParams({ libraryId }).toString()}`}
          >
            <span>{tag.name}</span>
            <small aria-label={t("curation.tagAssetCount").replace("{count}", String(tag.assetCount))}>
              {tag.assetCount}
            </small>
          </Link>
        ))}
        {tags.length === 0 && <span className={styles.noTags}>{t("curation.noTags")}</span>}
        {hasMoreTags && (
          <Button
            loading={loadingMoreTags}
            onClick={onLoadMoreTags}
            size="small"
            variant="quiet"
          >
            {t("curation.loadMoreTags")}
          </Button>
        )}
      </nav>
    </section>
  );
}

function tagWeight(count: number, maximumCount: number): 1 | 2 | 3 | 4 | 5 {
  if (maximumCount <= 1) return 1;
  const normalized = Math.log1p(count) / Math.log1p(maximumCount);
  return Math.min(5, Math.max(1, Math.ceil(normalized * 5))) as 1 | 2 | 3 | 4 | 5;
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
