# FIX-2026-09-01：人脸组合模型包与独立 generation 激活

- 目标版本：`POST-MVP-5` revision 2 / S2C
- 关联 Gate：`INT-S2C Privacy Ready`（保持 **Release No-Go**）
- 影响任务：`INT-241`、`INT-250`、`INT-251`
- 影响不变量：semantic/face 生命周期隔离；候选模型不得自我授权；旧可靠代次失败回退

## 目的

接受 ADR-0015 的 fail-closed 后端部分，把已经验证的 detector → alignment → 512 维 embedding 边界接入
canonical 模型包和激活 owner。实现不得把候选权重加入内建 catalog，也不得让应用 composition 在最终质量、
隐私、双架构和供应链 Gate 前注册可用 face runtime。

## 实现

- `internal/aimodel` 增加独立的 `face_detection_embedding` format v3：一个包必须精确绑定 detector、embedder、
  threshold profile、七项 transform contract 和逐文件许可；未知/重复/trailing JSON、purpose/version 混淆、
  路径和容量攻击均失败关闭。
- 激活 worker 按 purpose 分派并在同一审核步骤打开三项文件；face runtime 缺失时稳定返回
  `model_unavailable`，不会回退到 semantic runtime。
- migration 35 扩展 `ai_models` purpose，并让 `face_generations` 记录 model/package/profile hash。
  `CommitFaceModelActivation` 在短事务中只退休旧 face generation、创建新 generation、推进全局 revision 和
  完成 operation；semantic active pointer 及各库旧 active cluster 均不改变。
- threshold profile 由 `internal/face` 严格解析；不接受自报 `groupAssignmentAllowed` 或 pass 字段。激活 smoke
  要求 detector 输出有限，embedder 输出有限、非零且精确 512 维。
- 模型列表 API 增加独立 nullable `activeFaceModelId`；供应链 evidence schema v2 要求唯一
  `face_detector` 和 `face_embedder` 角色以及 notices/再分发批准。

## coser 功能证据

操作者授权的 `/Volumes/r0data/picture/coser` 始终只读。最大有界运行选择并解码 2,347 张图片，1,496 张含脸，
产生 1,515 个有限、唯一的 512 维候选；阈值扫描和逐脸拼图暴露了 bridge-face 错误合并，修复后的
smallest-ID anchor coherence 已由定向回归和 100k×512 容量测试固定。该证据足以证明实现链路和错误回退可测，
因此关闭 `INT-241`；它不是身份 ground truth，不关闭 `INT-250` 的 50×20/偏差质量矩阵或 `INT-251` 发布签署。

私有缩略图、路径、CSV 和 embedding 只在 `/tmp/foliopath-coser-review`，不进入 Git 或普通 CI artifact；原图
未移动、改名、修改或删除。

## 回归

定向 Go package、SQLite migration/activation、OpenAPI generation、供应链 hostile-input、架构 fitness 和
coser 准备工具 Python 回归均必须通过。最终内建 catalog、production runtime composition 和消费者 UI 继续为空，
直到 Release Gate 的外部输入全部到位。
