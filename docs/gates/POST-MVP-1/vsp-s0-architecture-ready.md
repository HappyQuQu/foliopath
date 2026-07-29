# Gate POST-MVP-1 / VSP-S0 / Architecture Ready

- 日期：2026-07-29
- 目标版本：`POST-MVP-1`
- Roadmap 切片：`VSP` 视频故事板悬停预览
- 需求：`FR-MED-009～011`、`FR-UI-008`
- 验收：`VSP-AC-001～008`
- 决策角色：产品负责人（产品用户）；架构/capability/安全与数据负责人（FolioPath maintainers）
- Change Record：[CR-2026-004](../../changes/CR-2026-004-video-storyboard-preview.md)
- Scope：[POST-MVP-1 revision 1](../../releases/POST-MVP-1-scope.md)
- Feature：[FTR-VID-001](../../features/video-storyboard-preview.md)
- Spike：[VSP-002](../../spikes/vsp-002-video-storyboard.md)
- 风险：R-006、R-009、R-013、R-015、R-016、R-018
- 结论：**Go**

## 输入完整性

| S0 输入 | 结论 |
| --- | --- |
| 目标版本与稳定需求 | `POST-MVP-1` revision 1 已冻结；不改变当前 MVP/RC |
| 用户结果与非目标 | feature spec 和 scope manifest 已固定 |
| Capability owner | `internal/thumbnail`；adapter/consumer 边界明确 |
| 数据与 API 影响 | 只追加 migration 和 OpenAPI 扩展；精确合同归 S1 |
| 正常/边界/失败/恢复 | `VSP-AC-001～008`、feature failure matrix 已定义 |
| Fixture 与可行性 | VSP-002 合成 2s～2h fast-seek/sprite spike 通过 |
| 资源与 fallback | R-018 已登记；低优先级、统一 LRU、按需/禁用 fallback 已固定 |
| ADR 判断 | 无新部署、技术、信任或持久化边界；当前不需要新 ADR |

## 架构决定

1. 复用模块化单体、现有媒体 worker、SQLite durable jobs 和统一 cache；不新增服务或队列。
2. `internal/thumbnail` 唯一拥有 `storyboard` variant、采样、派生、发布和 LRU。
3. `internal/media` adapter 实现最多 10 次有界 input seek 与一次 sprite 拼接，不拥有任务、
   cache、HTTP 或 UI 语义。
4. `internal/jobs` 唯一拥有 grid/poster 高于 storyboard 的调度与跨库公平。
5. `internal/files` 继续提供 Linux 锚定只读 FD；handler 不访问路径、SQLite 或 FFmpeg。
6. 共享 `MediaCollection` 唯一拥有 hover intent/动画生命周期；必须等待 S2 Backend Ready。
7. 首版 storyboard all-or-nothing；任意抽帧/拼接失败只回退 poster。

## 未关闭风险与 fallback

- 本地 spike 是低分辨率 Darwin/arm64 证据；目标 FFmpeg/libwebp、真实编码和 Linux 双架构
  留给 S1/S2。
- 45 秒总 deadline、1 MiB/frame、10 MiB temp、8 MiB final 是 S1 候选上限，不是已验证
  生产预算。
- 如果 S2 容量不能保证 grid/API，依次降低 storyboard admission/尺寸/质量，改为有界按需，
  最终禁用 storyboard 并保留 poster。
- 任何需要改变任务一致性/所有权、独立 worker 或新媒体技术的方案使本 Gate 失效并要求 ADR。

## 下一步获准范围

允许执行 `VSP-101～105`：

- 固定 capability use case、all-or-nothing failure 和资源上限；
- 修改并评审权威 OpenAPI 源；
- 设计只追加 migration、优先级 claim 和 bounded backfill；
- 建立 Contract Ready 自动测试/fixture 计划；
- 记录 `VSP-S1 Contract Ready`。

本 Gate 不授权生产 handler、migration 执行、worker 接入或前端 UI。只有 S1 Go 后才能开始
后端实现，只有 S2 Go 后才能开始前端真实接入。
