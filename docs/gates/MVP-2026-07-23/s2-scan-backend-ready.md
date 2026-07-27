# S2-107 可靠扫描 Backend Ready

## 结论

**Go — 可靠扫描后端达到 Backend Ready。**

`S2-101～107` 已完成冻结扫描契约、durable admission、有界 worker、目录/媒体增量索引、
generation 一致性、故障恢复、容量并发，以及全部扫描观察/取消/设置 HTTP operation。
前端现在可以通过生成 client 接入扫描历史、详情轮询、手动请求、协作取消和计划设置。

本结论不授权 Stage 3 浏览/缩略图后端，不把前端扫描流程标记为完成，也不代表可发布版本。
Integrated Done 仍要求前端真实 API 集成和浏览器 E2E；正式强杀、代表性存储与发布镜像仍由
Release Gate 验证。

## 范围与权威来源

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-SCN-001～009`、`FR-LIB-006～008`、`FR-DEP-003`、
  `NFR-SAFE-001`、`NFR-SEC-001～002`、`NFR-REL-001`、
  `NFR-PERF-001～002`、`NFR-OPS-001`
- HTTP 权威：`api/openapi.yaml`
- generation、查询游标、取消和 scheduler owner：`internal/scanner`
- durable queue/worker owner：`internal/jobs`
- HTTP/DTO/稳定消息 owner：`internal/api`
- SQLite adapter：`internal/store/sqlite`
- 正式组合：`internal/app`
- 前序 Gate：S2-101 Contract Ready、S2-102～106 实现记录

## 已交付契约

- `POST /api/v1/libraries/{libraryId}/scans`：新建返回 202，active coalesce 返回 200。
- `GET /api/v1/libraries/{libraryId}/scans`：默认 50/最大 200，按
  `created_at_ms DESC, id DESC` keyset；加密 cursor 绑定 library，篡改或跨库复用失败。
- `GET /api/v1/scans/{scanId}`：返回稳定状态、计数和最多 50 个安全聚合 issue；
  强 ETag 支持 `If-None-Match`/304。
- `POST /api/v1/scans/{scanId}/cancel`：queued 原子终止；running 只记录一次取消请求；
  pending 重复请求不推进 revision，terminal 返回 `scan_already_finished`。
- `GET/PATCH /api/v1/settings`：强 ETag/If-Match；计划间隔 1～8760 小时，null 关闭；
  同一设置资源继续承载缓存配额与语言字段。
- scheduler 默认每 24 小时按 library ID 有界分页，只经同一 durable admission 创建
  `scheduled` scan；设置变更只发送 wake hint，正确性不依赖进程内信号。

全部业务路由仍由统一 session、同源、CSRF、限流、防缓存、request ID 与安全错误中间件保护。
API 不暴露绝对路径、SQL、errno、stderr 或 stack。

## 一致性、故障与资源证据

- 同库唯一 active scan；全局 256 active、2 worker、256 catalog batch、容量 1 wake signal。
- creation/startup/manual/scheduled 共用 `scan_runs`，不存在第二内存队列或 feature-local worker。
- cancelled、offline、failed、interrupted 和 SQLite 满页均保留最后可靠 generation；
  只有成功 finalize 清理 stale row。
- 根 symlink、nested mount、root identity 改变、部分不可读、扫描 I/O 与数据库不可用均记录
  契约内稳定错误码。
- 10k 目录/100k 资产与 1,000 层 rollup 通过强制预算；Linux amd64/arm64 容量 CI 通过。
- 设置、scheduler、历史/详情/cancel 分别具备 capability、SQLite、HTTP、真实 composition
  和 architecture fitness 证据。

## 验证入口

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-race
make test-integration
make test-e2e
make spike-capacity
```

## 前端交接

允许：

- 生成 client adapter 消费上述 6 个扫描/设置 operation；
- 以 URL/Query 状态展示历史，以 ETag 和退避轮询详情；
- 展示稳定 code 对应的本地化文案，发起手动扫描和协作取消；
- 设置或关闭定时扫描。

仍禁止：

- 前端复制 query key、cursor、ETag、错误映射或扫描状态机；
- 把 wake signal、queued 状态或估算目录数显示成确定进度；
- 依赖尚未 Backend Ready 的目录浏览、缩略图、搜索或查看器接口；
- 声称扫描前端已完成、Integrated Done、Stage 3 已完成或 MVP 可发布。

- 评审日期：2026-07-27
