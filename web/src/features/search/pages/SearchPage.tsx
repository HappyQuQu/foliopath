import { FunnelSimple, MagnifyingGlass } from "@phosphor-icons/react";
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type FormEvent,
} from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

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
  EmptyState,
  ErrorState,
  InlineStatus,
  OfflineState,
  SearchInput,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { assetContentUrl, type Asset } from "../../../lib/api/catalog";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import {
  readMediaLayoutPreference,
  type MediaLayoutPreference,
} from "../../../lib/storage/preferences";
import { paths } from "../../../routes/paths";
import {
  createViewerLocationState,
  readViewerReturnState,
} from "../../../lib/navigation/viewer";
import { useLibraryQuery } from "../../libraries";
import { useSearchResultsQuery } from "../queries";
import {
  parseSearchUrlState,
  serializeSearchUrlState,
  type SearchDate,
  type SearchKind,
  type SearchScope,
  type SearchUrlState,
} from "../urlState";
import styles from "./SearchPage.module.css";

export function SearchPage({
  libraryId,
  session,
}: {
  libraryId?: string;
  session: AuthenticatedSession;
}) {
  const { locale, t } = useLocale();
  const location = useLocation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const state = useMemo(
    () => parseSearchUrlState(searchParams, libraryId),
    [libraryId, searchParams],
  );
  const [draft, setDraft] = useState(state.q);
  const inputRef = useRef<HTMLInputElement>(null);
  const layout = useState<MediaLayoutPreference>(
    readMediaLayoutPreference,
  )[0];
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
          }
        : undefined,
    [locale, previewAsset, t],
  );
  const mediaItems = useMemo(
    () => mapSearchItems(assets, locale, t("search.source")),
    [assets, locale, t],
  );
  const canonicalSearch = serializeSearchUrlState(state);

  useEffect(() => {
    if (searchParams.toString() !== canonicalSearch) {
      setSearchParams(canonicalSearch, { replace: true });
    }
  }, [canonicalSearch, searchParams, setSearchParams]);

  useEffect(() => setDraft(state.q), [state.q]);

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

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const q = draft.trim();
    if (!q) {
      inputRef.current?.focus();
      return;
    }
    updateState({ ...state, q });
  }

  function clearFilters() {
    updateState({
      ...state,
      date: "any",
      kind: "all",
      order: "desc",
      recursive: false,
      sort: "modifiedAt",
    });
  }

  const currentLibrary = libraryQuery.data?.library;
  const browseHref = libraryId ? paths.browse(libraryId) : undefined;
  const hasFilters =
    state.kind !== "all" ||
    state.date !== "any" ||
    state.sort !== "modifiedAt" ||
    state.order !== "desc" ||
    state.recursive;

  return (
    <AppShell
      active="search"
      {...(browseHref ? { browseHref } : {})}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      searchHref={`${libraryId ? paths.librarySearch(libraryId) : paths.search}?${canonicalSearch}`}
      settingsHref={paths.generalSettings}
      title={t("search.title")}
    >
      <section className={styles.page} aria-labelledby="search-heading">
        <header className={styles.header}>
          <p className={styles.eyebrow}>
            {currentLibrary?.name ?? t("search.allLibraries")}
          </p>
          <h1 id="search-heading">{t("search.heading")}</h1>
        </header>

        <div className={styles.command}>
          <SearchInput
            ref={inputRef}
            label={t("search.inputLabel")}
            onChange={(event) => setDraft(event.target.value)}
            onSubmit={submitSearch}
            placeholder={t("search.placeholder")}
            submitLabel={t("search.submit")}
            value={draft}
          />
          <SearchFilters
            canUseDirectory={Boolean(state.directoryId)}
            hasLibrary={Boolean(libraryId)}
            onChange={updateState}
            state={state}
          />
        </div>

        <div
          className={styles.workspace}
          data-has-preview={previewItem ? "" : undefined}
          style={{ "--preview-width": `${preview.width}px` } as CSSProperties}
        >
        <div className={styles.results}>
          {!state.q && (
            <EmptyState
              description={t("search.startDescription")}
              icon={MagnifyingGlass}
              title={t("search.startTitle")}
            />
          )}
          {state.q && resultsQuery.isPending && (
            <MediaCollectionSkeleton label={t("search.loading")} />
          )}
          {state.q && resultsQuery.isError && assets.length === 0 && (
            <ErrorState
              message={t("search.failed")}
              onRetry={() => void retryResults()}
            />
          )}
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
                  hasFilters ? (
                    <Button onClick={clearFilters}>
                      {t("search.clearFilters")}
                    </Button>
                  ) : (
                    <Button onClick={() => inputRef.current?.focus()}>
                      {t("search.editQuery")}
                    </Button>
                  )
                }
                description={t("search.emptyDescription").replace(
                  "{query}",
                  state.q,
                )}
                icon={MagnifyingGlass}
                title={t("search.emptyTitle")}
              />
            )}
          {assets.length > 0 && (
            <>
              <div className={styles.summary} aria-live="polite">
                <div>
                  <strong>
                    {t("search.resultsFor").replace("{query}", state.q)}
                  </strong>
                  <span>
                    {t("search.loadedCount").replace(
                      "{count}",
                      String(assets.length),
                    )}
                  </span>
                </div>
                {resultsQuery.isFetchNextPageError && (
                  <InlineStatus tone="danger">
                    {t("search.loadMoreFailed")}
                  </InlineStatus>
                )}
              </div>
              <MediaCollection
                ref={preview.collectionRef}
                hasNextPage={resultsQuery.hasNextPage}
                isFetchingNextPage={resultsQuery.isFetchingNextPage}
                items={mediaItems}
                labels={{
                  activatePreview: preview.pinned
                    ? t("browse.selectPinnedPreview")
                    : t("browse.activatePreview"),
                  animated: t("browse.kindAnimated"),
                  failedThumbnail: t("browse.thumbnailFailed"),
                  image: t("browse.kindImage"),
                  loadMore: t("search.loadMore"),
                  loadMoreFailed: t("search.loadMoreFailed"),
                  loadingMore: t("search.loadingMore"),
                  pendingThumbnail: t("browse.thumbnailPending"),
                  previewing: t("browse.currentlyPreviewing"),
                  retryLoadMore: t("search.retryLoadMore"),
                  unavailableThumbnail: t("browse.thumbnailUnavailable"),
                  video: t("browse.kindVideo"),
                }}
                layout={layout}
                onItemActivate={(assetId, activation) =>
                  preview.activate(assetId, activation)
                }
                onLoadMore={() => void resultsQuery.fetchNextPage()}
                onRetryLoadMore={() => void resultsQuery.fetchNextPage()}
                paginationError={resultsQuery.isFetchNextPageError}
                {...(previewAsset
                  ? { previewItemId: previewAsset.id }
                  : {})}
                {...(preview.selectedItemId
                  ? { selectedItemId: preview.selectedItemId }
                  : {})}
              />
            </>
          )}
        </div>
        {previewItem && (
          <MediaPreview
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
        <FunnelSimple aria-hidden="true" size={18} />
        {t("search.filters")}
      </legend>
      <label>
        <span>{t("search.scope")}</span>
        <select
          onChange={(event) =>
            onChange({
              ...state,
              recursive: false,
              scope: event.target.value as SearchScope,
            })
          }
          value={state.scope}
        >
          {hasLibrary && (
            <option value="library">{t("search.scopeLibrary")}</option>
          )}
          {hasLibrary && (
            <option disabled={!canUseDirectory} value="directory">
              {t("search.scopeDirectory")}
            </option>
          )}
          <option value="all">{t("search.scopeAll")}</option>
        </select>
      </label>
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
      <label>
        <span>{t("search.kind")}</span>
        <select
          onChange={(event) =>
            onChange({ ...state, kind: event.target.value as SearchKind })
          }
          value={state.kind}
        >
          <option value="all">{t("search.kindAll")}</option>
          <option value="image">{t("browse.kindImage")}</option>
          <option value="animated">{t("browse.kindAnimated")}</option>
          <option value="video">{t("browse.kindVideo")}</option>
        </select>
      </label>
      <label>
        <span>{t("search.date")}</span>
        <select
          onChange={(event) =>
            onChange({ ...state, date: event.target.value as SearchDate })
          }
          value={state.date}
        >
          <option value="any">{t("search.dateAny")}</option>
          <option value="30d">{t("search.date30d")}</option>
          <option value="year">{t("search.dateYear")}</option>
        </select>
      </label>
      <label>
        <span>{t("search.sort")}</span>
        <select
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
        </select>
      </label>
    </fieldset>
  );
}

function mapSearchItems(
  assets: Asset[],
  locale: string,
  sourceTemplate: string,
): MediaCollectionItem[] {
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
    sourceHref: paths.browse(asset.libraryId, asset.directoryId),
    sourceLabel: sourceTemplate
      .replace("{library}", asset.libraryName)
      .replace("{path}", sourceDirectory(asset.relativePath)),
    thumbnailStatus: asset.thumbnail.status,
    thumbnailUrl: asset.thumbnail.url,
    width: asset.width,
  }));
}

function sourceDirectory(relativePath: string): string {
  const separator = relativePath.lastIndexOf("/");
  return separator < 0 ? "/" : relativePath.slice(0, separator) || "/";
}
