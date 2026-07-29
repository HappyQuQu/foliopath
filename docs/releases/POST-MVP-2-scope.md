# POST-MVP-2 Scope Manifest — Revision 1

## 冻结记录

- 版本：`POST-MVP-2`
- 产品显示标识：`Post-MVP/2`
- Scope revision：`1`
- 状态：`Scope Frozen`
- 冻结日期：2026-07-29
- 基线事件：[CR-2026-005](../changes/CR-2026-005-automatic-library-discovery.md)
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- 前置版本：`MVP-2026-07-23`；本 manifest 不改变其 scope 或 Release Candidate 结论
- 独立能力关系：不改变已冻结 `POST-MVP-1`，也不以自动发现阻断其 VSP Gate
- Scope-budget exception：N/A；这是独立后续版本，不向冻结 MVP 插入能力

## 版本目标

当用户把文件或文件夹放入已有媒体库时，FolioPath 在支持的本地 Linux 文件系统上自动、
近实时地更新索引和当前页面，无需等待最长 24 小时或手动扫描；文件事件不可靠时安全降级，
完整 generation 扫描仍保证最终一致。

## 冻结功能需求

- `FR-SCN-010`：在受支持的 Linux 本地文件系统上，系统默认自动监听已启用媒体库的目录
  变化，并把新增、修改、删除和重命名提示转换为有界的定向校准任务；用户无需手动扫描。
- `FR-SCN-011`：新建目录必须在安全确认后加入监听范围并进入目录索引，包括空目录；删除或
  移动目录只有在父目录可可靠枚举、媒体库根身份未变化且目标缺失已确认时才能清理对应子树。
- `FR-SCN-012`：监听溢出、错误、资源不足、程序停机、根目录不可用或事件无法确认时，系统
  必须保留已有索引，显示 degraded/offline，并安排或等待完整扫描恢复一致性。
- `FR-SCN-013`：管理员必须能查看自动发现的 `active`、`degraded`、`unsupported` 或
  `disabled` 状态及脱敏原因；关闭自动发现不得关闭创建、启动、手动或定时完整扫描。
- `FR-SCN-014`：每次成功的定向校准必须递增独立内容 revision；可见页面检测到变化后自动
  失效并重新获取受影响查询，不使用 WebSocket。

## 冻结非功能需求

- `NFR-REL-002`：文件事件不参与完整 generation 的成功清理资格；事件丢失、重复、乱序、
  溢出或进程终止后，下一次成功完整扫描必须最终收敛。
- `NFR-PERF-003`：watcher、dirty 集合、任务 admission、定向枚举、SQLite 写入和前端刷新
  必须有界；10 万媒体／1 万目录目标档不能使用每文件常驻 watch 或无界事件/刷新队列。

详细事件、删除、掉盘、revision、API/UI 候选和质量属性由
[FTR-SCN-001](../features/automatic-library-discovery.md)定义。精确 debounce、batch、
watch/queue 上限、延迟 SLO、OpenAPI 和 schema 由 WCH-S0/S1 的真实证据冻结；只要不改变
上述用户结果和安全不变量，可在同一 feature 合同内收敛内部数值。

## 冻结非目标

- `POST-MVP-2-NG-001`：用 watcher 取代创建、启动、手动或定时完整扫描。
- `POST-MVP-2-NG-002`：保证 SMB、NFS、FUSE、云盘或不转发内核事件的挂载近实时。
- `POST-MVP-2-NG-003`：轮询整个文件树来模拟实时监听。
- `POST-MVP-2-NG-004`：WebSocket、独立 worker、Redis、外部消息队列或新部署单元。
- `POST-MVP-2-NG-005`：修改、移动、重命名或删除原始媒体。
- `POST-MVP-2-NG-006`：收到单个 delete/unmount 事件就直接清空媒体库或目录子树。
- `POST-MVP-2-NG-007`：macOS/Windows 发布级等价承诺。
- `POST-MVP-2-NG-008`：向用户暴露 debounce、watch 数、队列或并发等内部调优参数。

## 冻结验收

- `WCH-AC-001～012` 全部适用，定义见
  [feature spec](../features/automatic-library-discovery.md#验收标准)。
- `WCH-S0 Architecture Ready`、`WCH-S1 Contract Ready`、`WCH-S2 Backend Evidence
  Ready`、`WCH-S3 Consumer/UI Ready` 与 `WCH-S4 Integrated Slice Done` 必须依次通过。
- Linux amd64/arm64 使用同一事件、安全和恢复 fixture；目标浏览器覆盖
  Chromium、Firefox、WebKit。
- 所有适用的架构、生成、契约、unit/race/integration、mount、恢复、E2E、容量、可访问性
  和文档检查必须实际执行；未执行不能以计划代替。

## 继承约束

本版本继承 MVP 的全部产品与安全不变量，尤其是：

- 原媒体只读；
- `/library` 单一锚定媒体根与 mount crossing 失败关闭；
- filesystem 是媒体存在与层级真相；
- SQLite、watch 状态、任务和缓存是可恢复派生状态；
- 离线、权限失败或不确定删除保留可靠索引；
- 完整 generation 扫描是正确性基线；
- 认证业务 API、CSRF 和错误脱敏；
- 单容器模块化单体、现有 capability 依赖方向；
- 有界后台任务、短事务和跨媒体库公平。

本 manifest 冻结范围，不表示 ADR 已接受、实现已开始、Gate 已通过或版本已发布。
