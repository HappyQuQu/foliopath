# FIX-2026-09-02：人脸控制面与取消状态机闭环

- Slice：POST-MVP-5 revision 2 / S2C（`INT-241`、`INT-249`）
- Gate：INT-S2C Implementation Authorized / Release No-Go
- 不变式：任务状态由 capability 唯一 owner 推进；缺少最终模型、质量、双架构及隐私签署时保持不可达

## 变更

- 增加按库人脸设置、missing/all 分析任务、derived clear 与 manual clear 的 HTTP adapter；强制 ETag、
  幂等键、明确确认字串和 manual impact counts。
- 增加 `face.ControlService`，从服务端设置派生 active generation，客户端不能选择 generation 或提交路径。
- 通用 AI operation 取消现在把四类 face operation 路由回 face job/clear owner；owner 不存在时失败关闭，
  不再只改通用 operation 而让底层任务继续执行。
- 按库禁用与分析任务停止共用一个 SQLite 写事务：queued 任务直接进入 cancelled，running 任务进入
  cancelling 并由 lease heartbeat 协作收敛；事务竞争无论由 claim 或 disable 先获得写锁，都不会在禁用后
  留下可新领取的任务。已有 observation、人物和人工约束不被隐式删除。
- production composition 仍不注入任何 face dependency；架构测试持续锁定该 No-Go 边界。
- processor 在 runtime 前只接受冻结的 JPEG/PNG/WebP/GIF 图片格式；视频或未知格式不会进入 detector，
  source fingerprint 不匹配仍在任何 runtime 调用前失败关闭。

## 验证

- `go test ./internal/aimodel ./internal/face ./internal/store/sqlite ./internal/api ./tests/architecture -count=1`
- `make fmt`
- `make generate-check`
- `make lint`
- `make test`

上述检查均于 2026-09-02 执行成功。它们只证明控制面实现与失败关闭组合，不构成 S2C Release Gate Go。
