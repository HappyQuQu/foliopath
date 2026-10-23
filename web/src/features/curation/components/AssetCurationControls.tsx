import { Heart, LockKey, Plus, X } from "@phosphor-icons/react";
import { useId, useRef, useState, type FormEvent } from "react";

import { Button, ErrorState, IconButton, Input, LoadingState } from "../../../components/ui";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { useTagDisplay } from "../../../lib/i18n/tagDisplay";
import { ApiError } from "../../../lib/api/errors";
import {
  useAddManualAssetTagMutation,
  useAssetCurationQuery,
  useCreateAndAssignTagMutation,
  useFavoriteMutation,
  useReplaceAssetTagsMutation,
  useTagsQuery,
} from "../queries";
import styles from "./AssetCurationControls.module.css";

export function AssetCurationControls({
  assetId,
  csrfToken,
}: {
  assetId: string;
  csrfToken: string;
}) {
  return <SingleAssetCurationControls key={assetId} assetId={assetId} csrfToken={csrfToken} />;
}

function SingleAssetCurationControls({ assetId, csrfToken }: { assetId: string; csrfToken: string }) {
  const { t } = useLocale();
  const { tagName: displayName } = useTagDisplay();
  const headingId = useId();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const availableTagsHeadingId = useId();
  const newTagHintId = useId();
  const [tagName, setTagName] = useState("");
  const stateQuery = useAssetCurationQuery(assetId);
  const tagsQuery = useTagsQuery();
  const favoriteMutation = useFavoriteMutation();
  const createMutation = useCreateAndAssignTagMutation();
  const addMutation = useAddManualAssetTagMutation();
  const replaceMutation = useReplaceAssetTagsMutation();
  const tagMutationPending = createMutation.isPending || replaceMutation.isPending || addMutation.isPending;
  const mutationPending = tagMutationPending || favoriteMutation.isPending;

  function submitTag(event: FormEvent) {
    event.preventDefault();
    const name = tagName.trim();
    if (!name || mutationPending || (stateQuery.data?.tags.length ?? 20) >= 20) return;
    // Move focus before pending state disables the initiating control. Never restore
    // it from an asynchronous callback after the user has moved elsewhere.
    headingRef.current?.focus();
    createMutation.mutate(
      { assetId, csrfToken, name },
      { onSuccess: () => setTagName("") },
    );
  }

  if (stateQuery.isPending) {
    return <LoadingState label={t("curation.loadingState")} />;
  }
  if (stateQuery.isError) {
    return (
      <ErrorState
        message={t("curation.stateFailed")}
        onRetry={() => void stateQuery.refetch()}
      />
    );
  }

  const state = stateQuery.data;
  const selectedTagIds = new Set(state.tags.map((tag) => tag.id));
  const availableTags = (tagsQuery.data?.pages.flatMap((page) => page.items) ?? [])
    .filter((tag) => !selectedTagIds.has(tag.id));
  const atTagLimit = state.tags.length >= 20;

  function addExistingTag(tagId: string) {
    if (mutationPending || selectedTagIds.has(tagId) || atTagLimit) return;
    headingRef.current?.focus();
    addMutation.mutate({
      assetId,
      csrfToken,
      revision: state.revision,
      tagId,
    });
  }

  return (
    <section className={styles.panel} aria-labelledby={headingId}>
      <div className={styles.heading}>
        <h3 id={headingId} ref={headingRef} tabIndex={-1}>{t("curation.organize")}</h3>
        <Button
          aria-pressed={state.favorite}
          disabled={mutationPending}
          loading={favoriteMutation.isPending}
          onClick={() => {
            headingRef.current?.focus();
            favoriteMutation.mutate({
              assetId,
              csrfToken,
              favorite: !state.favorite,
            });
          }}
          size="small"
          variant="secondary"
        >
          <Heart aria-hidden="true" size={16} weight={state.favorite ? "fill" : "regular"} />
          {state.favorite ? t("curation.favorited") : t("curation.favorite")}
        </Button>
      </div>
      <div className={styles.tags} aria-label={t("curation.tags")}>
        {state.tags.length === 0 && (
          <span className={styles.emptyTags}>{t("curation.noTags")}</span>
        )}
        {state.tags.map((tag) => (
          <span className={styles.tag} key={tag.id}>
            <span className={styles.tagName}>{displayName(tag)}</span>
            <small title={t(tag.source === "ai" ? "curation.aiSourceHint" : "curation.manualSourceHint")}>{t(tag.source === "ai" ? "curation.aiSource" : "curation.manualSource")}</small>
            {tag.source === "ai" && <IconButton
              disabled={mutationPending}
              label={t("curation.keepManualTag").replace("{name}", displayName(tag))}
              onClick={() => {
                headingRef.current?.focus();
                addMutation.mutate({ assetId, csrfToken, revision: state.revision, tagId: tag.id });
              }}
            ><LockKey aria-hidden="true" size={13} /></IconButton>}
            <IconButton
              disabled={mutationPending}
              label={t("curation.removeTag").replace("{name}", displayName(tag))}
              onClick={() => {
                headingRef.current?.focus();
                replaceMutation.mutate({
                  assetId,
                  csrfToken,
                  revision: state.revision,
                  tagIds: state.tags.filter((item) => item.id !== tag.id).map((item) => item.id),
                });
              }}
            >
              <X aria-hidden="true" size={13} />
            </IconButton>
          </span>
        ))}
      </div>
      <section
        aria-labelledby={availableTagsHeadingId}
        className={styles.availableSection}
      >
        <p className={styles.emptyTags}>{t("curation.automaticProtection")}</p>
        <h4 id={availableTagsHeadingId}>{t("curation.availableTags")}</h4>
        {tagsQuery.isPending ? (
          <LoadingState label={t("curation.loadingState")} />
        ) : tagsQuery.isError && !tagsQuery.data ? (
          <ErrorState
            message={t("curation.stateFailed")}
            onRetry={() => void tagsQuery.refetch()}
          />
        ) : (
          <>
            <ul className={styles.availableTags}>
              {availableTags.map((tag) => (
                <li key={tag.id}>
                  <Button
                    aria-label={t("curation.addExistingTag").replace("{name}", displayName(tag))}
                    className={styles.availableTag}
                    disabled={mutationPending || atTagLimit}
                    onClick={() => addExistingTag(tag.id)}
                    size="small"
                    variant="secondary"
                  >
                    <Plus aria-hidden="true" size={15} />
                    {displayName(tag)}
                  </Button>
                </li>
              ))}
              {availableTags.length === 0 && (
                <li className={styles.emptyTags}>{t("curation.noAvailableTags")}</li>
              )}
            </ul>
            {tagsQuery.isFetchNextPageError && (
              <ErrorState
                message={t("curation.stateFailed")}
                onRetry={() => void tagsQuery.fetchNextPage()}
              />
            )}
            {tagsQuery.hasNextPage && (
              <Button
                className={styles.loadMoreTags}
                loading={tagsQuery.isFetchingNextPage}
                onClick={() => {
                  if (!tagsQuery.isFetchingNextPage) void tagsQuery.fetchNextPage();
                }}
                size="small"
                variant="quiet"
              >
                {t("curation.loadMoreTags")}
              </Button>
            )}
          </>
        )}
      </section>
      <form className={styles.form} onSubmit={submitTag}>
        <p className={styles.newTagHint} id={newTagHintId}>
          {t("curation.newTagHint")}
        </p>
        <Input
          aria-describedby={newTagHintId}
          aria-label={t("curation.newTag")}
          disabled={mutationPending || atTagLimit}
          maxLength={128}
          onChange={(event) => setTagName(event.target.value)}
          placeholder={t("curation.newTagPlaceholder")}
          value={tagName}
        />
        <Button
          className={styles.addTagButton}
          disabled={!tagName.trim() || mutationPending || atTagLimit}
          loading={createMutation.isPending}
          size="small"
          type="submit"
          variant="secondary"
        >
          <Plus aria-hidden="true" size={16} />
          {t("curation.createTag")}
        </Button>
      </form>
      {(createMutation.isError || replaceMutation.isError || addMutation.isError || favoriteMutation.isError) && (
        <p className={styles.error} role="alert">{t([createMutation.error, replaceMutation.error, addMutation.error].some(error => error instanceof ApiError && error.code === "precondition_failed") ? "curation.revisionConflict" : "curation.updateFailed")}</p>
      )}
    </section>
  );
}
