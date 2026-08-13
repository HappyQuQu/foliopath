import { Heart, Plus, X } from "@phosphor-icons/react";
import { useId, useState, type FormEvent } from "react";

import { Button, ErrorState, IconButton, Input, LoadingState } from "../../../components/ui";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import {
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
  const { t } = useLocale();
  const availableTagsHeadingId = useId();
  const newTagHintId = useId();
  const [tagName, setTagName] = useState("");
  const stateQuery = useAssetCurationQuery(assetId);
  const tagsQuery = useTagsQuery();
  const favoriteMutation = useFavoriteMutation();
  const createMutation = useCreateAndAssignTagMutation();
  const replaceMutation = useReplaceAssetTagsMutation();
  const tagMutationPending = createMutation.isPending || replaceMutation.isPending;
  const mutationPending = tagMutationPending || favoriteMutation.isPending;

  function submitTag(event: FormEvent) {
    event.preventDefault();
    const name = tagName.trim();
    if (!name || mutationPending || (stateQuery.data?.tags.length ?? 20) >= 20) return;
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
    replaceMutation.mutate({
      assetId,
      csrfToken,
      revision: state.revision,
      tagIds: [...state.tags.map((tag) => tag.id), tagId],
    });
  }

  return (
    <section className={styles.panel} aria-labelledby="asset-curation-heading">
      <div className={styles.heading}>
        <h3 id="asset-curation-heading">{t("curation.organize")}</h3>
        <Button
          aria-pressed={state.favorite}
          disabled={mutationPending}
          loading={favoriteMutation.isPending}
          onClick={() =>
            favoriteMutation.mutate({
              assetId,
              csrfToken,
              favorite: !state.favorite,
            })
          }
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
            {tag.name}
            <IconButton
              disabled={mutationPending}
              label={t("curation.removeTag").replace("{name}", tag.name)}
              onClick={() =>
                replaceMutation.mutate({
                  assetId,
                  csrfToken,
                  revision: state.revision,
                  tagIds: state.tags.filter((item) => item.id !== tag.id).map((item) => item.id),
                })
              }
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
                    aria-label={t("curation.addExistingTag").replace("{name}", tag.name)}
                    className={styles.availableTag}
                    disabled={mutationPending || atTagLimit}
                    onClick={() => addExistingTag(tag.id)}
                    size="small"
                    variant="secondary"
                  >
                    <Plus aria-hidden="true" size={15} />
                    {tag.name}
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
      {(createMutation.isError || replaceMutation.isError || favoriteMutation.isError) && (
        <p className={styles.error} role="alert">{t("curation.updateFailed")}</p>
      )}
    </section>
  );
}
