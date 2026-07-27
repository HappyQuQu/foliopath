# S3-104 前端浏览状态完成记录

## 结论

**Done — S3-104 已交付浏览集合的 skeleton、empty、error、offline 与 thumbnail pending/failed 状态。**

本记录不把 Stage 3 标记为 Integrated Done，也不宣称 S3-105～106 非模态预览、
S3-107 十万项容量或 S3-108 核心浏览 Gate 已完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 3 / `S3-104`
- 需求：`FR-BRW-005～007`、`FR-MED-001～003`、`FR-UI-001～004`、
  `NFR-ACC-001`、`NFR-PERF-001`
- 后端 Gate：[S3-007 Backend Ready](s3-browse-thumbnail-backend-ready.md)
- HTTP 权威：`api/openapi.yaml`
- 查询与 pending 刷新 owner：`web/src/features/browse/queries.ts`
- 集合/skeleton/分页错误 owner：`web/src/components/patterns/MediaCollection`
- empty/error/offline owner：`web/src/components/ui/AsyncState`

## 已交付行为

- 首次媒体页等待时使用 12 个 4:3 卡片骨架和两行身份占位；列密度与最终集合一致，
  reduced-motion 停止 shimmer，且只播报一次可访问 loading 名称。
- 普通空目录显示可复用 EmptyState；若直接目录为空但子目录有媒体，保留“包含子目录”
  恢复动作。离线媒体库即使资产页为空也显示 OfflineState，明确保留索引没有可显示
  媒体不等于原目录为空，且原始媒体不被修改。
- 首屏错误使用共享 ErrorState 并可重新查询；下一页错误不卸载已有卡片，只在集合尾部
  显示安全错误和局部重试。
- 资产页任一项为 `thumbnail.pending` 时按 2.5 秒刷新；全部进入 ready/failed/
  unavailable 后停止。failed/unavailable 使用稳定尺寸、图标、文字与语义色；因冻结
  API 没有“重新生成 thumbnail”写操作，前端不显示虚假单卡重试。
- 视频遮罩前景由中央 `--color-on-scrim` token 拥有，浅色/深色均保持可见。

## 证据

- `BrowsePage.test.tsx`：首屏 skeleton、offline-empty 不等于普通 empty、首屏错误恢复。
- `AsyncState.test.tsx`：empty 恢复动作、非紧急语义与 persistent offline status。
- `MediaCollection.test.tsx`：pending/failed 稳定卡片、下一页错误保留项目并局部重试。
- `queries.test.ts`：pending-only 轮询与 terminal 停止条件。
- 组件工作台：AsyncState empty/offline/error/loading；MediaCollection skeleton、
  thumbnail states、next-page failed，均可切换浅色/深色。
- `auth.spec.ts`：真实认证/ready WebP 成功链继续保留；受控冻结契约响应覆盖 skeleton、
  pending→failed、first-page error→empty 和 offline-empty，并检查 390px 无溢出及
  axe serious/critical。
- Product Design 同视口证据：
  `web/qa/s3-104-comparison-empty.jpg`、`s3-104-skeleton-light.png`、
  `s3-104-thumbnail-states-light.png`、`s3-104-thumbnail-states-dark.png` 与
  `s3-104-offline-dark.png`；完整记录在 `web/design-qa.md`。

## 验证

完成记录创建后执行：

```sh
npm --prefix web run check
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
make test-web-e2e
```

## 保留边界

- `S3-105～106`：共享非模态图片/视频预览、固定与双击规则。
- `S3-107～108`：十万媒体预算、核心浏览 E2E 和 Stage 3 Integrated Done。

- 评审日期：2026-07-28
