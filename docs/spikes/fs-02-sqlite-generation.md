# FS-02：SQLite 与扫描 generation spike 报告

## 结论

**状态：Passed（当前正确性范围通过）**

**验证日期：2026-07-23**

**验证环境：macOS（Darwin/arm64）、Go 1.26.4、SQLite 3.53.3（modernc.org/sqlite）**

真实文件型 SQLite、嵌入式 Goose 迁移、WAL/外键/busy timeout、单库单活动扫描、单调 generation、分批 upsert、失败/取消/离线/重启保留以及成功后的原子清理均已由单元和跨组件集成测试验证。当前实现满足 FS-02 定义的正确性 scope。

该结论不覆盖磁盘已满、真实 `SIGKILL`、长期 WAL/checkpoint 压力、SMB/NFS、在线备份/恢复或发布升级演练。这些仍是 FS-04/FS-05 与发布门槛，不能由本 spike 推导为已通过。

## 目标与范围

本 spike 验证：

- 存储层能在真实文件数据库上运行只追加的嵌入式迁移，并确认 WAL 实际启用；
- 每个连接启用外键并设置 busy timeout，拒绝已知缺少 WAL-reset 修复的 SQLite 版本；
- 每个媒体库的完整扫描使用单调 generation，且同一媒体库最多一个排队或运行扫描；
- 目录和资产按有界批次 upsert，文件系统遍历期间不持有数据库事务；
- failed、cancelled、offline、interrupted 或根身份变化不会清理旧索引；安全提交的新行可保留给后续完整扫描收敛；
- 只有成功 finalize 才在一个事务内删除陈旧行、更新目录计数、标记扫描成功并推进 `current_generation`；
- 不同媒体库的数据、generation 与清理互相隔离。

本轮不验证 FTS、目标容量性能、媒体元数据、HTTP API、缩略图、备份/恢复或多实例共享数据库。

## 实现摘要

- `migrations/00001_initial.sql` 创建媒体库、扫描代次、目录和资产表；`migrations/embed.go` 将迁移嵌入二进制，`internal/store/sqlite/migrate.go` 通过 Goose 执行。
- 初始 schema 可表示尚无开始时间的 `queued` 扫描，并用部分唯一索引禁止同库同时存在多个
  `queued`/`running` 扫描；当前 scanner spike 仍直接进入 `running`，正式 admission/领取由
  尚未实现的 `internal/jobs` 拥有。资产 kind 同步支持 `image`、`animated`、`video`，
  当前 GIF 扩展名分类为 `animated`。
- 目录父子关系和资产所属目录使用带 `library_id` 的复合外键，数据库层拒绝跨媒体库关系；
  repository 查询仍必须显式带库作用域，不能只依赖外键。
