# CR-2026-004：视频故事板悬停预览

## 状态

- 状态：Confirmed
- 变更等级：C2
- 目标版本：`POST-MVP-1` / `Post-MVP/1`
- Scope revision / 范围状态：[POST-MVP-1 revision 1](../releases/POST-MVP-1-scope.md)
  已冻结；不修改 `MVP-2026-07-23`
- Change Record ID / 基线事件：CR-2026-004 / 2026-07-29 产品用户确认
- 提出日期：2026-07-29
- 产品负责人：产品用户（本次 feature 确认）
- 架构负责人：FolioPath maintainers
- Capability Owner：`internal/thumbnail`
- 交互 Owner：共享 `web/src/components/patterns/MediaCollection`

## 用户问题与价值

- 用户 / JTBD：在大型视频集合中不打开播放器即可快速判断视频跨时间段的内容。
- 当前问题及证据：单张 poster 只代表一个时间点；文件名和 poster 不足以区分相似视频。
- 为什么进入目标版本：故事板能缩短视频筛选路径，但不是稳定 MVP 发布的必要条件，因此明确
  进入后续版本，不打断当前 RC 加固。

## 范围

- 新增 FR/NFR：`FR-MED-009～011`、`FR-UI-008`；验收 `VSP-AC-001～008`。
- 明确包含：支持视频的 4～10 帧均匀采样、WebP sprite、低优先级 durable job、统一缓存、
  桌面 hover、reduced-motion/触摸降级、浏览与搜索共享实现。
- 明确不包含：真实关键帧/场景检测、AI、视频转码、移动端手势、音频、用户可配置参数、新服务。
- 被替代/延期的现有范围：N/A；现有 poster、预览和播放继续保留。
- Scope-budget exception：N/A；不加入冻结 MVP。

完整产品和技术合同见
[FTR-VID-001 feature spec](../features/video-storyboard-preview.md)。

## 架构影响

- Capability 与依赖方向：`internal/thumbnail` 拥有派生/缓存规则，`internal/media` 实现有界
  抽帧，`internal/jobs` 拥有调度，`internal/files` 安全打开，API 只调用服务；共享
  `MediaCollection` 唯一拥有卡片 hover。
- API / 用户流程：计划扩展 authenticated thumbnail variant 和 Asset 派生状态；前端意图成立
  后才加载 sprite。
- 数据 / migration / 派生状态：新增只向前 migration，放宽 migration 8/9 的 grid-only
  CHECK，并保存 storyboard layout；历史 migration 不修改。
- 安全、隐私与信任边界：不新增边界；继续使用 `/library` 锚定只读打开、认证与错误脱敏。
- 性能、容量与并发：storyboard 低于 grid/poster；backfill、FFmpeg、缓存和前端活动动画均有界。
- 部署、升级、备份、恢复与观测：无新部署单元；缓存可重建，升级需验证历史任务/缩略图表。
- 平台、依赖、许可证与供应链：复用现有 FFmpeg/WebP runtime；双架构以同 fixture 复验。
- ADR：当前预期 N/A，因为不改变部署、核心技术、信任/持久化边界或所有权方向；如果实现需要
  改变任务一致性、独立 worker 或新媒体技术，必须先新增 ADR。

## 质量属性场景

- 刺激：扫描后大量视频需要生成故事板，同时用户继续浏览和搜索。
- 环境：四核/4 GiB 目标环境、多个媒体库、grid/poster 与 storyboard 同时排队。
- 系统响应：poster 优先可用；storyboard 有界低优先级处理；API 保持可用；失败只降级到 poster。
- 可测结果：grid 等待、跨库公平、RSS/队列/缓存、浏览 P95 和取消延迟满足 Contract Ready
  spike 固定的预算。

- 刺激：用户快速移动鼠标经过多个虚拟化视频卡片。
- 环境：桌面 fine pointer、浏览或搜索长列表。
- 系统响应：300ms 前不请求；同页最多一个活动动画；卡片回收后清理 timer/资源并恢复 poster。
- 可测结果：请求、timer、DOM、observer 和内存不随经过卡片数无界增长。

## 风险与验证

- 风险：沿用 R-006、R-009、R-013、R-015、R-016，并新增 R-018（storyboard backfill/hover
  造成资源与交互退化）。
- Fallback：依次降低 worker/尺寸/质量、改为有界按需 admission，最后禁用 storyboard 并保留 poster。
- 正常、边界、失败、恢复：短/长/VFR/长 GOP/损坏视频、部分 seek、源变化、offline、取消、
  重启、ENOSPC、cache missing、快速 hover、虚拟回收、touch/reduced-motion。
- Fixture 与目标环境：合成许可视频；Darwin 开发证据与原生 Linux amd64/arm64；目标三浏览器。
- 验收证据：unit、migration upgrade、真实 FFmpeg、SQLite/files/HTTP integration、容量、
  Storybook/axe/visual/E2E 和原媒体 hash/mtime 不变。

## Gate 影响与决定

- 新切片：`VSP-S0` Architecture Ready → `VSP-S1` Contract Ready → `VSP-S2` Backend
  Evidence Ready → `VSP-S3` Consumer/UI Ready → `VSP-S4` Integrated Slice Done。
- 产品决定：Confirmed；最多 10 帧、均匀采样、sprite、桌面 hover、失败回退 poster。
- 架构决定：[VSP-S0 Architecture Ready](../gates/POST-MVP-1/vsp-s0-architecture-ready.md)
  已 Go；现有模块边界可承载，精确 OpenAPI、migration 和资源预算进入 S1。
- 安全/数据评审：[VSP-S1 Contract Ready](../gates/POST-MVP-1/vsp-s1-contract-ready.md)
  已固定只追加 migration、源指纹 CAS、原件不变、失败关闭、资源与调度合同。
- 最终结论：Contract Ready Go；允许执行 `VSP-106～113` 后端实现与证据。
  S2 Go 前不允许实现产品业务 UI。
