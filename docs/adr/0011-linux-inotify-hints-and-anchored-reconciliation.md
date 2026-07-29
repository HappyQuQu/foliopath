# ADR-0011：Linux 文件事件只触发锚定的定向校准

## 状态

已接受（2026-07-29）

## 决策角色

- 产品：产品用户
- 架构：FolioPath maintainers
- 安全／数据／发布：FolioPath maintainers

## 背景与驱动因素

`POST-MVP-2` 的 `FR-SCN-010～014` 要求已有媒体库中的新增文件和目录自动、近实时出现在
索引和当前页面。现有创建、启动、手动和定时完整扫描可以最终收敛，但最长可见延迟由扫描周期
决定。

文件事件可能重复、乱序、合并、丢失或 overflow；网络文件系统可能完全不转发；一个挂载
消失可能表现为大量删除；Linux watch 还受内核资源上限约束。另一方面，FolioPath 要求所有
真实媒体访问由 `/library` FD 锚定，并以
`openat2(RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_XDEV)` 失败关闭。事件返回的
路径名不能绕过该边界。

本决定补充而不替代 [ADR-0003](0003-scan-consistency.md) 的 generation 一致性和
[ADR-0009](0009-linux-openat2-single-media-root.md) 的内核路径边界。

## 备选方案

### A. 只缩短定时完整扫描周期

- 优点：不引入新内核机制，复用现有 generation。
- 缺点：10 万媒体库频繁全树扫描造成持续 I/O；仍不能稳定达到数秒可见。

### B. watcher 事件直接 upsert/delete 数据库记录

- 优点：实现表面简单、延迟低。
- 缺点：把不可靠事件当成事实；掉盘或 overflow 可能误删大量索引；事件路径绕过安全 reopen；
  重复/乱序和慢写会制造半文件与任务风暴。

### C. Linux 目录 watcher 只产生提示，再执行锚定定向校准

- 优点：快速路径与安全/正确性解耦；变化重新经过已有 path/files 边界；删除要求可靠父目录
  枚举；overflow 可安全退回完整扫描。
- 缺点：需要 watch 生命周期、durable admission、content revision、状态与容量证据。

### D. 递归轮询目录 mtime 或快照

- 优点：不依赖 inotify 事件传播。
- 缺点：目录 mtime 不能可靠代表完整后代变化；实质仍是频繁扫描。

## 决策

选择 **C：Linux 目录 watcher 只产生提示，再执行锚定定向校准**。

### 正确性与清理资格

1. watcher 事件不是文件存在、缺失、类型、稳定性、路径安全或索引清理的证据。
2. `internal/scanner` 是事件合并、定向范围、状态、清理资格和 content revision 的唯一
   policy owner。
3. `internal/files` 实现 Linux 目录 watch adapter，并只返回 library-relative 事件。真实
   枚举、stat/open 和 watch 新目录前必须从原 `/library` 锚定边界重新解析。
4. 新增/修改通过安全枚举和 fingerprint 确认；慢写必须在 close-write/move-in 或稳定窗口后
   才进入媒体处理。
5. 删除只能清理一次成功定向校准明确覆盖的范围。媒体库根身份、父目录完整枚举或目标缺失中
   任一项不能确认时保留旧索引。
6. `IN_Q_OVERFLOW`、`ENOSPC`、unmount、watch invalidation、根替换、权限或 I/O 错误使
   watcher degraded/offline；此后停止删除性校准并合并安排完整扫描。
7. 完整 generation 扫描继续是唯一媒体库级陈旧清理和最终一致来源。

### Watch 范围与资源

1. Linux 发布实现只为已安全发现的目录注册 watch，不为媒体文件注册常驻 watch。
2. 新目录必须先安全确认，再注册 watch 和进入有界后代枚举。
3. watches、dirty 目录、事件 channel、durable queue、批次和每库/全局并发都有硬上限。
4. 达到任何上限时合并为“需要完整校准”，不继续假装 active。
5. 应用不得修改宿主机 sysctl；资源不足通过稳定错误码和运维文档提示。
6. 不对 SMB/NFS/FUSE 等不可靠事件源作近实时承诺；失败时安全降级。

### 持久任务与事务

1. process-local event channel 只是唤醒提示；被接受的定向工作必须进入可恢复、幂等的 durable
   queue，并复用 `internal/jobs` 的 claim/lease/retry/cancel 机制。
2. Contract Ready 可选择扩展现有 scan queue 或新增专用 reconcile queue，但不得复制
   job 状态机，也不得把 incremental work 伪装成 full generation。
3. 文件枚举、watch 注册和媒体处理期间不持有 SQLite 事务。一次定向提交只包含有界 upsert/
   明确范围 cleanup、派生任务 admission 和 content revision 递增。
4. 完整扫描可吸收其范围内尚未开始的增量工作；并发时不能让旧增量结果覆盖新 fingerprint。
5. 应用重启未收到的事件由现有启动完整扫描纠正；已入 durable queue 的工作按 lease 恢复。

### 客户端更新

1. 增加独立 `content_revision` 通知客户端重新获取；它不替代 reliable generation 或现有
   cursor revision。
2. 首版使用认证的轻量 HTTP revision 轮询和 TanStack Query 统一 invalidation，不增加
   WebSocket。
3. 页面隐藏时暂停轮询；revision 不变时不得重取目录/媒体页；变化后建立新的查询/cursor 链。
4. 如果后续证据要求 SSE/WebSocket，必须新增 ADR。

## 后果

### 收益

- 本地文件变化可以数秒内进入索引和页面，同时保留完整扫描的最终一致性。
- 掉盘、overflow 和批量删除事件不会直接清空可靠索引。
- 所有真实路径继续经过唯一 Linux 内核边界。
- 目录级 watch 数量随目录而非媒体文件数量增长。
- 不新增服务、外部队列或 WebSocket。

### 成本与剩余风险

- 需要 Linux inotify adapter、watch 生命周期、durable 定向任务、content revision 和前端
  invalidation。
- 1 万目录需要相应 watch 资源；宿主限制过低时只能 degraded。
- 网络盘可能没有可靠事件，只能继续依靠完整扫描。
- rename 按 path identity 表现为新路径加旧路径确认删除。
- 轮询增加轻量请求；频率和刷新范围必须实测。

### 迁移与运维

- 预计需要只追加 migration；精确表/列由 WCH-S1 冻结。
- 部署文档需说明 `inotify` 限制、degraded 状态和不会自动修改 sysctl。
- 自动发现可禁用；禁用不影响现有完整扫描。

## 验证与复审

- 对应 architecture fitness function：计划新增 AF-019，验证 watcher 不能绕过 `internal/files`、
  不能直接清理索引、资源必须有界且 overflow 失败关闭。
- 证据或测试：
  - Linux inotify close-write/move/create/delete/rename/new-directory；
  - openat2 traversal/symlink/nested mount/root replacement；
  - overflow/ENOSPC/unmount/权限失败；
  - durable lease、重复/乱序、强杀与启动收敛；
  - 100k 媒体／10k 目录 watches 和 burst；
  - revision 轮询、query invalidation、cursor/滚动/焦点 E2E；
  - 原媒体 hash/mtime 不变。
- 需要复审本 ADR 的条件：
  - 引入轮询全树、SSE/WebSocket、外部队列或新部署单元；
  - 支持 nested mounts 或保证网络文件系统近实时；
  - watcher 获得媒体库级清理资格；
  - 目录 watch 资源无法满足目标容量，需要改变核心机制。
- 替代／被替代关系：无；补充 ADR-0003 与 ADR-0009。
