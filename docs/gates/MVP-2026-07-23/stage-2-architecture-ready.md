# Stage 2 媒体库与可靠扫描 Architecture Ready

## 结论

**Go — Stage 2 Architecture Ready，仅授权契约设计。**

Stage 2 被拆为两个有顺序依赖的后端切片：

1. `S2-LIB` 媒体库管理：对应 `S2-001`～`S2-007`；
2. `S2-SCN` 可靠扫描：对应 `S2-101`～`S2-107`。

本记录允许 `S2-001` 与 `S2-101` 固定 OpenAPI、数据、事务、错误和恢复契约。任何产品实现
必须等对应 Contract Ready 记录通过后再开始；本记录不表示媒体库或扫描已经可用，也不授权
Stage 3～5、业务前端集成、非回环监听或发布。

## 稳定范围与用户结果

- 目标版本：`MVP-2026-07-23`
- Scope revision：1
- Roadmap stage：Stage 2
- 需求：`FR-LIB-001～008`、`FR-SCN-001～009`、`NFR-SAFE-001`、
  `NFR-SEC-001～002`、`NFR-PRIV-001`、`NFR-REL-001`、
  `NFR-PERF-001～002`、`NFR-OPS-001`
- Change Record：`BASELINE-2026-07-23`；没有新增或改变用户可见范围

Stage 2 完成后的单一用户结果是：管理员只需把一个大媒体目录只读映射到 `/library`，
就能在设置中选择根或普通后代目录、创建多个具名且不重叠的媒体库，并观察、取消、重试
首次或后续完整扫描；离线、失败、取消或重启不得把已有可靠索引误判为空，移除媒体库不得
改变任何原始文件。

## 切片与所有权

| 切片 | 业务规则 Owner | 依赖端口 / adapter | 交付边界 |
| --- | --- | --- | --- |
| `S2-LIB` 媒体库管理 | `internal/library` | `internal/files` 根检查与安全目录枚举；`internal/store/sqlite` repository；scanner request 窄端口 | 目录选择、创建、列表、详情、改名、离线重试和移除；根创建后不可变 |
| `S2-SCN` 可靠扫描 | `internal/scanner` | `internal/files` walker；`internal/store/sqlite` scan/catalog repository；`internal/jobs` durable admission、取消与公平调度 | 首次/启动/手动/定时完整扫描、状态/历史/issues、generation、取消与恢复 |

共同边界：

- `internal/api` 只做认证后的 HTTP DTO、状态码和公开错误映射，不查询 SQLite、不解析或打开
  真实路径、不直接控制 worker。
- `internal/app` 是唯一 concrete composition root，并拥有全局队列、并发预算和生命周期接线。
- `internal/pathpolicy` 只拥有纯词法相对路径规则；`internal/files` 是所有真实媒体文件系统
  I/O 的唯一边界。
- SQLite adapter 实现 capability-owned 接口；scanner 的 Stage 0 SQL 必须在契约定型后
  迁移到 adapter 内的 sqlc query 源，不得留下第二套查询。
- 现有 feasibility/spike 代码只能作为证据和待适配输入，不能因目录已存在而视为产品实现。

## 跨切片交接约束

`S2-001` 与 `S2-101` 必须共同冻结以下交接，任一项未定则两个切片都不能进入实现：

1. 创建媒体库时，唯一名称、规范相对根、库记录和 durable 首次完整扫描请求在一个短
   SQLite 事务中保持一致；真实路径检查在事务外完成，提交后才唤醒 worker。
2. 同一媒体库最多存在一个 queued/running 完整扫描。重复请求的合并、拒绝或安全排队语义
   必须有稳定错误/响应和数据库约束，不能依赖进程内去重。
3. 媒体库离线状态由安全根检查或扫描结果驱动，但“离线”不等于“空”；重试只请求重新检查
   或完整扫描，不先清理索引。
4. 移除媒体库只清理配置、索引、durable jobs 和派生缓存。缓存清理在事务外幂等执行，
   原媒体路径不进入任何删除端口。
5. 完整扫描只有在全部遍历与批次提交成功后才能原子 finalize 和清理旧 generation；
   失败、取消、中断、offline 或部分不可读都没有清理资格。
