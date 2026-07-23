# S0-106：容量证据 Gate 分配记录

- 日期：2026-07-23
- 目标版本：`MVP-2026-07-23`
- 类型：交付门禁澄清；不改变容量目标、产品 scope、架构或发布要求
- 影响：FS-04、Stage 0 Gate、扫描/浏览/搜索 Backend Gate、UI Gate、Release Gate
- 结论：`Accepted`

## 问题

原 S0-106 把代表性存储、生产 FTS/keyset、媒体/缩略图队列、HTTP 与前端并发都列为
Stage 0 完成条件。但这些证据分别依赖 Stage 2～4 才允许创建的生产能力，代表性设备又不属于
仓库可合成环境。要求 Stage 0 先证明尚未获准实现的功能，会形成循环依赖；用 raw SQL、
tmpfs 或 mock 代替则会产生虚假的产品容量结论。

## 证据分配

要求保留，并分配到最早能够真实产生证据的 Gate：

1. **FS-04 / Stage 0 扫描与索引可行性范围**
   - 四核、4 GiB、约 10,000 目录和 100,000 资产；
   - 混合宽度/深度与独立 1,000 层计数归并；
   - generation 正确性、扫描期间基础索引读取、Go heap、Linux 进程 RSS、SQLite 大小；
   - 1k/10k、5k/50k、10k/100k 三档趋势；
   - 仅用于相同容器/tmpfs 环境的暂定回归预算和超限 fallback。
2. **扫描、浏览与搜索 Backend Gate**
   - 生产有界队列和媒体/缩略图处理完成后测队列、RSS、磁盘增长和失败行为；
   - catalog keyset 完成后测稳定翻页及扫描并发；
   - FTS5 adapter 完成后测三种搜索作用域、语言 fixture、索引一致性与尾延迟。
3. **Browse/Search UI Gate**
   - 通过真实 API 测虚拟化 DOM 上限、滚动/焦点稳定性和用户可感知延迟；
   - 不允许 mock 或直接 SQLite 查询充当 UI 容量证据。
4. **Performance / Release Gate**
   - 在记录设备、CPU/内存、文件系统和存储介质的代表性环境重跑主档；
   - 使用最终双架构镜像测生产 RSS、WAL/checkpoint、缓存、HTTP 与并发退化；
   - 冻结发布预算；超限时降低并发/缩略图规格或声明实测支持上限。

## Stage 0 判断

`tests/performance/TestCapacityBaseline` 已在 Linux/arm64、四核、4 GiB、tmpfs 环境跑通主档，
并由 `make capacity-trend` 对三个规模档记录扫描、读取、heap、`/proc/self/status` `VmHWM`
RSS 与数据库大小。`stage0-comparable-v1` 只作为相同条件下的回归护栏，不是 NAS SLA。

因此 S0-106 的 Stage 0 可行性范围可以关闭；FS-04 完整产品/发布结论继续保持
`Conditional`，上述证据分别阻断对应 Backend、UI 和 Release Gate。此调整不允许提前创建
生产 catalog、搜索、媒体 handler 或业务 UI。

## 暂定回归预算

| 指标 | `stage0-comparable-v1` 上限 |
| --- | ---: |
| 主档扫描/finalize | 120 s |
| 扫描期间基础资产页读取 P95 | 250 ms |
| 扫描后目录页读取 P95 | 100 ms |
| 扫描后资产页读取 P95 | 100 ms |
| Linux 进程 RSS 高水位 | 1 GiB |
| checkpoint 后数据库族 | 1 GiB |

这些宽松上限用于尽早发现数量级回归；只有 Performance / Release Gate 可以把代表性设备结果
冻结为产品预算。

## Fallback

- 同条件回归超限：阻止合并，先定位复杂度、查询计划、事务或内存增长。
- 代表性设备超限：降低全局并发、缩略图规格或声明实测支持上限。
- 需要改变部署单元、存储或核心查询架构：先新增 ADR，不得在性能修复中静默改写架构。

## 证据

- [FS-04 容量基线](../../spikes/fs-04-capacity-baseline.md)
- [ADR-0001：Go/React/SQLite](../../adr/0001-go-react-sqlite.md)
- [ADR-0003：扫描一致性](../../adr/0003-scan-consistency.md)
- [当前 Stage 0 Gate](stage-0-current.md)
- [风险登记 R-005](../../risk-register.md)
