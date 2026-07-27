import {
  CaretDown,
  CaretRight,
  CheckSquare,
  Columns,
  Copy,
  Folder,
  GridFour,
  House,
  ImageSquare,
  Square,
} from "@phosphor-icons/react";
import {
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
  useToast,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import {
  assetContentUrl,
  type Asset,
  type Breadcrumb,
  type Directory,
} from "../../../lib/api/catalog";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import {
  mediaAvailability,
  mediaAvailabilityPresentation,
  mediaPosterUrl,
} from "../../../lib/media/availability";
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
  useLibrariesQuery,
  useLibraryQuery,
} from "../../libraries";
import {
  useAssetsQuery,
  useDirectoriesQuery,
  useDirectoryQuery,
} from "../queries";
import {
  browseUrl,
  defaultBrowseUrlState,
  parseBrowseUrlState,
  serializeBrowseUrlState,
  type BrowseUrlState,
} from "../urlState";
import styles from "./BrowsePage.module.css";

export function BrowsePage({
  directoryId,
  libraryId,
  session,
}: {
  directoryId?: string | undefined;
  libraryId: string;
  session: AuthenticatedSession;
}) {
  const { locale, t } = useLocale();
  const location = useLocation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [mediaLayout, setMediaLayout] = useState<MediaCollectionLayout>(
    readMediaLayoutPreference,
  );
  const toast = useToast();
  const browseState = useMemo(
    () => parseBrowseUrlState(searchParams),
    [searchParams],
  );
  const librariesQuery = useLibrariesQuery();
  const libraryQuery = useLibraryQuery(libraryId);
  const directoryQuery = useDirectoryQuery(directoryId);
  const childrenQuery = useDirectoriesQuery({ libraryId, parentId: directoryId });
  const assetsQuery = useAssetsQuery({
    directoryId,
    libraryId,
    order: browseState.order,
    recursive: browseState.recursive,
    sort: browseState.sort,
  });
  const { refetch: refreshLibrary } = libraryQuery;
  const { refetch: refreshDirectory } = directoryQuery;
  const {
    fetchNextPage: loadMoreChildren,
    refetch: refreshChildren,
  } = childrenQuery;
  const {
    fetchNextPage: loadMoreAssets,
    refetch: refreshAssets,
  } = assetsQuery;
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
    return assets.map((asset) => ({
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
      thumbnailStatus: asset.thumbnail.status,
      thumbnailUrl: asset.thumbnail.url,
      width: asset.width,
    }));
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
    directoryQuery.data?.name ?? currentLibrary?.name ?? t("browse.libraryFallback");
  const currentMediaCount =
    directoryQuery.data?.directAssetCount ?? currentLibrary?.assetCount ?? 0;
  const browseHref = browseUrl(libraryId, directoryId, browseState);
  const canonicalSearch = serializeBrowseUrlState(browseState);

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
  }, [assets, location.pathname, location.search, location.state, navigate, preview.collectionRef]);

  function updateBrowseState(nextState: BrowseUrlState) {
    setSearchParams(serializeBrowseUrlState(nextState));
  }

  function updateMediaLayout(nextLayout: MediaCollectionLayout) {
    setMediaLayout(nextLayout);
    writeMediaLayoutPreference(nextLayout);
  }

  async function copyDirectLink() {
    try {
      await navigator.clipboard.writeText(window.location.href);
      toast.show({ message: t("browse.linkCopied"), tone: "success" });
    } catch {
      toast.show({ message: t("browse.linkCopyFailed"), tone: "danger" });
    }
  }

  return (
    <AppShell
      active="browse"
      browseHref={browseHref}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      searchHref={
        directoryId
          ? `${paths.librarySearch(libraryId)}?${new URLSearchParams({
              directoryId,
              scope: "directory",
            }).toString()}`
          : paths.librarySearch(libraryId)
      }
      settingsHref={paths.generalSettings}
      sidebarContent={
        <DirectoryNavigation
          currentLibraryName={currentLibrary?.name}
          libraries={libraries}
          libraryId={libraryId}
          browseState={browseState}
          onLibraryChange={(nextLibraryId) =>
            navigate(
              browseUrl(
                nextLibraryId,
                undefined,
                defaultBrowseUrlState(),
              ),
            )
          }
          selectedDirectoryId={directoryId}
          selectedPathIds={selectedPathIds}
        />
      }
      title={t("browse.title")}
    >
      <section className={styles.page} aria-labelledby="browse-heading">
        {currentLibrary?.status === "offline" && (
          <div className={styles.banner}>
            <InlineStatus tone="warning">
              {t("browse.offlinePreserved")}
            </InlineStatus>
          </div>
        )}
        <header className={styles.header}>
          <Breadcrumbs
            browseState={browseState}
            breadcrumbs={breadcrumbs}
            libraryId={libraryId}
            rootName={currentLibrary?.name ?? currentName}
          />
          <IconButton label={t("browse.copyLink")} onClick={() => void copyDirectLink()}>
            <Copy aria-hidden="true" size={19} />
          </IconButton>
        </header>

        <BrowseToolbar
          browseState={browseState}
          mediaLayout={mediaLayout}
          onChange={updateBrowseState}
          onLayoutChange={updateMediaLayout}
        />

        <div
          className={styles.workspace}
          data-has-preview={previewItem ? "" : undefined}
          style={{ "--preview-width": `${preview.width}px` } as CSSProperties}
        >
        <div className={styles.content}>
          {(libraryQuery.isPending || (directoryId && directoryQuery.isPending)) && (
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
                  children.length > 0) && (
                  <section aria-labelledby="child-directories-heading">
                    <div className={styles.sectionHeading}>
                      <div>
                        <h2 id="child-directories-heading">
                          {t("browse.childDirectories")}
                        </h2>
                        <p>
                          {t("browse.directorySummary")
                            .replace(
                              "{directories}",
                              String(children.length),
                            )
                            .replace(
                              "{media}",
                              String(currentMediaCount),
                            )}
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
                          to={browseUrl(libraryId, directory.id, browseState)}
                        >
                          <Folder aria-hidden="true" size={30} weight="fill" />
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
                    <MediaCollectionSkeleton label={t("browse.loadingMedia")} />
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
                      onEnableRecursive={() =>
                        updateBrowseState(defaultBrowseUrlState(true))
                      }
                    />
                  )}
                  {assets.length > 0 && (
                    <MediaCollection
                      ref={preview.collectionRef}
                      hasNextPage={assetsQuery.hasNextPage}
                      isFetchingNextPage={assetsQuery.isFetchingNextPage}
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
                        unavailableThumbnail: t("browse.thumbnailUnavailable"),
                        video: t("browse.kindVideo"),
                      }}
                      layout={mediaLayout}
                      onItemActivate={(assetId, activation) =>
                        preview.activate(assetId, activation)
                      }
                      onLoadMore={loadMoreAssets}
                      onRetryLoadMore={() => void loadMoreAssets()}
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

function BrowseToolbar({
  browseState,
  mediaLayout,
  onChange,
  onLayoutChange,
}: {
  browseState: BrowseUrlState;
  mediaLayout: MediaCollectionLayout;
  onChange: (state: BrowseUrlState) => void;
  onLayoutChange: (layout: MediaCollectionLayout) => void;
}) {
  const { t } = useLocale();
  const sortValue = `${browseState.sort}:${browseState.order}`;
  const defaults = defaultBrowseUrlState(browseState.recursive);
  const customSort =
    browseState.sort !== defaults.sort || browseState.order !== defaults.order;

  return (
    <div className={styles.toolbar} role="region" aria-label={t("browse.tools")}>
      <Button
        aria-pressed={browseState.recursive}
        className={styles.recursiveToggle}
        onClick={() =>
          onChange(defaultBrowseUrlState(!browseState.recursive))
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
      <div className={styles.layoutControls} role="group" aria-label={t("browse.layout")}>
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
          <option value="modifiedAt:desc">{t("browse.sortModifiedDescending")}</option>
          <option value="modifiedAt:asc">{t("browse.sortModifiedAscending")}</option>
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
    </div>
  );
}

function EmptyMedia({
  browseState,
  directory,
  onEnableRecursive,
}: {
  browseState: BrowseUrlState;
  directory?: Directory | undefined;
  onEnableRecursive: () => void;
}) {
  const { t } = useLocale();
  const hasDescendantMedia =
    !browseState.recursive &&
    directory !== undefined &&
    directory.recursiveAssetCount > directory.directAssetCount;

  return (
    <EmptyState
      action={
        hasDescendantMedia ? (
          <Button onClick={onEnableRecursive}>
            {t("browse.enableRecursive")}
          </Button>
        ) : undefined
      }
      description={
        hasDescendantMedia
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
        browseState.recursive
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
    breadcrumbs.length > 0
      ? breadcrumbs
      : [{ id: "root", name: rootName, relativePath: "" }];

  return (
    <nav aria-label={t("browse.breadcrumbs")} className={styles.breadcrumbs}>
      <House aria-hidden="true" size={17} />
      {items.map((item, index) => {
        const current = index === items.length - 1;
        return (
          <span className={styles.crumb} key={item.id}>
            {index > 0 && <CaretRight aria-hidden="true" size={14} />}
            {current ? (
              <span aria-current="page">{item.name}</span>
            ) : (
              <Link
                to={browseUrl(
                  libraryId,
                  index === 0 ? undefined : item.id,
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
  libraries: { id: string; name: string; status: string }[];
  libraryId: string;
  onLibraryChange: (libraryId: string) => void;
  selectedDirectoryId?: string | undefined;
  selectedPathIds: Set<string>;
}) {
  const { t } = useLocale();
  return (
    <div className={styles.directoryNavigation}>
      <label className={styles.libraryPicker}>
        <span>{t("browse.library")}</span>
        <select
          aria-label={t("browse.library")}
          onChange={(event) => onLibraryChange(event.target.value)}
          value={libraryId}
        >
          {!libraries.some((library) => library.id === libraryId) && (
            <option value={libraryId}>
              {currentLibraryName ?? t("browse.libraryFallback")}
            </option>
          )}
          {libraries.map((library) => (
            <option key={library.id} value={library.id}>
              {library.name}
            </option>
          ))}
        </select>
      </label>
      <p className={styles.treeLabel}>{t("browse.directory")}</p>
      <nav aria-label={t("browse.directoryNavigation")} className={styles.tree}>
        <Link
          aria-current={selectedDirectoryId ? undefined : "page"}
          className={`${styles.treeLink} ${!selectedDirectoryId ? styles.treeLinkCurrent : ""}`}
          to={browseUrl(libraryId, undefined, browseState)}
        >
          <ImageSquare aria-hidden="true" size={18} />
          <span>{t("browse.allMedia")}</span>
        </Link>
        <DirectoryChildren
          browseState={browseState}
          depth={0}
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
  const {
    fetchNextPage: loadMoreDirectories,
    refetch: refreshDirectories,
  } = query;
  const directories = useMemo(
    () => query.data?.pages.flatMap((page) => page.items) ?? [],
    [query.data],
  );

  if (query.isPending) {
    return <span className={styles.treeState}>{t("browse.loadingDirectories")}</span>;
  }
  if (query.isError) {
    return (
      <Button size="small" variant="quiet" onClick={() => void refreshDirectories()}>
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
      <div className={styles.treeRow} style={{ "--tree-depth": depth } as CSSProperties}>
        {directory.hasChildren ? (
          <IconButton
            className={styles.treeToggle}
            label={
              expanded
                ? t("browse.collapseDirectory").replace("{name}", directory.name)
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
