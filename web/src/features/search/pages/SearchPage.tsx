import { AssetExifPanel } from "../../media";
import {
  ArrowClockwise,
  MagnifyingGlass,
} from "@phosphor-icons/react";
import { useQueryClient } from "@tanstack/react-query";
import {
  useEffect,
  useMemo,
  useState,
  type CSSProperties,
} from "react";
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
import {
  Button,
  Badge,
  EmptyState,
  ErrorState,
  InlineStatus,
  OfflineState,
  Select,
  RadioOption,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import {
  assetContentUrl,
  type Asset,
  type SemanticSearchCoverage,
  type SemanticVideoMatch,
  type SearchMatch,
} from "../../../lib/api/catalog";
import { ApiError } from "../../../lib/api/errors";
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
  readMediaSortPreference,
  type MediaLayoutPreference,
} from "../../../lib/storage/preferences";
import { paths } from "../../../routes/paths";
import {
  createViewerLocationState,
  readViewerReturnState,
} from "../../../lib/navigation/viewer";
import { defaultBrowseUrlState, DirectoryNavigation } from "../../browse";
import { useLibrariesQuery, useLibraryQuery } from "../../libraries";
import { AssetCurationControls, useFavoriteMutation } from "../../curation";
import {
  refreshSearchResults,
  useSearchResultsQuery,
} from "../queries";
import {
  defaultSearchUrlState,
  parseSearchUrlState,
  serializeSearchUrlState,
  type SearchDate,
  type SearchKind,
  type SearchUrlState,
} from "../urlState";
import styles from "./SearchPage.module.css";

