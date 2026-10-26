# 流式回填处理器与真实 SQLite 提交

2026-09-15，CR-2026-030 / POST-MVP-5，FR-INT-017/019/020、NFR-INT-007/009/010。
Owner：semantic 任务/编码，SQLite 事务；不改变 app 配置或旧激活逻辑。

## 已实现

- BackfillProcessor 统一 StreamImageEncoder；旧构造器通过 legacy bridge 保持 SigLIP tensor。
- 新 NewStreamBackfillProcessor 允许绑定 BoundEncoder，不复制任务循环、分页或进度逻辑。
- 源指纹检查与文件关闭仍在同一候选处理函数；最终提交沿用事务内可写状态/维度复查。
- 新单元测试验证绑定编码到 512 维 binary16、退役拒绝、源变化拒绝和流关闭。
- 真实 SQLite 任务生命周期测试：创建/绑定 512 维代次、准入、claim、分页、提交、任务成功；
  第二场景在编码返回后、提交前退役代次，确认零 embedding 写入且任务 failed。

## 实际执行

- `go test -race ./internal/semantic ./internal/inference/remoteml` 通过。
- `go test ./internal/store/sqlite ./internal/app` 通过（旧构造器回归包含在内）。
- `go test -race ./internal/store/sqlite -run TestStreamBackfillPersistedLifecycle -count=1` 通过。
- `make arch-check`、相关包 `go vet`、`git diff --check` 通过。

## 边界

本次 SQLite 测试的预处理、推理和文件内容源仍为合成桩。不是完整 files → native → ML →
SQLite 实际联合测试，也没有真实搜索页面/HTTP/API 测试。internal/app 仍走旧构造器。
部署身份准入、模型注册/激活、app 配置、完整本地联合验收仍待完成；没有修改用户媒体、
server222、正式数据库或部署。未提交/推送。
