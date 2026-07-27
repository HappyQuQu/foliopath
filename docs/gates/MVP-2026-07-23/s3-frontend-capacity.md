# S3-107 前端容量预算完成记录

## 结论

**Done — S3-107 已用 100,000 媒体主档固定浏览 DOM、cursor 请求、虚拟滚动、播放资源与焦点恢复预算。**

本记录不把 Stage 3 标记为 Integrated Done；完整核心浏览 Gate 仍属于 S3-108。低性能
NAS 客户端的最终 FPS、RSS 和目标浏览器矩阵仍属于 Stage 5 发布验收。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 3 / `S3-107`
- 需求：`FR-BRW-004～007`、`FR-MED-004～006`、`FR-UI-001～004`、
  `NFR-ACC-001`、`NFR-PERF-001～002`
- 后端 Gate：[S3-007 Backend Ready](s3-browse-thumbnail-backend-ready.md)
- 集合与虚拟焦点 owner：`web/src/components/patterns/MediaCollection`
- cursor / pending 刷新 owner：`web/src/features/browse/queries.ts`
- 活动媒体 owner：`web/src/components/patterns/MediaPreview`

## 固定预算

| 表面 | 主档与预算 | 结果 |
| --- | --- | --- |
| DOM | 100,000 个稳定资产；默认测试视口最多挂载 64 个 `li` | 通过；实际单测窗口低于 64 |
| 浏览器 DOM | 1280×720 Chromium，首屏与第 100,000 项 | 首屏 42 项；末端 40 项 |
| 虚拟滚动 | 从首项跳到第 100,000 项，保持语义顺序 | `scrollY=3,609,790`；挂载 `aria-posinset=99,961～100,000` |
| 焦点 | 关闭/恢复目标可暂时不在 DOM；等待必须有界 | 最多 12 个 animation frame；真实浏览器焦点落在 `预览 capacity-100000.jpg` |
| cursor 请求 | API 每页 50 项，只在距末端两行内预取 | 首屏 0 次额外预取；在途或分页错误时不重复触发 |
| pending 刷新 | 2.5 秒刷新不能随累计页数无界增长 | 最多 4 个已载入页；第 5 页起停止周期刷新 |
| 播放资源 | 任一时刻最多一个活动媒体节点 | 视频按资产 ID key；切换后旧 video 脱离 DOM，剩余 1 个 |

若用户实际遍历完整 100,000 项，50 项 cursor 页理论上是 2,000 个顺序请求；前端不提前
获取未接近的页，也不并发重复请求。该总量是完整遍历成本，不是首屏或后台请求。

## 实现变化

- `MediaCollection` 暴露集中容量常量和纯 `shouldLoadNextMediaPage` 判定，使接近末端、
  `hasNextPage` 与在途状态只有一个 owner。
- 焦点恢复在调用虚拟控制器后最多重试 12 帧；成功、超时、新恢复请求或组件卸载都会
  清理 animation frame，不留下永久轮询。
- pending thumbnail 自动刷新从“所有累计页”收紧为最多 4 页，修复 10 万项下可能周期
  重取大量历史 cursor 页的请求风暴。
- 组件工作台新增 `Capacity100k` 专用主档和“恢复最后一项焦点”控制，不进入生产路由或
  生产 import graph。

## 自动与浏览器证据

- `MediaCollection.test.tsx`：100k 主档、有界 DOM、完整 `aria-setsize`、首屏不预取、
  末端单请求判定、在途/分页错误抑制及第 100,000 项虚拟滚动调用。
- `queries.test.ts`：pending-only 刷新及 4 页上限。
- `MediaPreview.test.tsx`：video→video 切换卸载旧节点且 DOM 只剩一个 video。
- `Patterns/MediaCollection/Capacity100k`：真实 Chromium 1280×720 验证首屏/末端挂载数、
  361 万像素虚拟滚动、末端语义位置和焦点。
- S3-106 真实认证 E2E 继续覆盖关闭预览后的生产页面卡片焦点恢复；S3-108 将汇总完整
  浏览/预览链，不在本 Gate 重复宣称 Integrated Done。

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

- `S3-108`：核心浏览/预览 E2E 与 Stage 3 Integrated Done。
- `S4-006～009`：完整查看器、Range/codec/离线/删除状态和目标浏览器矩阵。
- Stage 5：代表性四核/4 GiB 发布设备上的 FPS、RSS、网络与存储预算。

- 评审日期：2026-07-28
