import { CaretDown, CaretRight, FileZip, Folder } from "@phosphor-icons/react";
import { InfiniteQueryObserver, notifyManager, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore, type CSSProperties } from "react";
import { Link, useNavigate } from "react-router-dom";

import { Button, IconButton, InlineStatus, Menu, type MenuAction } from "../../../components/ui";
import type { Directory, DirectoryPage } from "../../../lib/api/catalog";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { retryInfiniteNextPage } from "../../../lib/query/retryInfiniteNextPage";
import { catalogKeys, directoryQueryOptions } from "../queries";
import { browseUrl, defaultBrowseUrlState, directoryBrowseUrl, type BrowseUrlState } from "../urlState";
import { paths } from "../../../routes/paths";
import styles from "./BrowsePage.module.css";

type TreeRow = { key: string; depth: number } & (
  { kind: "root" } |
  { kind: "directory"; directory: Directory; expanded: boolean } |
  { kind: "load"; parentId: string | undefined; error: boolean; loading: boolean; initial: boolean }
);

type MenuTarget = {
  anchor: HTMLElement | { x: number; y: number };
  trigger: HTMLElement;
  directory?: Directory;
  root?: boolean;
};

export function DirectoryTree({ libraryId, libraryName, rootSelected, selectedDirectoryId, selectedPathIds, browseState, currentDirectory, canDelete, onDeleteDirectory }: {
  currentDirectory?: Directory | undefined;
  canDelete?: boolean | undefined;
  onDeleteDirectory?: ((directory: Directory) => void) | undefined;
  libraryId: string;
  libraryName: string;
  rootSelected: boolean;
  selectedDirectoryId?: string | undefined;
  selectedPathIds: Set<string>;
  browseState: BrowseUrlState;
}) {
  const { t } = useLocale();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const scrollRef = useRef<HTMLElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const inFlight = useRef(new Set<string>());
  const cacheVersion = useRef(0);
  const [workVersion, setWorkVersion] = useState(0);
  const [scrolled, setScrolled] = useState(false);
  const [menu, setMenu] = useState<MenuTarget | null>(null);
  const [expansion, setExpansion] = useState<{ all: boolean; exceptions: Set<string> }>(() => ({ all: false, exceptions: new Set(selectedPathIds) }));
  const pathKey = [...selectedPathIds].join(":");
  const scopeRef = useRef(libraryId);
  scopeRef.current = libraryId;

  const subscribe = useCallback((notify: () => void) => {
    const deliver = notifyManager.batchCalls(notify);
    return queryClient.getQueryCache().subscribe(event => {
    const key = event.query.queryKey;
    if (key[0] === "catalog" && key[1] === "directories" && key[2] === libraryId && key[4] === "") {
      cacheVersion.current++;
      deliver();
    }
    });
  }, [queryClient, libraryId]);
  const revision = useSyncExternalStore(subscribe, () => cacheVersion.current, () => 0);

  useEffect(() => {
    setExpansion(current => {
      const exceptions = new Set(current.exceptions);
      for (const id of selectedPathIds) {
        if (current.all) exceptions.delete(id); else exceptions.add(id);
      }
      return { ...current, exceptions };
    });
    setMenu(null);
    // Navigation opens its new path; collapsing the same path must not reopen it.
  }, [pathKey]);

  function expanded(id: string) { return expansion.all !== expansion.exceptions.has(id); }
  function toggle(id: string) {
    setExpansion(current => {
      const exceptions = new Set(current.exceptions);
      if (exceptions.has(id)) exceptions.delete(id); else exceptions.add(id);
      return { ...current, exceptions };
    });
  }

  const rows = useMemo(() => {
    const result: TreeRow[] = [{ key: "root", depth: 0, kind: "root" }];
    function append(parentId: string | undefined, depth: number) {
      const state = queryClient.getQueryState<InfiniteData<DirectoryPage>>(catalogKeys.directories(libraryId, parentId, ""));
      for (const page of state?.data?.pages ?? []) {
        for (const directory of page.items) {
          const isExpanded = expansion.all !== expansion.exceptions.has(directory.id);
          result.push({ key: directory.id, depth, kind: "directory", directory, expanded: isExpanded });
          if (directory.hasChildren && isExpanded) append(directory.id, depth + 1);
        }
      }
      if (!state?.data || state.data.pages.at(-1)?.nextCursor || state.isInvalidated && state.status === "error") {
        result.push({ key: `load:${parentId ?? "root"}`, depth, kind: "load", parentId, error: state?.status === "error", loading: state?.fetchStatus === "fetching", initial: !state?.data });
      }
    }
    append(undefined, 1);
    return result;
  }, [queryClient, libraryId, expansion, revision]);

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    getItemKey: index => rows[index]!.key,
    estimateSize: () => 32,
    measureElement: element => element.getBoundingClientRect().height || 32,
    overscan: 6,
    observeElementRect: (instance, callback) => {
      const element = instance.scrollElement;
      if (!element) return;
      const update = () => callback({ width: element.clientWidth || window.innerWidth, height: element.clientHeight || window.innerHeight });
      update();
      const observer = new ResizeObserver(update);
      observer.observe(element);
      return () => observer.disconnect();
    },
  });
  const visible = virtualizer.getVirtualItems();
  const visibleKey = visible.map(item => item.key).join(":");

  const load = useCallback(async (parentId: string | undefined, refresh = false) => {
    const key = parentId ?? "root";
    if (inFlight.current.has(key) || inFlight.current.size >= 2) return;
    inFlight.current.add(key);
    const observer = new InfiniteQueryObserver(queryClient, directoryQueryOptions({ libraryId, parentId }));
    try {
      if (refresh || !observer.getCurrentResult().hasNextPage && queryClient.getQueryState(catalogKeys.directories(libraryId, parentId, ""))?.isInvalidated) {
        await observer.refetch();
      } else await retryInfiniteNextPage({
        error: observer.getCurrentResult().error,
        loadNextPage: () => observer.fetchNextPage({ cancelRefetch: false }),
        refresh: () => observer.refetch(),
      });
    } finally {
      observer.destroy();
      inFlight.current.delete(key);
      if (scopeRef.current === libraryId) setWorkVersion(value => value + 1);
    }
  }, [queryClient, libraryId]);

  useEffect(() => {
    const parents = [undefined, ...rows.flatMap(row => row.kind === "directory" && row.expanded && row.directory.hasChildren ? [row.directory.id] : [])];
    for (const parentId of parents) {
      const state = queryClient.getQueryState(catalogKeys.directories(libraryId, parentId, ""));
      if (state?.isInvalidated && state.fetchStatus !== "fetching" && state.status !== "error") void load(parentId, true);
    }
    for (const item of visible) {
      const row = rows[item.index];
      if (row?.kind === "load" && !row.error && !row.loading && (row.initial || scrolled || expansion.all)) void load(row.parentId);
    }
  }, [visibleKey, revision, workVersion, rows, load, scrolled, expansion.all, queryClient, libraryId]);

  const targetDirectory = menu?.directory ?? (menu?.root ? undefined : currentDirectory);
  const menuItems: MenuAction[] = menu?.directory ? [
    { id: "toggle", label: t(expanded(menu.directory.id) ? "context.collapseBranch" : "context.expandBranch"), disabled: !menu.directory.hasChildren, onSelect: () => toggle(menu.directory!.id) },
  ] : [
    { id: "expand", label: t("context.expandAllDirectories"), disabled: rows.length === 1, onSelect: () => setExpansion({ all: true, exceptions: new Set() }) },
    { id: "collapse", label: t("context.collapseAllDirectories"), disabled: !expansion.all && !rows.some(row => row.kind === "directory" && row.expanded && row.directory.hasChildren), onSelect: () => { setExpansion({ all: false, exceptions: new Set() }); virtualizer.scrollToOffset(0); } },
  ];

  menuItems.push({ id: "delete", label: t(targetDirectory?.archivePath === targetDirectory?.relativePath && targetDirectory?.archivePath ? "deletion.deleteArchive" : menu?.directory ? "deletion.deleteDirectory" : "deletion.deleteCurrent"), danger: true, separatorBefore: true, disabled: !canDelete || !onDeleteDirectory || !targetDirectory || targetDirectory.relativePath === "" || Boolean(targetDirectory.archivePath && targetDirectory.archivePath !== targetDirectory.relativePath), onSelect: () => { if (targetDirectory) onDeleteDirectory?.(targetDirectory); } });
  menuItems.push({ id: "tasks", label: t("deletion.tasks"), href: paths.fileTasks, onSelect: () => navigate(paths.fileTasks) }, { id: "settings", label: t("deletion.settings"), href: paths.deletionSettings, onSelect: () => navigate(paths.deletionSettings) });

  function contextMenu(event: React.MouseEvent<HTMLElement>, directory?: Directory) {
    event.preventDefault();
    const target = event.currentTarget.querySelector<HTMLElement>("a") ?? event.currentTarget;
    setMenu({ anchor: { x: event.clientX, y: event.clientY }, trigger: target, ...(directory ? { directory } : { root: true }) });
  }

  return <>
    <div className={styles.directoryTreeHeader}>
      <p className={styles.treeLabel}>{t("browse.directory")}</p>
      <Button size="small" variant="quiet" ref={triggerRef} aria-haspopup="menu" aria-expanded={menu !== null && !menu.directory}
        onClick={() => { if (menu) setMenu(null); else if (triggerRef.current) setMenu({ anchor: triggerRef.current, trigger: triggerRef.current }); }}>
        {t("context.directoryActions")} <CaretDown aria-hidden="true" size={12} />
      </Button>
    </div>
    <nav aria-label={t("browse.directoryNavigation")} className={`${styles.tree} ${styles.directoryTree}`} ref={scrollRef} onScroll={() => setScrolled(true)}>
      <div className={styles.directoryTreeCanvas} style={{ height: virtualizer.getTotalSize() }}>
        {visible.map(item => {
          const row = rows[item.index]!;
          const directory = row.kind === "directory" ? row.directory : undefined;
          const name = directory?.name ?? libraryName;
          return <div key={item.key} className={styles.directoryTreePosition} data-index={item.index} ref={virtualizer.measureElement}
            style={{ transform: `translateY(${item.start}px)` }}>
            {row.kind === "load" ? <div className={`${styles.treeRow} ${styles.directoryTreeLoad}`} style={{ "--tree-depth": row.depth } as CSSProperties}>
              {row.error && <InlineStatus tone="danger">{t("browse.loadMoreDirectoriesFailed")}</InlineStatus>}
              <Button size="small" variant="quiet" loading={row.loading} onClick={() => void load(row.parentId)}>
                {t(row.error ? "browse.retryLoadMoreDirectories" : "browse.loadMoreDirectories")}
              </Button>
            </div> : <div className={styles.treeRow} style={{ "--tree-depth": row.depth } as CSSProperties}
              onContextMenu={event => contextMenu(event, directory)}
              onKeyDown={event => {
                if (event.key === "ContextMenu" || event.key === "F10" && event.shiftKey) {
                  event.preventDefault();
                  const anchor = event.currentTarget.querySelector<HTMLElement>("a")!;
                  setMenu({ anchor, trigger: anchor, ...(directory ? { directory } : { root: true }) });
                }
              }}>
              {row.kind === "directory" && directory?.hasChildren ? <IconButton className={styles.treeToggle}
                label={t(row.expanded ? "browse.collapseDirectory" : "browse.expandDirectory").replace("{name}", name)} aria-expanded={row.expanded}
                onClick={() => toggle(directory.id)}>{row.expanded ? <CaretDown aria-hidden="true" size={15} /> : <CaretRight aria-hidden="true" size={15} />}</IconButton>
                : <span className={row.kind === "root" ? styles.treeToggleStatic : styles.treeSpacer}>{row.kind === "root" && <CaretDown aria-hidden="true" size={15} />}</span>}
              <Link className={`${styles.treeLink} ${(directory ? directory.id === selectedDirectoryId : rootSelected) ? styles.treeLinkCurrent : ""}`}
                aria-current={(directory ? directory.id === selectedDirectoryId : rootSelected) ? "page" : undefined}
                title={name}
                onFocus={() => virtualizer.scrollToIndex(item.index, { align: "auto" })}
                to={directory ? directoryBrowseUrl(libraryId, directory, browseState) : browseUrl(libraryId, undefined, defaultBrowseUrlState())}>
                {directory?.archivePath === directory?.relativePath && directory?.archivePath ? <FileZip aria-hidden="true" size={17} /> : <Folder aria-hidden="true" size={17} />}<span>{name}</span>
              </Link>
            </div>}
          </div>;
        })}
      </div>
    </nav>
    {menu && <Menu label={t("context.directoryActions")} heading={menu.directory?.name ?? (menu.root ? libraryName : currentDirectory?.name ?? libraryName)} anchor={menu.anchor}
      returnFocusTo={menu.trigger.isConnected ? menu.trigger : triggerRef.current} items={menuItems} onClose={() => setMenu(null)} />}
  </>;
}
