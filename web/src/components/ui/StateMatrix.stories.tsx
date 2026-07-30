import type { Meta, StoryObj } from "@storybook/react-vite";
import { MagnifyingGlass } from "@phosphor-icons/react";
import type { ReactNode } from "react";

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
  render: () => (
    <div className={styles.matrix}>
      <StateCell title="Loading">
        <LoadingState label="正在载入媒体…" />
      </StateCell>
      <StateCell title="Empty">
        <EmptyState
          action={<Button>清除筛选</Button>}
          description="没有符合当前关键字和筛选条件的媒体。"
          icon={MagnifyingGlass}
          title="没有搜索结果"
        />
      </StateCell>
      <StateCell title="Offline">
        <OfflineState
          action={<Button>重试连接</Button>}
          description="可靠索引与缓存仍然保留；恢复媒体源后可以重新扫描。"
          title="媒体库当前离线"
        />
      </StateCell>
      <StateCell title="Error">
        <ErrorState
          message="FolioPath 暂时无法响应。当前界面与原始媒体均未改变。"
          onRetry={() => undefined}
        />
      </StateCell>
      <StateCell title="Conflict">
        <InlineStatus tone="danger">
          此媒体库已在其他窗口中更新。刷新后再试。
        </InlineStatus>
      </StateCell>
      <StateCell title="Cancel">
        <InlineStatus>
          扫描已取消；上一次可靠索引与已安全提交的新增内容仍然保留。
        </InlineStatus>
      </StateCell>
      <StateCell title="Pending">
        <Button loading variant="primary">
          正在保存
        </Button>
      </StateCell>
      <StateCell title="Success">
        <InlineStatus tone="success">
          扫描已完成，可靠索引已经更新。
        </InlineStatus>
      </StateCell>
    </div>
  ),
};

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
