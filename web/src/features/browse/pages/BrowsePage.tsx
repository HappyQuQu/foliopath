import {
  CaretDown,
  CaretRight,
  Copy,
  Folder,
  FolderOpen,
  House,
  ImageSquare,
} from "@phosphor-icons/react";
import { useEffect, useMemo, useState, type CSSProperties } from "react";
import { Link, useNavigate } from "react-router-dom";

import { AppShell } from "../../../components/patterns/AppShell/AppShell";
import {
  Button,
  ErrorState,
  IconButton,
  InlineStatus,
  LoadingState,
  useToast,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import type { Breadcrumb, Directory } from "../../../lib/api/catalog";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import {
  useLibrariesQuery,
  useLibraryQuery,
} from "../../libraries";
import { useDirectoriesQuery, useDirectoryQuery } from "../queries";
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
  const { t } = useLocale();
  const navigate = useNavigate();
  const toast = useToast();
  const librariesQuery = useLibrariesQuery();
  const libraryQuery = useLibraryQuery(libraryId);
  const directoryQuery = useDirectoryQuery(directoryId);
  const childrenQuery = useDirectoriesQuery({ libraryId, parentId: directoryId });
  const { refetch: refreshLibrary } = libraryQuery;
  const { refetch: refreshDirectory } = directoryQuery;
  const {
    fetchNextPage: loadMoreChildren,
    refetch: refreshChildren,
  } = childrenQuery;
  const libraries = useMemo(
    () => librariesQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [librariesQuery.data],
  );
  const children = useMemo(
    () => childrenQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [childrenQuery.data],
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
  const browseHref = paths.browse(libraryId, directoryId);

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
      settingsHref={paths.generalSettings}
      sidebarContent={
        <DirectoryNavigation
          currentLibraryName={currentLibrary?.name}
          libraries={libraries}
          libraryId={libraryId}
          onLibraryChange={(nextLibraryId) =>
            navigate(paths.browse(nextLibraryId))
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
            breadcrumbs={breadcrumbs}
            libraryId={libraryId}
            rootName={currentLibrary?.name ?? currentName}
          />
          <IconButton label={t("browse.copyLink")} onClick={() => void copyDirectLink()}>
            <Copy aria-hidden="true" size={19} />
          </IconButton>
        </header>

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
                <div className={styles.headingRow}>
                  <div>
                    <p className={styles.eyebrow}>{t("browse.currentDirectory")}</p>
                    <h1 id="browse-heading">{currentName}</h1>
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
                <section aria-labelledby="child-directories-heading">
                  <h2 id="child-directories-heading">
                    {t("browse.childDirectories")}
                  </h2>
                  {childrenQuery.isPending && (
                    <LoadingState label={t("browse.loadingDirectories")} />
                  )}
                  {childrenQuery.isError && (
                    <ErrorState
                      message={t("browse.directoriesFailed")}
                      onRetry={() => void refreshChildren()}
                    />
                  )}
                  {childrenQuery.isSuccess && children.length === 0 && (
                    <div className={styles.empty}>
                      <FolderOpen aria-hidden="true" size={34} />
                      <p>{t("browse.noChildDirectories")}</p>
                    </div>
                  )}
                  {children.length > 0 && (
                    <div className={styles.folderGrid}>
                      {children.map((directory) => (
                        <Link
                          className={styles.folderCard}
                          key={directory.id}
                          to={paths.browse(libraryId, directory.id)}
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
                <section className={styles.mediaPlaceholder} aria-labelledby="media-heading">
                  <ImageSquare aria-hidden="true" size={32} />
                  <div>
                    <h2 id="media-heading">{t("browse.mediaHeading")}</h2>
                    <p>{t("browse.mediaNextSlice")}</p>
                  </div>
                </section>
              </>
            )}
        </div>
      </section>
    </AppShell>
  );
}

function Breadcrumbs({
  breadcrumbs,
  libraryId,
  rootName,
}: {
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
              <Link to={paths.browse(libraryId, index === 0 ? undefined : item.id)}>
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
  currentLibraryName,
  libraries,
  libraryId,
  onLibraryChange,
  selectedDirectoryId,
  selectedPathIds,
}: {
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
          to={paths.browse(libraryId)}
        >
          <ImageSquare aria-hidden="true" size={18} />
          <span>{t("browse.allMedia")}</span>
        </Link>
        <DirectoryChildren
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
  depth,
  libraryId,
  parentId,
  selectedDirectoryId,
  selectedPathIds,
}: {
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
  depth,
  directory,
  libraryId,
  selectedDirectoryId,
  selectedPathIds,
}: {
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
          to={paths.browse(libraryId, directory.id)}
        >
          <Folder aria-hidden="true" size={17} />
          <span>{directory.name}</span>
        </Link>
      </div>
      {expanded && directory.hasChildren && (
        <DirectoryChildren
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
