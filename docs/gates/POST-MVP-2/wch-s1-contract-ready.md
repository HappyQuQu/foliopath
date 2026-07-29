# WCH-S1 媒体库自动发现 Contract Ready

## 结论

**Go — WCH-S1 Contract Ready。**

`FTR-SCN-001` 的 capability、HTTP、持久化、任务互斥、资源与失败合同已经冻结，可以实现
WCH-S2 生产后端和证据。本 Gate 不授权产品前端；前端必须等待
`WCH-S2 Backend Evidence Ready`。

## 基线与所有权

- 目标版本：[POST-MVP-2 revision 1](../../releases/POST-MVP-2-scope.md)
- Change Record：[CR-2026-005](../../changes/CR-2026-005-automatic-library-discovery.md)
- Feature：[FTR-SCN-001](../../features/automatic-library-discovery.md)
- 前序 Gate：[WCH-S0 Architecture Ready](wch-s0-architecture-ready.md)
- ADR：[ADR-0011](../../adr/0011-linux-inotify-hints-and-anchored-reconciliation.md)
- 权威 HTTP 合同：[`api/openapi.yaml`](../../../api/openapi.yaml)
- Capability Owner：`internal/scanner`
- 文件系统 adapter：`internal/files`
- durable execution owner：`internal/jobs`
- 持久化 adapter：`internal/store/sqlite`

## Capability 与一致性合同

- Linux 目录事件只标记 dirty 范围；事件本身没有存在、删除或路径安全证明力。
- 定向校准单位是一个规范的媒体库相对目录，安全枚举它的直接子项；新目录通过有界任务继续
  扩展，空目录也建立索引。
- 删除只在库根身份未变、父目录完整安全枚举成功且目标确实缺失时提交。任何不确定状态保留
  旧索引并 degraded/offline。
- 完整 scan 和定向 reconcile 按媒体库互斥。首版不吸收 queued reconcile；完整扫描后仍
  幂等执行，避免漏掉扫描期间的事件。
- `catalog_reconcile_jobs` 使用 `requested_revision/claimed_revision` 水位。运行中新增事件
  递增 requested；旧 claim 只能在水位未变化时删除任务，否则重新排队。
- 只有完整成功 generation 可以执行库级 stale cleanup；定向任务永远没有该资格。

## HTTP 合同

权威 OpenAPI 与生成 TypeScript client 已追加：

- Settings 的 `automaticDiscoveryEnabled`，默认 true，继续使用强 ETag/If-Match；
- Library 的 `automaticDiscoveryStatus`、`automaticDiscoveryErrorCode`、
  `lastAutomaticDiscoveryAt` 和 `contentRevision`；
- 认证 `GET /api/v1/catalog/state`，支持强 ETag、`no-store`、If-None-Match 与 304；
- 稳定错误码限定为 `watch_unavailable`、`watch_resource_limit`、`watch_overflow`、
  `source_unavailable`、`internal_error`。

content revision 只通知客户端重新获取，不参与 generation 或 cursor 校验。新增必需响应字段
是 `POST-MVP-2` 客户端与服务端协同发布的版本化扩展；旧独立客户端兼容不在本 Gate 承诺内。

## 只向前 migration 设计

实现必须新增 migration 12，不得修改 migration 1～11：

- settings 增加默认 1、CHECK 0/1 的 `automatic_discovery_enabled`；
- libraries 增加受 CHECK 约束的状态/错误、可空成功时间和正整数 `content_revision`；
- `catalog_search_state` 增加独立正整数 `content_revision`，不改变原 search cursor revision；
- 新增 `catalog_reconcile_jobs`，以 `(library_id, relative_dir_path)` 唯一，保存
  queued/running/failed、requested/claimed 水位、debounce、lease、attempt 与稳定错误；
- 媒体库删除级联删除任务；路径只保存 library-relative 值；
- library create/delete、full generation publish 与成功定向提交在相应短事务推进通知
  revision；文件系统 I/O 期间不持事务。

任务最多尝试 5 次，退避 1/2/4/8/16 秒。失败耗尽、overflow、资源不足或可靠枚举失败必须
保留索引、标记 degraded 并合并完整扫描。

## 资源合同

- 每实例最多 32,768 个目录 watch；
- 进程内未合并事件最多 8,192，dirty 目录最多 4,096；
- 定向执行全局最多 2 个，每库最多 1 个；
- debounce 750ms，持续变化最迟 5s admission；
- 直接目录枚举每批最多 2,048 entries；
- 达到任一上限时禁止无界扩容，受影响库 degraded 并合并完整扫描。

WCH-S2 可以用证据下调上限；提高上限必须重新评审资源预算。

## 安全与失败合同

- inotify 名称/cookie/顺序不可信；所有路径通过 `pathpolicy` 后再由 Linux 锚定
  `openat2` 边界重开。
- symlink、nested mount、root replacement、unmount、overflow 和 `ENOSPC` 均失败关闭。
- API、SQLite 和日志不保存/返回宿主路径、绝对媒体路径、errno、内核文本或 stack。
- watcher unsupported/disabled/degraded 不关闭创建、启动、手动或定时完整扫描。
- 不新增部署单元、WebSocket、Redis、外部消息队列或原媒体写能力。

## Contract Ready 证据

2026-07-29 实际执行：

- `make fmt-check`：通过；
- `make openapi-lint`：通过，只有既存的两个 health operation 4xx warning；
- `make contract-check`：通过；
- `make generate-check`：通过；
- `make arch-check`：通过。

OpenAPI 摘要锁已在兼容性评审后更新；生成 TypeScript schema 与权威源一致。

## WCH-S2 必需证据

- fresh/migration 11→12、CHECK/FK/唯一键、rollback/fail-closed；
- durable upsert/claim/watermark/lease/retry/restart 与 full-scan 按库互斥；
- create/close-write/move/rename/delete/empty-dir/slow-write；
- overflow、watch ENOSPC、symlink、nested mount、unmount、root replacement 与强杀恢复；
- 原生 linux/amd64 和 linux/arm64；
- 10k directories、100k media、100k burst 的资源、延迟、SQLite 写放大和跨库公平；
- 认证 catalog state 的 200/304/401/429/500 与 revision/cursor 独立性。

## 授权边界

本 Gate 只授权 migration 12、生产 watcher/定向校准后端、API adapter、composition、自动测试和
`WCH-S2 Backend Evidence Ready`。S2 Go 前不得实现生产自动刷新 UI、把功能描述为可用，或
用 mock 绕过后端合同。
