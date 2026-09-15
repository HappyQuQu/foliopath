import { useState } from "react";
import { Button, ErrorState, FormField, InlineStatus, LoadingState, Select, useToast } from "../../../components/ui";
import type { AIModelProxySettings, AIModelProxySnapshot } from "../../../lib/api/ai";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { useAIModelProxyQuery, useSaveAIModelProxyMutation, useTestAIModelProxyMutation } from "../queries";
import styles from "./AISettingsPage.module.css";

export function AIModelProxyPanel({ csrfToken }: { csrfToken: string }) {
  const query = useAIModelProxyQuery();
  const { t } = useLocale();
  return <section className={styles.card} aria-labelledby="model-proxy-title">
    <header className={styles.cardHeader}><div><h2 id="model-proxy-title">{t("ai.proxy.title")}</h2><p>{t("ai.proxy.description")}</p></div></header>
    {query.isPending ? <LoadingState label={t("ai.proxy.loading")} /> : query.isError || !query.data ?
      <ErrorState message={t("ai.proxy.loadFailed")} onRetry={() => void query.refetch()} /> :
      <ProxyForm key={query.data.revision} value={query.data} csrfToken={csrfToken} refresh={() => void query.refetch()} />}
  </section>;
}
function ProxyForm({ value, csrfToken, refresh }: { value: AIModelProxySnapshot; csrfToken: string; refresh: () => void }) {
  const { t } = useLocale(); const toast = useToast();
  const [mode, setMode] = useState(value.mode);
  const [url, setURL] = useState(value.proxyUrl);
  const [result, setResult] = useState<"available" | "unavailable" | "failed" | null>(null);
  const save = useSaveAIModelProxyMutation(); const probe = useTestAIModelProxyMutation();
  const pending = save.isPending || probe.isPending;
  const body: AIModelProxySettings = { mode, proxyUrl: mode === "custom" ? url.trim() : "", revision: value.revision };
  const dirty = body.mode !== value.mode || body.proxyUrl !== value.proxyUrl;
  const ready = mode !== "custom" || body.proxyUrl !== "";
  return <form onSubmit={(event) => {
    event.preventDefault(); if (!ready || pending || !dirty) return;
    void save.mutateAsync({ body, csrfToken }).then(() => toast.show({ message: t("ai.proxy.saved"), tone: "success" })).catch(() => {});
  }}>
    <p>{t(value.effectiveMode === "proxy" ? "ai.proxy.currentProxy" : "ai.proxy.currentDirect")} · {t(value.mode === "inherit" ? "ai.proxy.sourceDocker" : "ai.proxy.sourceUI")}</p>
    <p>{t(value.deploymentConfigured ? "ai.proxy.dockerSet" : "ai.proxy.dockerUnset")}</p>
    <label htmlFor="model-proxy-mode">{t("ai.proxy.mode")}</label>
    <Select id="model-proxy-mode" value={mode} disabled={pending} onChange={(event) => { setMode(event.target.value as AIModelProxySettings["mode"]); setResult(null); }}>
      <option value="inherit">{t("ai.proxy.inherit")}</option><option value="direct">{t("ai.proxy.direct")}</option><option value="custom">{t("ai.proxy.custom")}</option>
    </Select>
    {mode === "custom" && <FormField label={t("ai.proxy.url")} value={url} maxLength={2048} required disabled={pending} placeholder="http://192.168.2.231:7890" autoComplete="off" spellCheck={false}
      description={t("ai.proxy.urlHelp")} onChange={(event) => { setURL(event.target.value); setResult(null); }} />}
    <p>{t("ai.proxy.scope")}</p>
    <div className={styles.actions}>
      <Button type="button" disabled={pending || !ready} onClick={() => { setResult(null); void probe.mutateAsync({ body, csrfToken }).then((v) => setResult(v.available ? "available" : "unavailable")).catch(() => setResult("failed")); }}>{t(probe.isPending ? "ai.proxy.testing" : "ai.proxy.test")}</Button>
      <Button type="submit" variant="primary" disabled={pending || !ready || !dirty}>{t(save.isPending ? "ai.proxy.saving" : "ai.proxy.save")}</Button>
    </div>
    {result && <InlineStatus tone={result === "available" ? "success" : "warning"}>{t(`ai.proxy.${result}`)}</InlineStatus>}
    {save.isError && <ErrorState message={t("ai.proxy.saveFailed")} onRetry={refresh} />}
  </form>;
}
