import {
  ArrowClockwise,
  CaretDown,
  CaretRight,
  Check,
  CheckSquare,
  CircleNotch,
  Columns,
  Folder,
  GridFour,
  House,
  ImageSquare,
  MagnifyingGlass,
  Square,
} from "@phosphor-icons/react";
import { useQueryClient } from "@tanstack/react-query";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
} from "react";
import {
  Link,
  useLocation,
  useNavigate,
  useSearchParams,
} from "react-router-dom";

import { AppShell } from "../../../components/patterns/AppShell/AppShell";
import {
  MediaCollection,
  MediaCollectionSkeleton,
  type MediaCollectionItem,
  type MediaCollectionLayout,
} from "../../../components/patterns/MediaCollection/MediaCollection";
import {
  MediaPreview,
  type MediaPreviewItem,
} from "../../../components/patterns/MediaPreview/MediaPreview";
import { mediaPreviewDetails } from "../../../components/patterns/MediaPreview/mediaPreviewDetails";
import { useMediaPreviewController } from "../../../components/patterns/MediaPreview/useMediaPreviewController";
import {
  Button,
  EmptyState,
  ErrorState,
  IconButton,
  InlineStatus,
  LoadingState,
  OfflineState,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import {
  assetContentUrl,
  type Asset,
  type AssetCounts,
  type Breadcrumb,
  type Directory,
} from "../../../lib/api/catalog";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import {
  mediaAvailability,
  mediaAvailabilityPresentation,
  mediaPosterUrl,
  mediaStoryboard,
} from "../../../lib/media/availability";
import { retryInfiniteNextPage } from "../../../lib/query/retryInfiniteNextPage";
import {
  readMediaLayoutPreference,
  writeMediaLayoutPreference,
} from "../../../lib/storage/preferences";
import { paths } from "../../../routes/paths";
import {
  createViewerLocationState,
  readViewerReturnState,
} from "../../../lib/navigation/viewer";
import {
  refreshLibraryDetail,
  useLibrariesQuery,
  useLibraryQuery,
} from "../../libraries";
import type { LibrarySummary } from "../../../lib/api/libraries";
import {
  refreshCatalogScope,
  useAssetsQuery,
  useDirectoriesQuery,
  useDirectoryQuery,
} from "../queries";
import {
  browseUrl,
  defaultBrowseUrlState,
  kindsForBrowse,
  parseBrowseUrlState,
  serializeBrowseUrlState,
  type BrowseUrlState,
} from "../urlState";
import styles from "./BrowsePage.module.css";

export function BrowsePage({
  directoryId,
  libraryId,
  logoutPending,
  onLogout,
  session,
}: {
  directoryId?: string | undefined;
  libraryId: string;
  logoutPending?: boolean;
  onLogout?: () => Promise<void>;
  session: AuthenticatedSession;
}) {
  const { locale, t } = useLocale();
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const [mediaLayout, setMediaLayout] = useState<MediaCollectionLayout>(
    readMediaLayoutPreference,
  );
  const [directoryFilterDraft, setDirectoryFilterDraft] = useState("");
  const [manualRefreshPending, setManualRefreshPending] = useState(false);
  const browseState = useMemo(
    () => parseBrowseUrlState(searchParams),
    [searchParams],
  );
  const librariesQuery = useLibrariesQuery();
  const libraryQuery = useLibraryQuery(libraryId);
  const directoryQuery = useDirectoryQuery(directoryId);
  const childrenQuery = useDirectoriesQuery({
    libraryId,
    parentId: directoryId,
    q: browseState.q,
  });
  const browseKinds = kindsForBrowse(browseState.kind);
  const assetsQuery = useAssetsQuery({
    directoryId,
    kinds: browseKinds,
    libraryId,
    order: browseState.order,
    q: browseState.q,
    recursive: browseState.recursive,
    sort: browseState.sort,
  });
  const { refetch: refreshLibrary } = libraryQuery;
  const { refetch: refreshDirectory } = directoryQuery;
  const { fetchNextPage: loadMoreChildren, refetch: refreshChildren } =
    childrenQuery;
  const { fetchNextPage: loadMoreAssets, refetch: refreshAssets } = assetsQuery;
  const libraries = useMemo(
    () => librariesQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [librariesQuery.data],
  );
  const children = useMemo(
    () => childrenQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [childrenQuery.data],
  );
  const assets = useMemo(
    () => assetsQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [assetsQuery.data],
  );
  const mediaItems = useMemo<MediaCollectionItem[]>(() => {
    const formatter = new Intl.DateTimeFormat(locale, {
      dateStyle: "medium",
      timeStyle: "short",
    });
    return assets.map((asset) => {
      const storyboard = mediaStoryboard(asset);
      return {
        height: asset.height,
        id: asset.id,
        kind: asset.kind,
        modifiedLabel: formatter.format(new Date(asset.modifiedAt)),
        name: asset.name,
        ...(browseState.recursive
          ? {
              sourceHref: browseUrl(
                libraryId,
                asset.directoryId,
                defaultBrowseUrlState(),
              ),
              sourceLabel: t("browse.openSourceDirectory").replace(
                "{path}",
                sourceDirectory(asset.relativePath),
              ),
            }
          : {}),
        ...(storyboard ? { storyboard } : {}),
        thumbnailStatus: asset.thumbnail.status,
        thumbnailUrl: asset.thumbnail.url,
        width: asset.width,
      };
    });
  }, [assets, browseState.recursive, libraryId, locale, t]);
  const preview = useMediaPreviewController({
    items: assets,
    resetKey: `${libraryId}:${directoryId ?? "root"}`,
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
  const currentLibrary = libraryQuery.data?.library;
  const breadcrumbs: Breadcrumb[] = directoryQuery.data?.breadcrumbs ?? [];
  const selectedPathIds = useMemo(
    () => new Set(breadcrumbs.map((item) => item.id)),
    [breadcrumbs],
  );
  const currentName =
    directoryQuery.data?.name ??
    currentLibrary?.name ??
    t("browse.libraryFallback");
  const currentMediaCount = browseState.q
    ? assets.length
    : (directoryQuery.data?.directAssetCount ??
      currentLibrary?.assetCount ??
      0);
  const canonicalSearch = serializeBrowseUrlState(browseState);
  const navigationScope = `${libraryId}:${directoryId ?? "root"}`;
  const previousNavigationScope = useRef(navigationScope);

  useEffect(() => {
    setDirectoryFilterDraft(browseState.q);
  }, [browseState.q]);

  useEffect(() => {
    const q = directoryFilterDraft.trim();
    if (q === browseState.q) return;

    const timer = window.setTimeout(() => {
      const scopeChanged = !q || !browseState.q;
      const defaults = defaultBrowseUrlState(browseState.recursive, q);
      updateBrowseState({
        ...browseState,
        q,
        ...(scopeChanged ? { order: defaults.order, sort: defaults.sort } : {}),
      });
    }, 300);
    return () => window.clearTimeout(timer);
  }, [browseState, directoryFilterDraft]);

  const refreshCurrentScope = useCallback(async () => {
    await Promise.all([
      refreshCatalogScope(queryClient, {
        directoryId,
        kinds: browseKinds,
        libraryId,
        order: browseState.order,
        q: browseState.q,
        recursive: browseState.recursive,
        sort: browseState.sort,
      }),
      refreshLibraryDetail(queryClient, libraryId),
    ]);
  }, [
    browseKinds,
    browseState.order,
    browseState.q,
    browseState.recursive,
    browseState.sort,
    directoryId,
    libraryId,
    queryClient,
  ]);

  useEffect(() => {
    if (previousNavigationScope.current === navigationScope) return;
    previousNavigationScope.current = navigationScope;
    void refreshCurrentScope();
  }, [navigationScope, refreshCurrentScope]);

  useEffect(() => {
    if (searchParams.toString() !== canonicalSearch) {
      setSearchParams(canonicalSearch, { replace: true });
    }
  }, [canonicalSearch, searchParams, setSearchParams]);

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

  function updateBrowseState(nextState: BrowseUrlState) {
    setSearchParams(serializeBrowseUrlState(nextState));
  }

  function updateMediaLayout(nextLayout: MediaCollectionLayout) {
    setMediaLayout(nextLayout);
    writeMediaLayoutPreference(nextLayout);
  }

  return (
    <AppShell
      active="browse"
      browseHref={browseUrl(libraryId, directoryId, browseState)}
      homeHref={paths.root}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      logoutPending={logoutPending}
      onLogout={onLogout}
      searchHref={
        directoryId
          ? `${paths.librarySearch(libraryId)}?${new URLSearchParams({
              directoryId,
              scope: "directory",
            }).toString()}`
          : paths.librarySearch(libraryId)
      }
      settingsHref={paths.generalSettingsForLibrary(libraryId)}
      showIdentityLabel={false}
      sidebarContent={
        <DirectoryNavigation
          currentLibraryName={currentLibrary?.name}
          libraries={libraries}
          libraryId={libraryId}
          browseState={browseState}
          onLibraryChange={(nextLibraryId) =>
            navigate(
              browseUrl(nextLibraryId, undefined, defaultBrowseUrlState()),
            )
          }
          selectedDirectoryId={directoryId}
          selectedPathIds={selectedPathIds}
        />
      }
      topbarContent={
        <Breadcrumbs
          browseState={browseState}
          breadcrumbs={breadcrumbs}
          libraryId={libraryId}
          rootName={currentLibrary?.name ?? currentName}
        />
      }
      title={t("browse.title")}
    >
      <section className={styles.page} aria-labelledby="browse-heading">
        <BrowseToolbar
          counts={assetsQuery.data?.pages[0]?.counts}
          browseState={browseState}
          directoryFilter={directoryFilterDraft}
          mediaLayout={mediaLayout}
          onChange={updateBrowseState}
          onDirectoryFilterChange={setDirectoryFilterDraft}
          onLayoutChange={updateMediaLayout}
          onRefresh={async () => {
            setManualRefreshPending(true);
            try {
              await refreshCurrentScope();
            } finally {
              setManualRefreshPending(false);
            }
          }}
          refreshPending={manualRefreshPending}
        />
        {currentLibrary?.status === "offline" && (
          <div className={styles.banner}>
            <InlineStatus tone="warning">
              {t("browse.offlinePreserved")}
            </InlineStatus>
          </div>
        )}
        {currentLibrary?.status === "scanning" && (
          <div className={styles.scanBanner}>
            <CircleNotch
              aria-hidden="true"
              className={styles.scanIcon}
              size={18}
            />
            <span>
              {t("browse.scanningBanner").replace(
                "{name}",
                currentLibrary.name,
              )}
            </span>
            <span className={styles.scanStats}>
              {t("browse.mediaCount").replace(
                "{count}",
                String(currentLibrary.assetCount),
              )}
            </span>
          </div>
        )}

        <div
          className={styles.workspace}
          data-has-preview={previewItem ? "" : undefined}
          style={
            {
              "--preview-sticky-top":
                "calc(var(--size-header) + var(--size-context-bar) * 2)",
              "--preview-width": `${preview.width}px`,
            } as CSSProperties
          }
        >
          <div className={styles.content}>
            {(libraryQuery.isPending ||
              (directoryId && directoryQuery.isPending)) && (
              <LoadingState label={t("browse.loadingLocation")} />
            )}
            {(libraryQuery.isError || directoryQuery.isError) && (
              <ErrorState
                message={t("browse.locationFailed")}
                onRetry={() => {
                  void refreshLibrary();
                  void refreshDirectory();
                }}
              />
            )}
            {libraryQuery.isSuccess &&
              (!directoryId || directoryQuery.isSuccess) && (
                <>
                  <h1 className={styles.screenReaderOnly} id="browse-heading">
                    {currentName}
                  </h1>
                  {(childrenQuery.isPending ||
                    childrenQuery.isError ||
                    children.length > 0 ||
                    browseState.q) && (
                    <section aria-labelledby="child-directories-heading">
                      <div className={styles.sectionHeading}>
                        <div>
                          <h2 id="child-directories-heading">
                            {t("browse.childDirectories")}
                          </h2>
                          <p>
                            {t("browse.directorySummary")
                              .replace("{directories}", String(children.length))
                              .replace("{media}", String(currentMediaCount))}
                          </p>
                        </div>
                      </div>
                      {childrenQuery.isPending && (
                        <LoadingState label={t("browse.loadingDirectories")} />
                      )}
                      {childrenQuery.isError && (
                        <ErrorState
                          message={t("browse.directoriesFailed")}
                          onRetry={() => void refreshChildren()}
                        />
                      )}
                      {children.length > 0 && (
                        <div className={styles.folderGrid}>
                          {children.map((directory) => (
                            <Link
                              className={styles.folderCard}
                              key={directory.id}
                              to={browseUrl(
                                libraryId,
                                directory.id,
                                browseState,
                              )}
                            >
                              <Folder
                                aria-hidden="true"
                                size={30}
                                weight="fill"
                              />
                              <strong>{directory.name}</strong>
                              <span>
                                {t("browse.mediaCount").replace(
                                  "{count}",
                                  String(directory.recursiveAssetCount),
                                )}
                              </span>
                            </Link>
                          ))}
                        </div>
                      )}
                      {childrenQuery.hasNextPage && (
                        <Button
                          className={styles.loadMore}
                          loading={childrenQuery.isFetchingNextPage}
                          onClick={() => void loadMoreChildren()}
                        >
                          {t("browse.loadMoreDirectories")}
                        </Button>
                      )}
                    </section>
                  )}
                  <section aria-labelledby="media-heading">
                    <div className={styles.sectionHeading}>
                      <div>
                        <h2 id="media-heading">{t("browse.mediaHeading")}</h2>
                        <p>
                          {browseState.recursive
                            ? t("browse.recursiveMediaDescription")
                            : t("browse.directMediaDescription")}
                        </p>
                      </div>
                      <span>
                        {t("browse.loadedMediaCount").replace(
                          "{count}",
                          String(assets.length),
                        )}
                      </span>
                    </div>
                    {assetsQuery.isPending && (
                      <MediaCollectionSkeleton
                        label={t("browse.loadingMedia")}
                      />
                    )}
                    {assetsQuery.isError && assets.length === 0 && (
                      <ErrorState
                        message={t("browse.mediaFailed")}
                        onRetry={() => void refreshAssets()}
                      />
                    )}
                    {assetsQuery.isSuccess &&
                      assets.length === 0 &&
                      currentLibrary?.status === "offline" && (
                        <OfflineState
                          description={t("browse.offlineEmptyDescription")}
                          title={t("browse.offlineEmptyTitle")}
                        />
                      )}
                    {assetsQuery.isSuccess &&
                      assets.length === 0 &&
                      currentLibrary?.status !== "offline" && (
                        <EmptyMedia
                          browseState={browseState}
                          directory={directoryQuery.data}
                          onClearQuery={() => setDirectoryFilterDraft("")}
                          onEnableRecursive={() =>
                            updateBrowseState({
                              ...defaultBrowseUrlState(true, browseState.q),
                              kind: browseState.kind,
                            })
                          }
                        />
                      )}
                    {assets.length > 0 && (
                      <MediaCollection
                        ref={preview.collectionRef}
                        hasNextPage={assetsQuery.hasNextPage}
                        isFetchingNextPage={
                          assetsQuery.isFetchingNextPage ||
                          (assetsQuery.isFetchNextPageError &&
                            assetsQuery.isRefetching)
                        }
                        items={mediaItems}
                        labels={{
                          activatePreview: preview.pinned
                            ? t("browse.selectPinnedPreview")
                            : t("browse.activatePreview"),
                          animated: t("browse.kindAnimated"),
                          failedThumbnail: t("browse.thumbnailFailed"),
                          image: t("browse.kindImage"),
                          loadMore: t("browse.loadMoreMedia"),
                          loadMoreFailed: t("browse.loadMoreMediaFailed"),
                          loadingMore: t("browse.loadingMoreMedia"),
                          pendingThumbnail: t("browse.thumbnailPending"),
                          previewing: t("browse.currentlyPreviewing"),
                          retryLoadMore: t("browse.retryLoadMoreMedia"),
                          unavailableThumbnail: t(
                            "browse.thumbnailUnavailable",
                          ),
                          video: t("browse.kindVideo"),
                        }}
                        layout={mediaLayout}
                        onItemActivate={(assetId, activation) =>
                          preview.activate(assetId, activation)
                        }
                        onLoadMore={loadMoreAssets}
                        onRetryLoadMore={() =>
                          void retryInfiniteNextPage({
                            error: assetsQuery.error,
                            loadNextPage: assetsQuery.fetchNextPage,
                            refresh: assetsQuery.refetch,
                          })
                        }
                        paginationError={assetsQuery.isFetchNextPageError}
                        {...(previewAsset
                          ? { previewItemId: previewAsset.id }
                          : {})}
                        {...(preview.selectedItemId
                          ? { selectedItemId: preview.selectedItemId }
                          : {})}
                      />
                    )}
                  </section>
                </>
              )}
          </div>
          {previewItem && (
            <MediaPreview
              availability={
                previewAsset && mediaAvailability(previewAsset)
                  ? mediaAvailabilityPresentation(
                      mediaAvailability(previewAsset)!,
                      t,
                      () => void refreshAssets(),
                    )
                  : undefined
              }
              canGoNext={
                preview.previewIndex >= 0 &&
                preview.previewIndex < assets.length - 1
              }
              canGoPrevious={preview.previewIndex > 0}
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
                  paths.media(
                    previewAsset.libraryId,
                    previewAsset.id,
                    returnTo,
                  ),
                  {
                    state: createViewerLocationState(assets, returnTo),
                  },
                );
              }}
              onPinnedChange={preview.updatePinned}
              onPrevious={() =>
                preview.moveTo(assets[preview.previewIndex - 1])
              }
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

function BrowseToolbar({
  browseState,
  counts,
  directoryFilter,
  mediaLayout,
  onChange,
  onDirectoryFilterChange,
  onLayoutChange,
  onRefresh,
  refreshPending,
}: {
  browseState: BrowseUrlState;
  counts?: AssetCounts | undefined;
  directoryFilter: string;
  mediaLayout: MediaCollectionLayout;
  onChange: (state: BrowseUrlState) => void;
  onDirectoryFilterChange: (query: string) => void;
  onLayoutChange: (layout: MediaCollectionLayout) => void;
  onRefresh: () => Promise<void>;
  refreshPending: boolean;
}) {
  const { locale, t } = useLocale();
  const sortValue = `${browseState.sort}:${browseState.order}`;
  const defaults = {
    ...defaultBrowseUrlState(browseState.recursive, browseState.q),
    ...(browseState.allMedia ? { allMedia: true as const } : {}),
    kind: browseState.kind,
  };
  const customSort =
    browseState.sort !== defaults.sort || browseState.order !== defaults.order;

  return (
    <div
      className={styles.toolbar}
      role="region"
      aria-label={t("browse.tools")}
    >
      <Button
        aria-pressed={browseState.recursive}
        className={styles.recursiveToggle}
        onClick={() =>
          onChange({
            ...defaultBrowseUrlState(!browseState.recursive, browseState.q),
            kind: browseState.kind,
          })
        }
        variant="secondary"
      >
        {browseState.recursive ? (
          <CheckSquare aria-hidden="true" size={18} weight="fill" />
        ) : (
          <Square aria-hidden="true" size={18} />
        )}
        {t("browse.includeSubdirectories")}
      </Button>
      <div
        aria-label={t("browse.mediaType")}
        className={styles.kindControls}
        role="radiogroup"
      >
        {(["all", "image", "video"] as const).map((kind) => (
          <Button
            aria-label={
              counts
                ? `${
                    kind === "all"
                      ? t("browse.kindAll")
                      : kind === "image"
                        ? t("browse.kindImage")
                        : t("browse.kindVideo")
                  } ${new Intl.NumberFormat(locale).format(
                    kind === "all"
                      ? counts.all
                      : kind === "image"
                        ? counts.images
                        : counts.videos,
                  )}`
                : undefined
            }
            aria-checked={browseState.kind === kind}
            className={styles.kindButton}
            key={kind}
            onClick={() => onChange({ ...browseState, kind })}
            role="radio"
            size="small"
            variant={browseState.kind === kind ? "secondary" : "quiet"}
          >
            <span>
              {kind === "all"
                ? t("browse.kindAll")
                : kind === "image"
                  ? t("browse.kindImage")
                  : t("browse.kindVideo")}
            </span>
            {counts && (
              <span className={styles.kindCount}>
                {new Intl.NumberFormat(locale).format(
                  kind === "all"
                    ? counts.all
                    : kind === "image"
                      ? counts.images
                      : counts.videos,
                )}
              </span>
            )}
          </Button>
        ))}
      </div>
      <label className={styles.directoryFilter}>
        <MagnifyingGlass aria-hidden="true" size={17} />
        <span className={styles.screenReaderOnly}>
          {t("browse.filterCurrentDirectory")}
        </span>
        <input
          autoComplete="off"
          onChange={(event) =>
            onDirectoryFilterChange(event.currentTarget.value)
          }
          placeholder={t("browse.filterCurrentDirectory")}
          type="search"
          value={directoryFilter}
        />
      </label>
      <div
        className={styles.layoutControls}
        role="group"
        aria-label={t("browse.layout")}
      >
        <IconButton
          label={t("browse.layoutGrid")}
          onClick={() => onLayoutChange("grid")}
          pressed={mediaLayout === "grid"}
        >
          <GridFour aria-hidden="true" size={18} />
        </IconButton>
        <IconButton
          label={t("browse.layoutMasonry")}
          onClick={() => onLayoutChange("masonry")}
          pressed={mediaLayout === "masonry"}
        >
          <Columns aria-hidden="true" size={18} />
        </IconButton>
      </div>
      <label className={styles.sortControl}>
        <span>{t("browse.sort")}</span>
        <select
          value={sortValue}
          onChange={(event) => {
            const [sort, order] = event.target.value.split(":") as [
              BrowseUrlState["sort"],
              BrowseUrlState["order"],
            ];
            onChange({ ...browseState, order, sort });
          }}
        >
          <option value="name:asc">{t("browse.sortNameAscending")}</option>
          <option value="name:desc">{t("browse.sortNameDescending")}</option>
          <option value="modifiedAt:desc">
            {t("browse.sortModifiedDescending")}
          </option>
          <option value="modifiedAt:asc">
            {t("browse.sortModifiedAscending")}
          </option>
          <option value="size:desc">{t("browse.sortSizeDescending")}</option>
          <option value="size:asc">{t("browse.sortSizeAscending")}</option>
        </select>
      </label>
      {customSort && (
        <Button
          className={styles.resetSort}
          onClick={() => onChange(defaults)}
          size="small"
          variant="quiet"
        >
          {t("browse.resetSort")}
        </Button>
      )}
      <Button
        aria-label={t("browse.refresh")}
        className={styles.refresh}
        loading={refreshPending}
        onClick={() => void onRefresh()}
        size="small"
        variant="quiet"
      >
        <ArrowClockwise aria-hidden="true" size={17} />
      </Button>
    </div>
  );
}

function EmptyMedia({
  browseState,
  directory,
  onClearQuery,
  onEnableRecursive,
}: {
  browseState: BrowseUrlState;
  directory?: Directory | undefined;
  onClearQuery: () => void;
  onEnableRecursive: () => void;
}) {
  const { t } = useLocale();
  const hasDescendantMedia =
    !browseState.recursive &&
    browseState.kind === "all" &&
    directory !== undefined &&
    directory.recursiveAssetCount > directory.directAssetCount;

  return (
    <EmptyState
      action={
        browseState.q ? (
          <Button onClick={onClearQuery}>
            {t("browse.clearDirectoryFilter")}
          </Button>
        ) : hasDescendantMedia ? (
          <Button onClick={onEnableRecursive}>
            {t("browse.enableRecursive")}
          </Button>
        ) : undefined
      }
      description={
        browseState.q
          ? t("browse.noDirectoryFilterResultsDescription")
          : hasDescendantMedia
            ? t("browse.descendantMediaAvailable").replace(
                "{count}",
                String(
                  directory.recursiveAssetCount - directory.directAssetCount,
                ),
              )
            : t("browse.noMediaDescription")
      }
      icon={ImageSquare}
      title={
        browseState.q
          ? t("browse.noDirectoryFilterResults")
          : browseState.recursive
            ? t("browse.noRecursiveMedia")
            : t("browse.noDirectMedia")
      }
    />
  );
}

function sourceDirectory(relativePath: string): string {
  const separator = relativePath.lastIndexOf("/");
  return separator > 0 ? relativePath.slice(0, separator) : "/";
}

function Breadcrumbs({
  browseState,
  breadcrumbs,
  libraryId,
  rootName,
}: {
  browseState: BrowseUrlState;
  breadcrumbs: Breadcrumb[];
  libraryId: string;
  rootName: string;
}) {
  const { t } = useLocale();
  const items =
    breadcrumbs.length > 1
      ? breadcrumbs.slice(1)
      : breadcrumbs.length > 0
        ? breadcrumbs
        : [{ id: "root", name: rootName, relativePath: "" }];

  return (
    <nav aria-label={t("browse.breadcrumbs")} className={styles.breadcrumbs}>
      <House aria-hidden="true" size={17} />
      {breadcrumbs.length > 1 && (
        <span className={styles.screenReaderOnly}>{rootName}</span>
      )}
      {items.map((item, index) => {
        const current = index === items.length - 1;
        return (
          <span
            className={styles.crumb}
            data-first={index === 0 ? "" : undefined}
            key={item.id}
          >
            {index > 0 && <CaretRight aria-hidden="true" size={14} />}
            {current ? (
              <span aria-current="page">{item.name}</span>
            ) : (
              <Link
                to={browseUrl(
                  libraryId,
                  item.relativePath ? item.id : undefined,
                  browseState,
                )}
              >
                {item.name}
              </Link>
            )}
          </span>
        );
      })}
    </nav>
  );
}

function DirectoryNavigation({
  browseState,
  currentLibraryName,
  libraries,
  libraryId,
  onLibraryChange,
  selectedDirectoryId,
  selectedPathIds,
}: {
  browseState: BrowseUrlState;
  currentLibraryName?: string | undefined;
  libraries: LibrarySummary[];
  libraryId: string;
  onLibraryChange: (libraryId: string) => void;
  selectedDirectoryId?: string | undefined;
  selectedPathIds: Set<string>;
}) {
  const { t } = useLocale();
  const [libraryMenuOpen, setLibraryMenuOpen] = useState(false);
  const libraryPickerRef = useRef<HTMLDivElement>(null);
  const libraryTriggerRef = useRef<HTMLButtonElement>(null);
  const libraryOptionRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const currentLibrary = libraries.find((library) => library.id === libraryId);
  const allMediaSelected =
    !selectedDirectoryId && browseState.allMedia === true;
  const libraryRootSelected = !selectedDirectoryId && !allMediaSelected;

  function statusLabel(library: LibrarySummary | undefined) {
    return library?.status === "ready"
      ? t("libraries.statusReady")
      : library?.status === "scanning"
        ? t("libraries.statusScanning")
        : library?.status === "offline"
          ? t("libraries.statusOffline")
          : library?.status === "error"
            ? t("libraries.statusError")
            : t("libraries.statusPending");
  }

  function focusLibraryOption(index: number) {
    const options = libraryOptionRefs.current.filter(
      (option): option is HTMLButtonElement => option !== null,
    );
    if (!options.length) return;
    options[Math.min(options.length - 1, Math.max(0, index))]?.focus();
  }

  function openLibraryMenu(direction: "first" | "selected" | "last") {
    setLibraryMenuOpen(true);
    requestAnimationFrame(() => {
      const selectedIndex = Math.max(
        0,
        libraries.findIndex((library) => library.id === libraryId),
      );
      focusLibraryOption(
        direction === "first"
          ? 0
          : direction === "last"
            ? libraries.length - 1
            : selectedIndex,
      );
    });
  }

  useEffect(() => {
    if (!libraryMenuOpen) return;
    function closeOnOutsideClick(event: MouseEvent) {
      if (
        event.target instanceof Node &&
        !libraryPickerRef.current?.contains(event.target)
      ) {
        setLibraryMenuOpen(false);
      }
    }
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key !== "Escape") return;
      event.preventDefault();
      setLibraryMenuOpen(false);
      libraryTriggerRef.current?.focus();
    }
    document.addEventListener("mousedown", closeOnOutsideClick);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("mousedown", closeOnOutsideClick);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [libraryMenuOpen]);

  const currentLibraryLabel =
    currentLibrary?.name ?? currentLibraryName ?? t("browse.libraryFallback");
  const currentStatusLabel = statusLabel(currentLibrary);

  return (
    <div className={styles.directoryNavigation}>
      <div
        className={styles.libraryPicker}
        onBlur={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget)) {
            setLibraryMenuOpen(false);
          }
        }}
        ref={libraryPickerRef}
      >
        <span className={styles.libraryLabel}>{t("browse.library")}</span>
        <button
          aria-controls="library-picker-options"
          aria-expanded={libraryMenuOpen}
          aria-haspopup="listbox"
          aria-label={`${t("browse.library")}：${currentLibraryLabel}`}
          className={styles.libraryTrigger}
          onClick={() => {
            if (libraryMenuOpen) {
              setLibraryMenuOpen(false);
            } else {
              openLibraryMenu("selected");
            }
          }}
          onKeyDown={(event) => {
            if (event.key === "ArrowDown") {
              event.preventDefault();
              openLibraryMenu("first");
            } else if (event.key === "ArrowUp") {
              event.preventDefault();
              openLibraryMenu("last");
            }
          }}
          ref={libraryTriggerRef}
          type="button"
        >
          <span className={styles.libraryTriggerCopy}>
            <strong>{currentLibraryLabel}</strong>
          </span>
          <CaretDown
            aria-hidden="true"
            className={styles.libraryChevron}
            data-open={libraryMenuOpen || undefined}
            size={18}
            weight="bold"
          />
        </button>
        <span className={styles.libraryStatus}>
          <span
            className={styles.libraryStatusDot}
            data-status={currentLibrary?.status}
          />
          {currentStatusLabel}
          {currentLibrary
            ? ` · ${t("browse.mediaCount").replace(
                "{count}",
                String(currentLibrary.assetCount),
              )}`
            : ""}
        </span>
        {libraryMenuOpen && (
          <div
            aria-label={t("browse.library")}
            className={styles.libraryMenu}
            id="library-picker-options"
            onKeyDown={(event) => {
              const options = libraryOptionRefs.current.filter(
                (option): option is HTMLButtonElement => option !== null,
              );
              if (!options.length) return;
              const currentIndex = options.indexOf(
                document.activeElement as HTMLButtonElement,
              );
              if (event.key === "ArrowDown") {
                event.preventDefault();
                focusLibraryOption((currentIndex + 1) % options.length);
              } else if (event.key === "ArrowUp") {
                event.preventDefault();
                focusLibraryOption(
                  (currentIndex - 1 + options.length) % options.length,
                );
              } else if (event.key === "Home") {
                event.preventDefault();
                focusLibraryOption(0);
              } else if (event.key === "End") {
                event.preventDefault();
                focusLibraryOption(options.length - 1);
              }
            }}
            role="listbox"
          >
            {libraries.map((library, index) => {
              const selected = library.id === libraryId;
              return (
                <button
                  aria-selected={selected}
                  className={styles.libraryOption}
                  key={library.id}
                  onClick={() => {
                    setLibraryMenuOpen(false);
                    if (!selected) onLibraryChange(library.id);
                  }}
                  ref={(element) => {
                    libraryOptionRefs.current[index] = element;
                  }}
                  role="option"
                  type="button"
                >
                  <span className={styles.libraryOptionCopy}>
                    <strong>{library.name}</strong>
                    <span className={styles.libraryStatus}>
                      <span
                        className={styles.libraryStatusDot}
                        data-status={library.status}
                      />
                      {statusLabel(library)}
                      {` · ${t("browse.mediaCount").replace(
                        "{count}",
                        String(library.assetCount),
                      )}`}
                    </span>
                  </span>
                  {selected && (
                    <Check aria-hidden="true" size={19} weight="bold" />
                  )}
                </button>
              );
            })}
          </div>
        )}
      </div>
      <p className={styles.treeLabel}>{t("browse.directory")}</p>
      <nav aria-label={t("browse.directoryNavigation")} className={styles.tree}>
        <Link
          aria-current={allMediaSelected ? "page" : undefined}
          className={`${styles.treeLink} ${styles.treeAllMedia} ${allMediaSelected ? styles.treeLinkCurrent : ""}`}
          to={browseUrl(libraryId, undefined, {
            ...defaultBrowseUrlState(true),
            allMedia: true,
          })}
        >
          <ImageSquare aria-hidden="true" size={18} />
          <span>{t("browse.allMedia")}</span>
        </Link>
        <div
          className={styles.treeRow}
          style={{ "--tree-depth": 0 } as CSSProperties}
        >
          <span className={styles.treeToggleStatic}>
            <CaretDown aria-hidden="true" size={15} />
          </span>
          <Link
            aria-current={libraryRootSelected ? "page" : undefined}
            className={`${styles.treeLink} ${libraryRootSelected ? styles.treeLinkCurrent : ""}`}
            to={browseUrl(libraryId, undefined, defaultBrowseUrlState())}
          >
            <Folder aria-hidden="true" size={17} />
            <span>{currentLibraryName ?? t("browse.libraryFallback")}</span>
          </Link>
        </div>
        <DirectoryChildren
          browseState={browseState}
          depth={1}
          libraryId={libraryId}
          selectedDirectoryId={selectedDirectoryId}
          selectedPathIds={selectedPathIds}
        />
      </nav>
    </div>
  );
}

