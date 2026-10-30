import { Button } from "../../components/ui";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import type { useDeletionFlow } from "./useDeletionFlow";

export function AssetDeletionAction({ flow, asset, onAccepted }: { flow: ReturnType<typeof useDeletionFlow>; asset: { id: string; archiveMember?: boolean }; onAccepted: () => void }) {
  const { t } = useLocale();
  if (asset.archiveMember) return null;
  return <Button variant="danger" disabled={!flow.canDelete} onClick={() => flow.start([{ type: "asset", id: asset.id }], onAccepted)}>{t("deletion.delete")}</Button>;
}
