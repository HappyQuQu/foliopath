# CR-2026-005：媒体库自动发现

## 状态

- 状态：Confirmed
- 变更等级：C3（用户可见能力 + 文件系统机制 + 增量任务一致性）
- 目标版本：`POST-MVP-2` / `Post-MVP/2`
- Scope revision / 范围状态：[POST-MVP-2 revision 2](../releases/POST-MVP-2-scope-r2.md)
  已冻结；不修改 `MVP-2026-07-23` 或 `POST-MVP-1`
- Change Record ID / 输入事件：CR-2026-005 / 2026-07-29 用户要求自动发现、方案文档并继续推进
- 提出日期：2026-07-29
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- Capability Owner：`internal/scanner`

## 用户问题与价值

- 用户 / JTBD：当我把文件或文件夹放进已有媒体库时，希望后台很快完成索引；进入目录或
  点击页面刷新即可看到新内容，不等待最长 24 小时，也不手动发起完整扫描。
- 当前行为：创建、启动、手动和默认 24 小时完整扫描可最终收敛，但没有近实时快速路径。
- 目标结果：在支持的本地 Linux 文件系统上，变化通常数秒内进入索引和当前页面；监听不可靠时
  明确降级并由完整扫描兜底。
- 为什么不进入 MVP：当前 MVP scope 与 RC 功能已冻结，`MVP-NG-008` 明确把 watcher 作为
  未来优化；该能力默认进入后续版本。

## 范围

- 提案 FR/NFR：`FR-SCN-010～014`、`NFR-REL-002`、`NFR-PERF-003`。
- 提案验收：`WCH-AC-001～012`。
- 明确包含：Linux 目录 watch、事件合并、定向校准、新目录递归纳管、durable 有界任务、
  overflow/offline 降级、内容 revision、目录导航重取和页面刷新按钮。
- 明确不包含：依赖 watcher 保证正确性、网络盘近实时承诺、全树轮询、原媒体写入、新服务、
  Redis/WebSocket，以及单个删除事件直接清理索引。
- 被替代/延期的现有范围：N/A。
- Scope-budget exception：N/A；不加入冻结 MVP 或 `POST-MVP-1`。

完整方案见
[FTR-SCN-001 媒体库自动发现](../features/automatic-library-discovery.md)。

## 架构影响

- Capability 与依赖方向：`internal/scanner` 拥有事件到定向校准的语义；`internal/files`
  实现 Linux watch 与安全重开；`internal/jobs` 继续拥有 durable lease worker；
  `internal/store/sqlite` 实现队列和 content revision；`internal/app` 组装生命周期。
- API / 用户流程：Settings 和 Library 增加自动发现开关/状态/revision；浏览/搜索页面在
  目录导航或用户点击刷新时由 TanStack Query owner 重建当前 cursor 链，不持续轮询。
- 数据 / migration / 派生状态：预计新增或扩展 durable incremental queue，并增加独立
  `content_revision`；只追加 migration，精确方案由 Contract Ready 冻结。
- 安全、隐私与信任边界：监听路径是不可信提示，实际确认仍经过 pathpolicy 与 Linux
  `openat2` 锚定边界；API/日志不暴露绝对路径。
- 性能、容量与并发：每目录而非每文件 watch；dirty 集合、队列、批次、并发和前端刷新全部
  有界；10 万媒体／1 万目录与 burst/overflow 必须实测。
- 部署、升级、恢复与观测：无新部署单元；可能要求记录宿主 `inotify` 限制。watch 不可用时
  应用继续运行并降级到完整扫描。
- ADR：实现前必须新增并接受 watcher 生命周期、增量清理资格和 durable queue 语义 ADR。

## 质量属性场景

- 刺激：本地媒体库新增文件、空目录或发生原子移动。
- 环境：扫描和媒体任务可能同时运行。
- 系统响应：短暂合并后安全定向校准并提交 content revision；用户导航或点击刷新时重新获取
  当前范围。
- 可测结果：索引 `P95 <= 10s`；用户操作后的页面响应沿用现有浏览查询预算，不产生后台
  revision 轮询。

- 刺激：媒体根掉线、事件 overflow、ENOSPC 或 watcher 错误。
- 环境：事件可能重复、乱序或批量表现为删除。
- 系统响应：停止删除性校准，保留可靠索引，标记 degraded/offline，合并安排完整扫描。
- 可测结果：恢复和成功完整扫描前，不可靠缺失不得清理；队列、RSS 和 goroutine 保持有界。

## 风险与验证

- 新风险：`R-019`，事件丢失/溢出/网络盘/watch 上限或误判删除。
- 继承风险：R-002 路径逃逸、R-003 离线误清理、R-005 容量、R-013 跨库饥饿、
  R-016 验证漂移。
- Fallback：先扩大合并与降低定向并发，再按库 degraded，最终全局禁用自动发现并保留完整扫描。
- Fixture：新增/修改/删除/rename、慢写、空目录、深目录、symlink、nested mount、
  root replacement、unmount、权限错误、overflow、ENOSPC、强杀与 100k/10k burst。
- 证据：unit/race、Linux openat2/inotify integration、真实 SQLite、HTTP contract、
  容器恢复、双架构容量和浏览器 E2E。

## Gate 影响与决定

- 新切片：`WCH-S0 Architecture Ready → WCH-S1 Contract Ready → WCH-S2 Backend
  Evidence Ready → WCH-S3 Consumer/UI Ready → WCH-S4 Integrated Slice Done`。
- 产品决定：Confirmed；`POST-MVP-2` revision 2 保留自动近实时索引、默认开启、安全降级
  和完整扫描兜底，把打开页面的消费方式改为目录导航重取和手动刷新。
- 架构决定：[WCH-S0](../gates/POST-MVP-2/wch-s0-architecture-ready.md)已 Go；
  WCH-001 Linux/arm64 spike 通过，ADR-0011 已接受。
- API/数据决定：[WCH-S1 Contract Ready](../gates/POST-MVP-2/wch-s1-contract-ready.md)
  已 Go；OpenAPI、migration 12、稳定错误码、content revision、任务水位与资源上限已冻结。
- 当前结论：允许 WCH-S2 后端实现与证据；产品 UI 仍须等待 Backend Evidence Ready。