function DirectoryChildren({
  browseState,
  depth,
  libraryId,
  parentId,
  selectedDirectoryId,
  selectedPathIds,
}: {
  browseState: BrowseUrlState;
  depth: number;
  libraryId: string;
  parentId?: string | undefined;
  selectedDirectoryId?: string | undefined;
  selectedPathIds: Set<string>;
}) {
  const { t } = useLocale();
  const query = useDirectoriesQuery({ libraryId, parentId });
  const { fetchNextPage: loadMoreDirectories, refetch: refreshDirectories } =
    query;
  const directories = useMemo(
    () => query.data?.pages.flatMap((page) => page.items) ?? [],
    [query.data],
  );

  if (query.isPending) {
    return (
      <span className={styles.treeState}>{t("browse.loadingDirectories")}</span>
    );
  }
  if (query.isError) {
    return (
      <Button
        size="small"
        variant="quiet"
        onClick={() => void refreshDirectories()}
      >
        {t("common.retry")}
      </Button>
    );
  }

  return (
    <>
      {directories.map((directory) => (
        <DirectoryTreeItem
          browseState={browseState}
          depth={depth}
          directory={directory}
          key={directory.id}
          libraryId={libraryId}
          selectedDirectoryId={selectedDirectoryId}
          selectedPathIds={selectedPathIds}
        />
      ))}
      {query.hasNextPage && (
        <Button
          loading={query.isFetchingNextPage}
          size="small"
          variant="quiet"
          onClick={() => void loadMoreDirectories()}
        >
          {t("browse.loadMoreDirectories")}
        </Button>
      )}
    </>
  );
}