- `internal/store/sqlite.Open` 只接受文件路径，配置 WAL、外键和 busy timeout，并在启动时读取实际 `journal_mode` 与 `sqlite_version()`。当前运行时版本为 3.53.3。
- SQLite 官方文档说明 WAL 依赖同机共享内存语义、不适用于网络文件系统，并记录了相关 WAL-reset 修复版本；实现因此保留本地文件系统约束并拒绝缺少修复的版本。参见 [SQLite WAL documentation](https://sqlite.org/wal.html)。
- 扫描能力接口由 `internal/scanner` 所有，SQLite 只是适配器。开始扫描分配 generation，批量 upsert 使用短事务；文件系统遍历发生在事务外。
- 扫描请求只携带媒体库 ID；scanner 从 repository 读取数据库中不可变的库根，调用者不能自由拼接另一个 `RootRelativePath`。媒体库创建、scanner 与 files 使用同一编码/遍历词法策略。
- finalize 在一个事务内执行陈旧目录/资产清理、目录计数、扫描状态与媒体库当前代次更新。约束和事务条件保证 complete/cancel 竞态只有一个一致结果。
- scanner 在遍历前捕获根身份，在成功 finalize 前再次验证；离线或身份变化转为非成功状态，不取得清理资格。

## 已执行验证

从仓库根目录执行：

```sh
go version
go test -count=1 -v ./internal/scanner ./internal/store/sqlite ./tests/integration
go test -race -count=1 ./internal/scanner ./internal/store/sqlite ./tests/integration
```

结果：

- 所有命令通过；`go version` 返回 `go1.26.4 darwin/arm64`。
- `TestOpenUsesFileDatabaseAndAppliesPragmasToEveryConnection` 记录 SQLite runtime `3.53.3`，并验证真实普通文件数据库、WAL、外键和 busy timeout 的连接行为。
- 数据库测试执行 `integrity_check`、`foreign_key_check` 和 WAL checkpoint；版本门槛测试覆盖 3.51.3 主线修复以及 3.50.7、3.44.6 官方回移边界。
- generation 故障测试覆盖：成功 g1、失败/取消/离线/中断 g2 保留旧索引与当前代次、后续成功 g3 清理陈旧行；finalize 注入失败时整笔回滚。
- 重启用例关闭并重新打开同一个文件数据库，再把遗留 `running` 扫描标记为 interrupted；这是受控重启模拟，不是操作系统级 `SIGKILL`。
- 并发用例覆盖两个 Store 实例竞争同库活动扫描，以及 complete/cancel 竞态的一致获胜者。
- schema/adapter 用例还覆盖 queued 扫描可读取且 `started_at` 为空、queued admission 阻止
  第二个完整扫描、animated GIF kind 可持久化，以及跨媒体库目录父子/资产目录关系被复合
  外键拒绝；故障注入还验证当前代次目录/资产即使被外部损坏为指向同库陈旧目录，finalize
  也会在 cleanup 前失败关闭并保留两侧行。
- scanner 服务测试覆盖根身份改变、离线捕获和 A → B → A 替换不会清理旧索引；取消即使发生在 `CompleteFullScan` 入口也使用独立 finalize context 持久化；同时覆盖隐藏项扫描和系统目录跳过。
- `tests/integration` 使用临时媒体树贯通 library → files → scanner → SQLite，覆盖递归扫描、空目录、直接/递归计数、格式/系统目录跳过、故障后的成功收敛和跨媒体库隔离。

所有数据库与媒体树均在 `t.TempDir()` 中创建。未运行目标容量 benchmark，也未把本地测试耗时解释为产品性能承诺。

## 未验证与剩余门槛

1. **磁盘故障**：未注入 `SQLITE_FULL`、fsync/I/O 错误、只读切换或 finalize 时磁盘耗尽；R-009 和相关数据完整性门槛仍开放。
2. **真实进程死亡**：当前重启测试是受控关闭/重开，不等价于写入或 checkpoint 任意时刻的 `SIGKILL`/断电。
3. **长期 WAL 压力**：未执行长时间读写并发、checkpoint 饥饿、超大 WAL、`SQLITE_BUSY` 风暴或内存/磁盘增长测量。
4. **SMB/NFS**：未在网络文件系统上测试，且 MVP 明确不支持把活动 `/app/data` 放在 SMB/NFS。根据 SQLite 的 WAL 约束，文档警告不是兼容性证明。
5. **备份、恢复与升级**：未演练在线 backup API、受控 checkpoint、恢复校验、迁移中断或跨版本升级；R-004/R-011 不能关闭。
6. **规模与搜索**：没有 10 万媒体/1 万目录容量、FTS5 中英文搜索或浏览并发数据；这些属于 FS-04。

## 决策与下一步

- FS-02 在当前正确性 scope 记为通过，generation/SQLite 方案可以作为后续纵向切片的基础。
- 保留真实文件数据库、WAL 返回值验证、SQLite 安全版本门槛、短事务和原子 finalize 作为不可回退的测试约束。
- 在 FS-04 加入持续读写、checkpoint、busy timeout、数据库增长和目标容量测量。
- 在 FS-05 的 Linux 容器中注入磁盘满与进程强杀，演练备份/恢复/升级，并确认 `/app/data` 位于具备可靠锁语义的本地文件系统。
- 在这些运维门槛完成前，R-003/R-004 只能标记为“缓解中”，不能关闭，也不能宣布项目已达到发布就绪。
