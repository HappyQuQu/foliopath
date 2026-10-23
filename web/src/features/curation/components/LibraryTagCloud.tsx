import { useEffect, useId, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { Button, EmptyState, ErrorState, Input, LoadingState } from "../../../components/ui";
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
  const { tagName } = useTagDisplay();
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
      <header>
        <h2 id={headingId}>{t("tagCloud.title")}</h2>
        <p>{t("tagCloud.scope")}</p>
      </header>
      <TagLanguageControl />
      <form className={styles.search} role="search" onSubmit={event => { event.preventDefault(); search(draft); }}>
        <Input aria-label={t("tagCloud.search")} type="search" maxLength={32} value={draft}
          placeholder={t("tagCloud.search")} onChange={event => setDraft(event.target.value)} />
        <Button size="small" type="submit" variant="secondary">{t("tagCloud.searchAction")}</Button>
      </form>
      {q && <Button size="small" variant="quiet" onClick={() => search("")}>{t("tagCloud.clearSearch")}</Button>}
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
          <p className={styles.legend}>{t("tagCloud.legend")}</p>
          <nav className={styles.tags} aria-label={t("tagCloud.title")}>
            {tags.map(tag => {
              const weight = tagWeight(tag.assetCount);
              const target = new URLSearchParams(params);
              target.set("libraryId", libraryId);
              target.set("kind", "image");
              return <Link key={tag.id} className={styles.tag} data-weight={weight}
                data-frequency={weight >= 4 ? "frequent" : weight >= 3 ? "regular" : "rare"}
                aria-current={tag.id === selectedTagId ? "page" : undefined}
                to={`${paths.tag(tag.id)}?${target.toString()}`}>
                <span>{tagName(tag)}</span>
                <small aria-label={t("tagCloud.count").replace("{count}", String(tag.assetCount))}>{tag.assetCount}</small>
              </Link>;
            })}
          </nav>
          <p className={styles.legend}>{t("tagCloud.pageCount").replace("{count}", String(tags.length))}</p>
        </>
      )}
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

// Stable across pagination and search; never compare an incomplete page to the library.
function tagWeight(count: number): number {
  return count >= 200 ? 5 : count >= 50 ? 4 : count >= 10 ? 3 : count >= 3 ? 2 : 1;
}