function DirectoryTreeItem({
  browseState,
  depth,
  directory,
  libraryId,
  selectedDirectoryId,
  selectedPathIds,
}: {
  browseState: BrowseUrlState;
  depth: number;
  directory: Directory;
  libraryId: string;
  selectedDirectoryId?: string | undefined;
  selectedPathIds: Set<string>;
}) {
  const { t } = useLocale();
  const selected = selectedDirectoryId === directory.id;
  const onSelectedPath = selectedPathIds.has(directory.id);
  const [expanded, setExpanded] = useState(onSelectedPath);

  useEffect(() => {
    if (onSelectedPath) setExpanded(true);
  }, [onSelectedPath]);

  return (
    <div>
      <div
        className={styles.treeRow}
        style={{ "--tree-depth": depth } as CSSProperties}
      >
        {directory.hasChildren ? (
          <IconButton
            className={styles.treeToggle}
            label={
              expanded
                ? t("browse.collapseDirectory").replace(
                    "{name}",
                    directory.name,
                  )
                : t("browse.expandDirectory").replace("{name}", directory.name)
            }
            onClick={() => setExpanded((value) => !value)}
          >
            {expanded ? (
              <CaretDown aria-hidden="true" size={15} />
            ) : (
              <CaretRight aria-hidden="true" size={15} />
            )}
          </IconButton>
        ) : (
          <span className={styles.treeSpacer} />
        )}
        <Link
          aria-current={selected ? "page" : undefined}
          className={`${styles.treeLink} ${selected ? styles.treeLinkCurrent : ""}`}
          to={browseUrl(libraryId, directory.id, browseState)}
        >
          <Folder aria-hidden="true" size={17} />
          <span>{directory.name}</span>
        </Link>
      </div>
      {expanded && directory.hasChildren && (
        <DirectoryChildren
          browseState={browseState}
          depth={depth + 1}
          libraryId={libraryId}
          parentId={directory.id}
          selectedDirectoryId={selectedDirectoryId}
          selectedPathIds={selectedPathIds}
        />
      )}
    </div>
  );
}