export function SearchPage({
  libraryId,
  logoutPending,
  onLogout,
  session,
}: {
  libraryId?: string;
  logoutPending?: boolean;
  onLogout?: () => Promise<void>;
  session: AuthenticatedSession;
}) {
  const { locale, t } = useLocale();
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const [mediaSortPreference] = useState(readMediaSortPreference);
  const state = useMemo(
    () => parseSearchUrlState(searchParams, libraryId, mediaSortPreference),
    [libraryId, mediaSortPreference, searchParams],
  );
  const [refreshPending, setRefreshPending] = useState(false);
  const favoriteMutation = useFavoriteMutation();
  const layout = useState<MediaLayoutPreference>(
    readMediaLayoutPreference,
  )[0];
  const librariesQuery = useLibrariesQuery();
  const libraryQuery = useLibraryQuery(libraryId ?? "");
  const resultsQuery = useSearchResultsQuery({
    state,
    ...(libraryId ? { libraryId } : {}),
  });
  const { refetch: retryResults } = resultsQuery;
  const assets = useMemo(
    () => resultsQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [resultsQuery.data],
  );
  const preview = useMediaPreviewController({
    items: assets,
    resetKey: libraryId ?? "all-libraries",
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
            mimeType: previewAsset.mimeType,
            ...(mediaPosterUrl(previewAsset)
              ? { posterUrl: mediaPosterUrl(previewAsset) }
              : {}),
          }
        : undefined,
    [locale, previewAsset, t],
  );
  const semanticCoverage = semanticCoverageFromPages(resultsQuery.data?.pages);
  const semanticVideoMatches = semanticVideoMatchesFromPages(resultsQuery.data?.pages);
  const searchMatches = useMemo<Record<string, SearchMatch>>(() => {
    const matches = Object.assign({}, ...(resultsQuery.data?.pages.map(page => page.searchMatches ?? {}) ?? [])) as Record<string, SearchMatch>;
    for (const asset of assets) {
      if (!matches[asset.id]) matches[asset.id] = { sources: asset.searchSources ?? (state.mode === "semantic" ? ["semantic"] : []), semanticScore: null };
    }
    return matches;
  }, [assets, resultsQuery.data, state.mode]);
  const ocrPage = resultsQuery.data?.pages.findLast(page => page.searchOCRState);
  const semanticState = resultsQuery.data?.pages.findLast(page => page.searchSemanticState)?.searchSemanticState;
  const mediaItems = useMemo(
    () => mapSearchItems(
      assets,
      locale,
      t("search.source"),
      semanticVideoMatches,
      t("search.videoMatch"),
      (state.mode === "all" || state.mode === "ocr") ? "" : t(state.mode === "semantic" ? "search.matchSemantic" : "search.matchLiteral"),
      searchMatches,
      source => sourceLabel(source, t),
    ),
    [assets, locale, semanticVideoMatches, searchMatches, state.mode, t],
  );
  const canonicalSearch = serializeSearchUrlState(state);

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

  function updateState(next: SearchUrlState) {
    setSearchParams(serializeSearchUrlState(next));
  }

  function clearFilters() {
    const defaults = defaultSearchUrlState(
      libraryId,
      undefined,
      mediaSortPreference,
    );
    updateState({
      ...state,
      date: "any",
      kind: "all",
      ...(state.mode === "semantic" ? { kind: "image" as const } : {}),
      order: defaults.order,
      recursive: false,
      sort: defaults.sort,
    });
  }

  const currentLibrary = libraryQuery.data?.library;

  return (
    <AppShell
      autoHideChrome={layout === "seamless" && mediaItems.length > 0}
      active="search"
      browseHref={libraryId ? paths.browse(libraryId) : paths.root}
      homeHref={paths.root}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      logoutPending={logoutPending}
      onLogout={onLogout}
      searchHref={`${libraryId ? paths.librarySearch(libraryId) : paths.search}?${canonicalSearch}`}
      searchInput={{
        label: t(state.mode === "ocr" ? "search.modeOCR" : state.mode === "all" ? "search.fusedInputLabel" : state.mode === "semantic" ? "search.semanticInputLabel" : "search.inputLabel"),
        placeholder: t(state.mode === "ocr" ? "search.ocrPlaceholder" : state.mode === "all" ? "search.fusedPlaceholder" : state.mode === "semantic" ? "search.semanticPlaceholder" : "search.placeholder"),
      }}
      settingsHref={
        libraryId
          ? paths.generalSettingsForLibrary(libraryId)
          : paths.generalSettings
      }
      navigationKey={`search:${libraryId ?? "all"}`}
      navigationLabel={t("context.openSearch")}
      sidebarLabel={t("context.search")}
      sidebarContent={<DirectoryNavigation
        activeQuickAccess="search"
        browseState={defaultBrowseUrlState()}
        currentLibraryName={currentLibrary?.name ?? t("search.allLibraries")}
        hideDirectoryTree
        libraries={(() => {
          const listed = librariesQuery.data?.pages.flatMap(page => page.items) ?? [];
          return currentLibrary && !listed.some(item => item.id === currentLibrary.id) ? [...listed, currentLibrary] : listed;
        })()}
        libraryId={libraryId ?? ""}
        onLibraryChange={id => {
          const next = { ...state, scope: "library" as const, recursive: false };
          delete next.directoryId;
          navigate(`${paths.librarySearch(id)}?${serializeSearchUrlState(next)}`);
        }}
        quickAccessDetail={<ContextSidebar title={t("context.search")}>
          <SearchFilters canUseDirectory={Boolean(state.directoryId)} hasLibrary={Boolean(libraryId)} onChange={updateState} state={state} />
          <Button size="small" variant="quiet" onClick={clearFilters}>{t("search.clearFilters")}</Button>
        </ContextSidebar>}
        selectedPathIds={new Set<string>()}
      />}
      topbarContent={<span>{t(state.scope === "all" ? "search.scopeAll" : state.scope === "directory" ? "search.scopeDirectory" : "search.scopeLibrary")}{state.scope !== "all" && currentLibrary ? ` · ${currentLibrary.name}` : ""}</span>}
      title={t("search.title")}
    >
      <section className={styles.page} aria-labelledby="search-heading">
        <div
          className={styles.workspace}
          data-has-preview={previewItem ? "" : undefined}
          style={{ "--preview-width": `${preview.width}px` } as CSSProperties}
        >
        <div className={styles.results}>
        <header className={styles.header}>
          <div className={styles.titleRow}>
            <h1 id="search-heading">{state.q ? t("search.resultsFor").replace("{query}", state.q) : t("search.heading")}</h1>
            {assets.length > 0 && <span className={styles.resultCount} aria-live="polite">{t("search.loadedCount").replace("{count}", String(assets.length))}</span>}
          </div>
        <div className={styles.command}>
          <p className={styles.applied} aria-label={t("context.resultsFilters")}>
            {t(state.mode === "ocr" ? "search.modeOCR" : state.mode === "all" ? "search.modeAll" : state.mode === "semantic" ? "search.modeSemantic" : "search.modeFilename")} · {t(state.kind === "all" ? "search.kindAll" : state.kind === "video" ? "browse.kindVideo" : state.kind === "animated" ? "browse.kindAnimated" : "browse.kindImage")}
            {state.mode === "semantic" ? ` · ${t("context.similarityOrder")}` : ` · ${t(state.date === "30d" ? "search.date30d" : state.date === "year" ? "search.dateYear" : "search.dateAny")}`}
          </p>
          <Button
            className={styles.refresh}
            disabled={!state.q}
            loading={refreshPending}
            onClick={async () => {
              setRefreshPending(true);
              try {
                await refreshSearchResults(queryClient, libraryId, state);
              } finally {
                setRefreshPending(false);
              }
            }}
            variant="quiet"
          >
            <ArrowClockwise aria-hidden="true" size={17} />
            {t("search.refresh")}
          </Button>
        </div>
        </header>

          {!state.q && (
            <EmptyState
              description={t(state.mode === "ocr" ? "search.ocrDescription" : state.mode === "all" ? "search.fusedStart" : state.mode === "semantic" ? "search.semanticStartDescription" : "search.startDescription")}
              icon={MagnifyingGlass}
              title={t("search.startTitle")}
            />
          )}
          {state.q && resultsQuery.isPending && (
            <MediaCollectionSkeleton label={t("search.loading")} />
          )}
          {state.q && resultsQuery.isError && assets.length === 0 && (
            <ErrorState
              message={semanticErrorMessage(resultsQuery.error, state.mode, t)}
              onRetry={() => void retryResults()}
            />
          )}
          {state.q && (ocrPage?.searchOCRState === "disabled" || (ocrPage?.searchOCRCoverage && ocrPage.searchOCRCoverage.ready < ocrPage.searchOCRCoverage.eligible) || (semanticCoverage && !semanticCoverage.complete) || (semanticState && !["ready", "not_requested"].includes(semanticState))) && <div className={styles.resultMeta}>
          {state.q && ocrPage?.searchOCRState === "disabled" && <InlineStatus tone="warning">{t("search.ocrDisabled")}</InlineStatus>}
          {((ocrPage?.searchOCRCoverage && ocrPage.searchOCRCoverage.ready < ocrPage.searchOCRCoverage.eligible) || (semanticCoverage && !semanticCoverage.complete)) && <InlineStatus appearance="plain" tone="warning">{t("search.incompleteResults")}</InlineStatus>}
          {state.q && semanticState && !["ready", "not_requested"].includes(semanticState) && <InlineStatus tone="warning">{t("search.semanticDegraded")}</InlineStatus>}
          {state.q && state.mode === "all" && semanticCoverage && semanticCoverage.excludedLibraries.length > 0 && <InlineStatus tone="warning">{t("search.semanticExcluded")}</InlineStatus>}
          </div>}
          {state.q &&
            resultsQuery.isSuccess &&
            assets.length === 0 &&
            currentLibrary?.status === "offline" && (
              <OfflineState
                action={
                  <Button onClick={() => void retryResults()}>
                    {t("common.retry")}
                  </Button>
                }
                description={t("search.offlineDescription")}
                title={t("search.offlineTitle")}
              />
            )}
          {state.q &&
            resultsQuery.isSuccess &&
            assets.length === 0 &&
            currentLibrary?.status !== "offline" && (
              <EmptyState
                action={
                    <Button
                      onClick={() =>
                        document.querySelector<HTMLInputElement>(
                          "#global-search",
                        )?.focus()
                      }
                    >
                      {t("search.editQuery")}
                    </Button>
                }
                description={t(state.mode === "semantic" && semanticCoverage && !semanticCoverage.complete
                  ? "search.semanticEmptyIncomplete"
                  : "search.emptyDescription").replace("{query}", state.q)}
                icon={MagnifyingGlass}
                title={t("search.emptyTitle")}
              />
            )}
          {assets.length > 0 && (
            <>
              {resultsQuery.isFetchNextPageError && <InlineStatus tone="danger">{t("search.loadMoreFailed")}</InlineStatus>}
              <div className={styles.media}>
              <MediaCollection
                ref={preview.collectionRef}
                hasNextPage={resultsQuery.hasNextPage}
                isFetchingNextPage={resultsQuery.isFetching}
                items={mediaItems}
                labels={{
                  activatePreview: preview.pinned
                    ? t("browse.selectPinnedPreview")
                    : t("browse.activatePreview"),
                  addFavorite: t("curation.addFavorite"),
                  animated: t("browse.kindAnimated"),
                  failedThumbnail: t("browse.thumbnailFailed"),
                  image: t("browse.kindImage"),
                  loadMore: t("search.loadMore"),
                  loadMoreFailed: t("search.loadMoreFailed"),
                  loadingMore: t("search.loadingMore"),
                  pendingThumbnail: t("browse.thumbnailPending"),
                  tileSize: t("browse.tileSize"),
                  previewing: t("browse.currentlyPreviewing"),
                  removeFavorite: t("curation.removeFavorite"),
                  retryLoadMore: t("search.retryLoadMore"),
                  unavailableThumbnail: t("browse.thumbnailUnavailable"),
                  video: t("browse.kindVideo"),
                }}
                layout={layout}
                {...(favoriteMutation.variables?.assetId
                  ? { favoritePendingId: favoriteMutation.variables.assetId }
                  : {})}
                onFavoriteToggle={(assetId, favorite) =>
                  favoriteMutation.mutate({
                    assetId,
                    csrfToken: session.csrfToken,
                    favorite,
                  })
                }
                onItemActivate={(assetId, activation) =>
                  preview.activate(assetId, activation)
                }
                onLoadMore={() => void resultsQuery.fetchNextPage()}
                onRetryLoadMore={() =>
                  void retryInfiniteNextPage({
                    error: resultsQuery.error,
                    loadNextPage: resultsQuery.fetchNextPage,
                    refresh: resultsQuery.refetch,
                  })
                }
                paginationError={resultsQuery.isFetchNextPageError}
                {...(previewAsset
                  ? { previewItemId: previewAsset.id }
                  : {})}
                {...(preview.selectedItemId
                  ? { selectedItemId: preview.selectedItemId }
                  : {})}
              />
              </div>
            </>
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
                    () => void retryResults(),
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
                <>
                {searchMatches[previewAsset.id]?.sources.length ? <section className={styles.evidence} aria-label={t("search.whyMatched")}>
                  <h3>{t("search.whyMatched")}</h3>
                  {searchMatches[previewAsset.id]!.sources.map(source => <div key={source}>
                    <Badge tone={source === "semantic" ? "accent" : "neutral"}>{sourceLabel(source, t)}</Badge>
                    {source === "ocr" ? <>
                      {searchMatches[previewAsset.id]?.ocr?.map(excerpt => <p key={excerpt.line}>{excerpt.text}</p>)}
                    </> : source !== "semantic" ? <p>{source === "filename" ? previewAsset.name : previewAsset.relativePath}</p> : null}
                  </div>)}
                </section> : null}
                <AssetCurationControls
                  assetId={previewAsset.id}
                  csrfToken={session.csrfToken}
                />
                </>
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
              position:
                preview.previewIndex >= 0
                  ? t("browse.previewPosition")
                      .replace(
                        "{current}",
                        String(preview.previewIndex + 1),
                      )
                      .replace("{total}", String(assets.length))
                  : t("search.previewOutsideResults"),
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

function SearchFilters({
  canUseDirectory,
  hasLibrary,
  onChange,
  state,
}: {
  canUseDirectory: boolean;
  hasLibrary: boolean;
  onChange: (state: SearchUrlState) => void;
  state: SearchUrlState;
}) {
  const { t } = useLocale();

  return (
    <fieldset className={styles.filters}>
      <legend className={styles.visuallyHidden}>
        {t("search.filters")}
      </legend>

      <fieldset>
        <legend>{t("search.scope")}</legend>
        {hasLibrary && <RadioOption name="search-scope" value="library" checked={state.scope === "library"} onChange={() => onChange({ ...state, recursive: false, scope: "library" })}>{t("search.scopeLibrary")}</RadioOption>}
        {hasLibrary && <RadioOption name="search-scope" value="directory" checked={state.scope === "directory"} disabled={!canUseDirectory} onChange={() => onChange({ ...state, recursive: false, scope: "directory" })}>{t("search.scopeDirectory")}</RadioOption>}
        <RadioOption name="search-scope" value="all" checked={state.scope === "all"} onChange={() => onChange({ ...state, recursive: false, scope: "all" })}>{t("search.scopeAll")}</RadioOption>
      </fieldset>
      {state.scope === "directory" && (
        <label className={styles.checkLabel}>
          <input
            checked={state.recursive}
            onChange={(event) =>
              onChange({ ...state, recursive: event.target.checked })
            }
            type="checkbox"
          />
          <span>{t("search.includeChildren")}</span>
        </label>
      )}
      <fieldset className={styles.modeControl}>
        <legend>{t("search.mode")}</legend>
        <RadioOption name="search-mode" value="all" checked={state.mode === "all"} onChange={() => onChange({ ...state, mode: "all" })}>{t("search.modeAll")}</RadioOption>
        <RadioOption name="search-mode" value="filename" checked={state.mode === "filename"} onChange={() => onChange({ ...state, mode: "filename" })}>{t("search.modeFilename")}</RadioOption>
        <RadioOption name="search-mode" value="semantic" checked={state.mode === "semantic"} onChange={() => onChange({ ...state, date: "any", kind: "image", mode: "semantic", order: "desc", sort: "modifiedAt" })}>{t("search.modeSemantic")}</RadioOption>
        <RadioOption name="search-mode" value="ocr" checked={state.mode === "ocr"} onChange={() => onChange({ ...state, mode: "ocr", order: "desc", sort: "modifiedAt" })}>{t("search.modeOCR")}</RadioOption>
      </fieldset>
      <label>
        <span>{t("search.kind")}</span>
        <Select
          onChange={(event) =>
            onChange({ ...state, kind: event.target.value as SearchKind })
          }
          value={state.kind}
        >
          {state.mode !== "semantic" && <option value="all">{t("search.kindAll")}</option>}
          <option value="image">{t("browse.kindImage")}</option>
          {state.mode !== "semantic" && <option value="animated">{t("browse.kindAnimated")}</option>}
          <option value="video">{t("browse.kindVideo")}</option>
        </Select>
      </label>
      {state.mode !== "semantic" && <label>
        <span>{t("search.date")}</span>
        <Select
          onChange={(event) =>
            onChange({ ...state, date: event.target.value as SearchDate })
          }
          value={state.date}
        >
          <option value="any">{t("search.dateAny")}</option>
          <option value="30d">{t("search.date30d")}</option>
          <option value="year">{t("search.dateYear")}</option>
        </Select>
      </label>}
      {state.mode === "filename" && <label>
        <span>{t("search.sort")}</span>
        <Select
          onChange={(event) => {
            const [sort, order] = event.target.value.split(":") as [
              SearchUrlState["sort"],
              SearchUrlState["order"],
            ];
            onChange({ ...state, order, sort });
          }}
          value={`${state.sort}:${state.order}`}
        >
          <option value="modifiedAt:desc">
            {t("browse.sortModifiedDescending")}
          </option>
          <option value="modifiedAt:asc">
            {t("browse.sortModifiedAscending")}
          </option>
          <option value="name:asc">{t("browse.sortNameAscending")}</option>
          <option value="name:desc">{t("browse.sortNameDescending")}</option>
          <option value="size:desc">{t("browse.sortSizeDescending")}</option>
          <option value="size:asc">{t("browse.sortSizeAscending")}</option>
        </Select>
      </label>}
    </fieldset>
  );
}

function semanticCoverageFromPages(
  pages: Array<{ semanticCoverage?: SemanticSearchCoverage }> | undefined,
): SemanticSearchCoverage | undefined {
  return pages?.findLast((page) => page.semanticCoverage)?.semanticCoverage;
}

function semanticVideoMatchesFromPages(
  pages: Array<{ semanticVideoMatches?: Record<string, SemanticVideoMatch> }> | undefined,
): Record<string, SemanticVideoMatch> {
  return Object.assign(
    {},
    ...(pages?.map((page) => page.semanticVideoMatches ?? {}) ?? []),
  );
}

function semanticErrorMessage(
  error: unknown,
  mode: SearchUrlState["mode"],
  t: ReturnType<typeof useLocale>["t"],
): string {
  if (error instanceof ApiError && error.code === "search_cursor_stale") return t("search.fusedStale");
  if (mode !== "semantic" || !(error instanceof ApiError)) return t("search.failed");
  if (error.code === "ai_disabled") return t("search.semanticDisabled");
  if (error.code === "model_unavailable") return t("search.semanticModelUnavailable");
  if (error.code === "semantic_not_ready") return t("search.semanticNotReady");
  if (error.code === "semantic_busy") return t("search.semanticBusy");
  if (error.code === "semantic_cursor_stale") return t("search.semanticStale");
  return t("search.semanticFailed");
}

function mapSearchItems(
  assets: Asset[],
  locale: string,
  sourceTemplate: string,
  videoMatches: Record<string, SemanticVideoMatch>,
  videoMatchTemplate: string,
  retrievalLabel: string,
  matches: Record<string, SearchMatch>,
  label: (source: SearchMatch["sources"][number]) => string,
): MediaCollectionItem[] {
  const formatter = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  });
  return assets.map((asset) => {
    const storyboard = mediaStoryboard(asset);
    const videoMatch = videoMatches[asset.id];
    return {
      ...(matches[asset.id]?.sources.length ? { badges: matches[asset.id]!.sources.map(source => ({ label: label(source), tone: source === "semantic" ? "accent" as const : "neutral" as const })) } : {}),
      ...(retrievalLabel ? { contextLabel: videoMatch
        ? `${retrievalLabel} · ${videoMatchTemplate
            .replace("{time}", formatDuration(videoMatch.timestampMs))
            .replace("{plan}", String(videoMatch.planSize))}`
        : retrievalLabel } : {}),
      favorite: asset.favorite ?? false,
      height: asset.height,
      id: asset.id,
      kind: asset.kind,
      modifiedLabel: formatter.format(new Date(asset.modifiedAt)),
      name: asset.name,
      sourceHref: paths.browse(asset.libraryId, asset.directoryId),
      sourceLabel: sourceTemplate
        .replace("{library}", asset.libraryName)
        .replace("{path}", sourceDirectory(asset.relativePath)),
      ...(storyboard ? { storyboard } : {}),
      thumbnailStatus: asset.thumbnail.status,
      thumbnailUrl: asset.thumbnail.url,
      width: asset.width,
    };
  });
}

function formatDuration(milliseconds: number): string {
  const totalSeconds = Math.floor(milliseconds / 1000);
  return `${Math.floor(totalSeconds / 60)}:${String(totalSeconds % 60).padStart(2, "0")}`;
}

function sourceLabel(source: SearchMatch["sources"][number], t: ReturnType<typeof useLocale>["t"]): string {
  return t(source === "filename" ? "search.sourceFilename" : source === "path" ? "search.sourcePath" : source === "ocr" ? "search.modeOCR" : "search.sourceSemantic");
}

function sourceDirectory(relativePath: string): string {
  const separator = relativePath.lastIndexOf("/");
  return separator < 0 ? "/" : relativePath.slice(0, separator) || "/";
}