6. 计划扫描默认 24 小时，可由 typed setting 修改或关闭；scheduler 只是 durable scan
   request 的触发器，不能拥有第二套扫描状态机。

这些约束延续既有 ADR 和模块设计，没有改变部署单元、持久化技术、信任边界或依赖方向，
因此本 Gate 不需要新增 ADR。

## 明确非目标

- Stage 3 的目录/媒体浏览、缩略图、海报、媒体元数据和缓存配额实现；
- Stage 4 的搜索、查看器、原内容/Range 和视频播放；
- 正式 React 媒体库/扫描页面、业务 UI mock 契约或前端直连未冻结端点；
- 上传、导入、移动、改名、编辑、删除或下载原始媒体；
- 原地修改媒体库根、重叠媒体库、每库一个 volume、`/library` 后代 mount 或多个媒体根；
- 跟随目录 symlink、跨文件系统扫描、watcher 正确性依赖或视频转码；
- 远程媒体源、SMB/NFS 上的 `/app/data`、多用户/权限组、匿名 LAN 或可信代理；
- 声称已达到 10 万媒体/1 万目录的完整产品或发布性能。

## 契约设计输入

### HTTP 与认证

- `api/openapi.yaml` 必须继续作为唯一 wire 契约，所有 Stage 2 端点位于 `/api/v1`，默认经过
  已完成的 session 认证；状态修改还必须经过 CSRF。
- `S2-001` 固定 server-approved 目录选择、媒体库 CRUD 的请求/响应、幂等/并发、离线与
  移除语义；API 只暴露 allowed-root-relative 值、安全容器显示值、ID 和稳定错误码。
- `S2-101` 固定扫描创建、列表/详情、issues、取消、计划设置与轮询语义；HTTP 请求只创建、
  查询或请求取消 durable 工作，不等待整库扫描完成。
- 公共错误必须使用既有统一 shape，不得泄露宿主机路径、SQL、errno、stack、Cookie、
  token 或原始文件名的无界内容。

### 数据与事务

- migration 只追加；必须固定媒体库唯一名称、规范根唯一/重叠的业务防线、scan run 状态、
  generation、取消请求、统计、有限错误摘要、schedule setting 和 durable admission。
- 目录、asset、scan 和 job 全部以 `library_id` 作用域；媒体身份保持
  `library_id + normalized relative path`。
- 事务不得覆盖目录遍历、真实路径打开、缓存清理或 worker 执行。创建库/首次扫描一致性、
  finalize 清理、移除批次和重启恢复必须各自有明确事务 owner。
- exact job 状态、唯一约束、幂等键、租约/重启恢复、批次上限和错误保留范围在 Contract
  Ready 前必须可自动验证，不能留给 handler 临时决定。

### 资源与可观察性

- 遍历流式进行；全局队列、每库 admission、批次、worker、错误/issues 保留数量和请求体
  都必须有上限。不得按目录项启动无界 goroutine。
- 必须可观察 library/scan ID、状态、阶段、开始/结束时间、进度可用性、发现/跳过/错误统计、
  队列深度和安全失败码；相对路径只保留排障所需的有界安全摘要。
- 取消传播到 walker 和数据库批次，在安全提交点退出；应用停机把未完成工作恢复为可重试
  状态，不能伪装成功。

## Fixture 与验收场景

所有测试只使用临时目录和合成媒体，不读取开发者真实媒体。

### 正常

- `/library` 下含普通子目录、空目录、隐藏目录和受支持媒体；创建具名库后持久化配置和一次
  durable full scan，扫描成功后目录/资产/直接与递归计数一致。
- 创建多个不重叠子目录库；改名保持 library ID、根、索引和 scan history 不变。
- 手动、启动和默认 24 小时计划都走同一 full-scan admission；修改或关闭计划可跨重启恢复。

### 边界

- 选择 `/library` 根时使用空 `root_rel_path`，并因此拒绝任何其他库；拒绝相同、祖先和
  后代重叠。
- 覆盖 traversal、重复编码、NUL、symlink、特殊节点、nested mount、深目录、空目录、
  Unicode/大小写碰撞、系统派生/回收目录跳过和同库重复扫描请求。
