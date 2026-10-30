import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button, Dialog, DialogCloseButton, ErrorState, InlineStatus, LoadingState, Switch, useToast } from "../../components/ui";
import type { AuthenticatedSession } from "../../lib/api/auth";
import { updateDeletionSettings, type DeletionSettings, type DeletionSettingsUpdate } from "../../lib/api/deletion";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import { deletionUnavailableMessage } from "./presentation";
import { deletionKeys, useDeletionSettingsQuery } from "./queries";
import styles from "./Deletion.module.css";

export function DeletionSettingsPanel({ session }: { session: AuthenticatedSession }) {
  const query = useDeletionSettingsQuery();
  const { t } = useLocale();
  if (query.isPending) return <LoadingState label={t("deletion.checking")} />;
  if (!query.data) return <ErrorState message={t("deletion.loadFailed")} onRetry={() => void query.refetch()} />;
  return <SettingsForm key={query.data.revision} settings={query.data} session={session} />;
}
function SettingsForm({ settings, session }: { settings: DeletionSettings; session: AuthenticatedSession }) {
  const { t } = useLocale();
  const client = useQueryClient();
  const toast = useToast();
  const [enabled, setEnabled] = useState(settings.enabled);
  const [confirm, setConfirm] = useState(false);
  const save = useMutation({ mutationFn: (body: DeletionSettingsUpdate) => updateDeletionSettings(body, settings.revision, session.csrfToken),
    onSuccess: value => { client.setQueryData(deletionKeys.settings, value); toast.show({ message: t("deletion.saved"), tone: "success" }); },
    onError: () => { void client.invalidateQueries({ queryKey: deletionKeys.settings }); },
  });
  const dirty = enabled !== settings.enabled || settings.recycleBinEnabled || settings.retentionDays !== null;
  function submit() { save.mutate({ enabled, recycleBinEnabled: false, retentionDays: null, acknowledge: true }); }
  return <section aria-label={t("deletion.settings")} className={`${styles.panel} ${styles.settings}`}>
    {!settings.available && <InlineStatus tone="warning">{t(deletionUnavailableMessage(settings))}</InlineStatus>}
    <label className={styles.field}><div><strong>{t("deletion.enabled")}</strong><p>{t("deletion.enabledHint")}</p></div><Switch checked={enabled} disabled={save.isPending || (!settings.available && !enabled)} onChange={event => setEnabled(event.target.checked)} /></label>
    <InlineStatus tone="warning">{t("deletion.directHint")}</InlineStatus>
    {save.isError && <InlineStatus tone="danger">{t("deletion.settingsFailed")}</InlineStatus>}
    <div className={styles.formActions}><Button variant="primary" disabled={!dirty} loading={save.isPending} onClick={() => { if (enabled && !settings.enabled) setConfirm(true); else submit(); }}>{t("deletion.save")}</Button></div>
    <Dialog open={confirm} onOpenChange={value => { if (!save.isPending) setConfirm(value); }} title={t("deletion.confirmSettings")} actions={<><DialogCloseButton onClick={() => setConfirm(false)} /><Button variant="danger" loading={save.isPending} onClick={submit}>{t("deletion.save")}</Button></>}>
      <div className={styles.stack}><p>{t("deletion.directHint")}</p>{save.isError && <InlineStatus tone="danger">{t("deletion.settingsFailed")}</InlineStatus>}</div>
    </Dialog>
  </section>;
}
