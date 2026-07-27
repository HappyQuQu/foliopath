# S2-004 媒体库生命周期实现记录

## 结论

**Go — S2-004 实现完成；进入 S2-005。**

本记录只确认媒体库生命周期切片已按冻结契约实现，不把媒体库后端标记为 Backend Ready，
也不表示扫描 worker、索引、缩略图或前端集成已经完成。媒体库 Backend Ready 仍由
`S2-005～S2-007` 决定，扫描执行仍由 `S2-102～S2-107` 决定。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-LIB-001～008`、`FR-SCN-001`、`NFR-SAFE-001`、`NFR-SEC-001～002`
- Contract Ready：
  [媒体库](s2-library-contract-ready.md)与[可靠扫描](s2-scan-contract-ready.md)
- 用例 owner：`internal/library`；扫描 admission owner：`internal/scanner`
- 文件系统 adapter：`internal/files`
- SQLite adapter：`internal/store/sqlite`
- HTTP/composition：`internal/api`、`internal/app`

## 已实现行为

- `POST /api/v1/libraries` 严格接收名称、allowed-root-relative 根和幂等键。服务先通过
  anchored `internal/files` 重新打开并验证真实目录，再由一个 SQLite immediate 短事务
  原子写入 library、唯一 `library_created` queued scan 和摘要化 idempotency record。
- 同 key/同规范请求重放原结果且跳过易变文件系统复检；同 key/不同请求或结果已移除时稳定
  冲突。提交前不唤醒，提交后只发送容量为 1 的进程内 hint；durable row 才是事实来源。
- 列表使用持久化 Unicode numeric natural key、显示名和稳定 ID 的有界 keyset 页面；
  cursor 加密并具备完整性保护。详情只返回相对根和服务端构造的 `/library/...` 标签。
- PATCH 只允许改名并要求强 If-Match；过期 validator 为 412，无变化改名不推进 revision，
  active removal 阻止改名。扫描 admission/finalize/失败造成的公开 Library 变化推进 revision。
- offline library 可以通过 `POST /api/v1/libraries/{id}/scans` 进入同一 durable manual
  admission；同库 active scan 原子 coalesce，全局 256 active 上限和 active removal
  拒绝规则已实施。S2-102 worker 将消费同一 `scan_runs` 和 bounded wake signal。
- DELETE 要求 If-Match 与幂等键，原子创建 durable removal、取消 queued scan 或请求
  running scan 协作取消。单 worker 等待 scan terminal 后，重复执行有界、幂等 SQLite
  批次及 `/app/data/cache/libraries/lib_<id>` 清理，最后删除配置并保留 terminal removal
  和 idempotency 结果供轮询。

## 安全边界

- 生命周期 API 从不接收宿主机路径；公开响应不返回宿主机路径、SQL、errno 或堆栈。
- 根目录复检只通过 kernel-anchored `internal/files`，并稳定区分 unavailable、symlink、
  mount boundary 和 outside allowed root。
- removal worker 只持有 SQLite repository 与 application-data cache cleaner；没有
  `/library` handle、文件打开、移动、改名、写入或删除能力。
- migration 5 只追加 `libraries.name_sort_key` 及其索引；旧库在 Store 启动时有界事务回填，
  已发布 migration 未被改写。

## 自动证据

- capability 单元测试覆盖根验证先于事务、提交后唤醒、重放跳过文件系统、加密 cursor、
  新 admission 与 coalesce 的 wake 差异。
- SQLite 测试覆盖创建三记录 commit/replay/conflict、revision 改名、离线 retry、
  active removal 拒绝、自然 keyset 排序、scan cancellation、分批 cleanup 及 terminal
  removal/idempotency 保留。
- HTTP 测试覆盖创建 Location/ETag/安全 display path、严格 strong If-Match，以及新 scan
  `202`/coalesced `200` 的 Location/ETag。
- 完整仓库验证结果在本实现 PR 中记录；S2-005、S2-006 仍负责更深的路径故障矩阵与原媒体
  逐字节不变证明，不能由本记录提前宣称。

## 下一步

1. `S2-005`：补 traversal、symlink、nested mount、TOCTOU、重叠、离线和权限失败矩阵。
2. `S2-006`：证明移除前后 synthetic fixture 原文件逐字节不变。
3. `S2-007`：审计并记录媒体库 Backend Ready。
4. `S2-102`：让正式有界扫描 worker 消费当前 durable admission 与 wake signal。
