import { MagnifyingGlass, X } from "@phosphor-icons/react";
import { useEffect, useId, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { Button, EmptyState, ErrorState, IconButton, Input, LoadingState } from "../../../components/ui";
import { ApiError } from "../../../lib/api/errors";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { useTagDisplay } from "../../../lib/i18n/tagDisplay";
import { paths } from "../../../routes/paths";
import { useLibraryTagCloudQuery } from "../queries";
import styles from "./LibraryTagCloud.module.css";
import { TagLanguageControl } from "./TagLanguageControl";

/** One bounded server page; the global tag vocabulary remains a separate query. */
export function LibraryTagCloud({ libraryId, selectedTagId }: {
  libraryId: string;
  selectedTagId?: string;
}) {
  const { t } = useLocale();
  const { tagName, language } = useTagDisplay();
  const headingId = useId();
  const [params, setParams] = useSearchParams();
  const q = params.get("tagQ") ?? "";
  const cursor = params.get("tagCursor") ?? "";
  const [draft, setDraft] = useState(q);
  const [predecessors, setPredecessors] = useState(() => new Map<string, string>());
  const predecessor = predecessors.get(cursor);
  const query = useLibraryTagCloudQuery(libraryId, q, cursor);
  const tags = query.data?.items ?? [];
  const expired = query.error instanceof ApiError && query.error.code === "invalid_cursor";

  useEffect(() => { setDraft(q); setPredecessors(new Map()); }, [libraryId, q]);

  function changePage(nextCursor: string) {
    const next = new URLSearchParams(params);
    if (nextCursor) next.set("tagCursor", nextCursor); else next.delete("tagCursor");
    setParams(next);
  }
  function search(value: string) {
    const next = new URLSearchParams(params);
    const normalized = value.trim().normalize("NFC");
    if (normalized) next.set("tagQ", normalized); else next.delete("tagQ");
    next.delete("tagCursor");
    setPredecessors(new Map());
    setParams(next);
  }

  return (
    <section className={styles.cloud} aria-labelledby={headingId}>
      <header className={styles.heading}>
        <div><h2 id={headingId}>{t("tagCloud.title")}</h2>{query.isSuccess && <small title={t("tagCloud.pageCount").replace("{count}", String(tags.length))}>{t("tagCloud.pageShort").replace("{count}", String(tags.length))}</small>}
          {selectedTagId && <Link className={styles.clearFilter} aria-label={t("tagCloud.clearFilter")} title={t("tagCloud.clearFilter")} to={`${paths.tags}?${params.toString()}`}><X size={16} aria-hidden="true" /></Link>}
        </div>
        <p title={t("tagCloud.scope")}>{t("tagCloud.scopeCompact")}</p>
      </header>
      <div className={styles.tools}>
      <form className={styles.search} role="search" onSubmit={event => { event.preventDefault(); search(draft); }}>
        <Input aria-label={t("tagCloud.search")} type="search" maxLength={32} value={draft}
          placeholder={t("tagCloud.search")} onChange={event => setDraft(event.target.value)} />
        <IconButton label={t("tagCloud.searchAction")} type="submit"><MagnifyingGlass aria-hidden="true" size={17} /></IconButton>
      </form>
      </div>
      {q && <Button size="small" variant="quiet" onClick={() => search("")}>{t("tagCloud.clearSearch")}</Button>}
      <div className={styles.scrollArea}>
      {query.isPending ? <LoadingState /> : query.isError ? (
        <>
          <ErrorState message={t(expired ? "tagCloud.expired" : "tagCloud.failed")}
            {...(!expired ? { onRetry: () => void query.refetch() } : {})} />
          {expired && <Button size="small" onClick={() => { setPredecessors(new Map()); changePage(""); }}>{t("tagCloud.first")}</Button>}
        </>
      ) : tags.length === 0 ? (
        <EmptyState title={t(q ? "tagCloud.noMatch" : "tagCloud.empty")}
          description={t(q ? "tagCloud.noMatchDescription" : "tagCloud.emptyDescription")} />
      ) : (
        <>
          <p className={styles.visuallyHidden} id={`${headingId}-legend`}>{t("tagCloud.legend")}</p>
          <nav className={styles.tags} aria-label={t("tagCloud.title")} aria-describedby={`${headingId}-legend`}>
            {tags.map(tag => {
              const bilingual = language === "bilingual" ? tag.localization : undefined;
              const target = new URLSearchParams(params);
              target.set("libraryId", libraryId);
              target.set("kind", "image");
              return <Link key={tag.id} className={styles.tag} data-weight={tagWeight(tag.assetCount)}
                aria-current={tag.id === selectedTagId ? "page" : undefined}
                to={`${paths.tag(tag.id)}?${target.toString()}`}>
                <span className={styles.tagName} title={tagName(tag)}>
                  <span>{bilingual?.zhCN ?? tagName(tag)}</span>
                  {bilingual && <span className={styles.translation} lang="en">{bilingual.en}</span>}
                </span>
                <small aria-label={t("tagCloud.count").replace("{count}", String(tag.assetCount))}>{tag.assetCount}</small>
              </Link>;
            })}
          </nav>
        </>
      )}
      </div>
      <footer className={styles.language}><TagLanguageControl compact /></footer>
      {!query.isError && (cursor || query.data?.nextCursor) && <div className={styles.pagination}>
        <Button size="small" variant="quiet" disabled={!cursor || query.isFetching} onClick={() => {
          changePage(predecessor ?? "");
        }}>{t(cursor && predecessor === undefined ? "tagCloud.first" : "tagCloud.previous")}</Button>
        <Button size="small" variant="quiet" disabled={!query.data?.nextCursor || query.isFetching} onClick={() => {
          if (!query.data?.nextCursor) return;
          const nextCursor = query.data.nextCursor;
          setPredecessors(value => new Map(value).set(nextCursor, cursor));
          changePage(nextCursor);
        }}>{t("tagCloud.next")}</Button>
      </div>}
    </section>
  );
}

function tagWeight(count: number): number {
  return count >= 200 ? 5 : count >= 50 ? 4 : count >= 10 ? 3 : count >= 3 ? 2 : 1;
}
