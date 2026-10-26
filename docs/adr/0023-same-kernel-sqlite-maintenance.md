# ADR-0023：SQLite 在线维护必须共享同一内核锁域

状态：Accepted，2026-10-23。FR-SCN-004/006、NFR-OPS-001；scan-worker 故障维护切片。

## 背景与证据

Docker Desktop 的 Linux 主应用与 macOS 宿主机直接打开同一 bind-mounted SQLite WAL 文件时，
文件锁/共享内存并不构成同一内核锁域。合成库在正常扫描＋读取/取消下 120 秒、519 次扫描与约 2.4 万次读取无退出；
对同一合成目录加入宿主机 sqlite3 quick_check 后，复现 worker claim 的 SQLITE_IOERR_SHORT_READ（522）与自动重启。
此前原 6010 的无原因码退出不能逐次回溯，但维护操作存在同类风险，不能继续用这条路径验证健康。

## 决策

- 在线 SQLite 检查在拥有该数据的主应用容器内执行，共享 Linux 内核/挂载与锁域，不从宿主机打开活跃 DB。
- 增加 `foliopath database-check`：固定 `/app/data/foliopath.db`，Linux-only、只读、15 秒上限，
  在一个只读快照中 quick_check＋foreign_key_check。不得启动服务器、worker、模型或迁移，不支持任意路径参数。
- 成功/失败只输出固定文案，不打印 SQLite 原始诊断、表名、数据、SQL 或主机路径。现有 readiness 快检不执行全库检查。
- 不放宽扫描租约、事务或 fail-closed 语义，不对未知 I/O/损坏无限重试；增加固定错误来源/阶段/数值码帮助定位。
- 备份/恢复继续先停止主应用再完整复制数据；宿主机只可检查已停写且脱离活动库的备份副本。

## 替代方案与后果

拒绝吞掉 worker I/O 错误、自动重建数据库、增大重试次数或绕过 WAL：它们可能掩盖真实损坏。
没有新增部署单元、网络端点、卷、数据库或 UI；独立 AI 服务仍无媒体/数据库挂载。
命令在容器 shell 权限范围内使用，安全验证须证明缺失文件不被创建、坏库失败、只读、取消和并发扫描不被干扰。
这是维护路径修复，不证明所有未知 worker 退出或无限时运行均已解决。
