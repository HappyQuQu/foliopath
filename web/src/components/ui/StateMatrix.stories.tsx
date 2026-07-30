import type { Meta, StoryObj } from "@storybook/react-vite";
import { MagnifyingGlass } from "@phosphor-icons/react";
import type { ReactNode } from "react";

import { useLocale } from "../../lib/i18n/LocaleProvider";
import {
  EmptyState,
  ErrorState,
  LoadingState,
  OfflineState,
} from "./AsyncState/AsyncState";
import { Button } from "./Button/Button";
import { InlineStatus } from "./InlineStatus/InlineStatus";
import styles from "./StateMatrix.stories.module.css";

const meta = {
  title: "UI/StateMatrix",
  parameters: { layout: "fullscreen" },
  tags: ["autodocs"],
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

export const Complete: Story = {
  render: () => <StateMatrix />,
};

function StateMatrix() {
  const { locale, t } = useLocale();
  const query = locale === "zh-CN" ? "京都" : "Kyoto";

  return (
    <div className={styles.matrix}>
      <StateCell title="Loading">
        <LoadingState label={t("search.loading")} />
      </StateCell>
      <StateCell title="Empty">
        <EmptyState
          action={<Button>{t("search.clearFilters")}</Button>}
          description={t("search.emptyDescription").replace("{query}", query)}
          icon={MagnifyingGlass}
          title={t("search.emptyTitle")}
        />
      </StateCell>
      <StateCell title="Offline">
        <OfflineState
          action={<Button>{t("common.retry")}</Button>}
          description={t("search.offlineDescription")}
          title={t("search.offlineTitle")}
        />
      </StateCell>
      <StateCell title="Error">
        <ErrorState message={t("search.failed")} onRetry={() => undefined} />
      </StateCell>
      <StateCell title="Conflict">
        <InlineStatus tone="danger">
          {t("settings.changedElsewhere")}
        </InlineStatus>
      </StateCell>
      <StateCell title="Cancel">
        <InlineStatus>
          {t("scan.descriptionCancelled")} {t("scan.indexPreserved")}
        </InlineStatus>
      </StateCell>
      <StateCell title="Pending">
        <Button loading variant="primary">
          {t("common.loading")}
        </Button>
      </StateCell>
      <StateCell title="Success">
        <InlineStatus tone="success">
          {t("scan.descriptionSucceeded")}
        </InlineStatus>
      </StateCell>
    </div>
  );
}

function StateCell({
  children,
  title,
}: {
  children: ReactNode;
  title: string;
}) {
  return (
    <section className={styles.cell}>
      <h2>{title}</h2>
      {children}
    </section>
  );
}