- 约 10 万媒体/1 万目录、四核/4 GiB 为容量验收档；Contract Ready 固定可测预算和环境，
  Backend Ready 提供生产 scanner 子范围证据，完整产品结论保留到后续 Gate。

### 失败

- 根不存在、不可读、运行期 unmount/root replacement、子树权限错误、I/O 错误、WAL busy/
  full/corrupt、进程中断和取消均不得清理旧索引。
- Linux `openat2`、resolve flags、seccomp/LSM 或 no-mount-crossing 边界不可用时失败关闭，
  不得回退到 realpath、`os.Root` 或 device/inode 检查。
- 重复/并发创建、名称冲突、重叠根、并发改名/移除/扫描必须返回稳定结果，不产生孤儿库、
  重复 full scan 或跨库记录。

### 恢复

- 离线库恢复同一根后重试完整扫描并收敛；恢复前仍显示旧索引和离线状态。
- failed/cancelled/interrupted generation 保留上次可靠索引及已安全提交的新增记录，后续成功
  full scan 才清理 stale 项。
- 应用重启后 durable queued/running 状态按冻结协议恢复或重试；重复唤醒保持幂等。
- 移除库前后 fixture 原文件逐字节一致；中断的派生数据清理可重试且永不触碰 `/library`。

## 风险与 Gate 分配

| 风险 | Stage 2 必须提供的证据 | 仍保留到后续 Gate |
| --- | --- | --- |
| R-002 路径逃逸 | 生产认证 handler 到 `internal/files` 的边界、恶意路径、symlink/nested mount、错误脱敏 | 最终只读 volume、运行期 unmount 与发布镜像 |
| R-003 扫描误清理 | 失败/取消/offline/中断无 cleanup，成功 finalize 原子收敛 | 发布级强杀和代表性存储故障 |
| R-004 / R-011 SQLite 与恢复 | 短事务、WAL busy/full/corrupt、重启恢复，不支持网络数据盘 | 真实版本升级、正式备份恢复 |
| R-005 / R-013 容量与公平 | 有界队列/批次、同库 admission、跨库公平、生产 scanner 容量回归 | HTTP/前端并发与完整产品容量 |
| R-012 路径碰撞 | 已定义平台语义、Unicode/大小写/规范化 fixture 和稳定冲突 | 发布平台/文件系统声明 |
| R-016 可复现性 | OpenAPI/migration/sqlc 漂移、单元/race/集成/Linux boundary fixture | 最终 E2E 与 Release matrix |

没有新增风险；以上风险状态不因本 Gate 自动下降。

## Gate S0 逐项判断

| 判断项 | 证据 | 结论 |
| --- | --- | --- |
| 目标版本与稳定需求 | MVP revision 1、FR-LIB/FR-SCN 与适用 NFR 已冻结 | 通过 |
| 单一用户结果与非目标 | 单根映射 → UI 多库 → 可靠扫描闭环；Stage 3～5 与写原件能力明确排除 | 通过 |
| Owner 与依赖 | library/scanner/jobs/files/sqlite/api/app 责任和交接已明确 | 通过 |
| ADR 与架构影响 | ADR-0001/0002/0003/0004/0006/0008/0009 已覆盖；无新 C3 决策 | 通过 |
| Fixture 与验收 | 正常、边界、失败、恢复及容量环境已列出 | 通过 |
| 风险与 fallback | R-002/003/004/005/011/012/013/016 已分配；安全/一致性失败关闭 | 通过 |

## 下一步与禁止声明

- 当前后端任务：先执行 `S2-001`；`S2-101` 可同步设计，但两个 Contract Ready 必须共同解决
  首次扫描交接，且不得并行进入实现。
- 允许的下一步：修改并评审 Stage 2 所需 OpenAPI、只追加 migration、capability interface、
  错误矩阵、事务/恢复语义和自动化契约测试。
- 禁止的下一步：以已有 spike 代码直接宣称 `S2-002`/`S2-102` 完成，或让业务 UI 依赖
  尚未通过 Contract Ready 的 mock/临时 DTO。
- 禁止的声明：媒体库或扫描可用、Stage 2 Backend Ready、前端可集成、可在 LAN/公网预览、
  容量目标已证明或 MVP 可发布。
- 评审日期：2026-07-27
