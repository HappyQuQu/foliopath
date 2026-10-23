# 视频任务按钮与实际浏览器闭环

CR-2026-030 / POST-MVP-5 / FR-INT-004/016，Owner 前端 ai。
消费既有 `POST /libraries/{libraryId}/ai/video-semantic/jobs`，无后端合同或存储变更。

## 实现

- 视频进度区增加 missing-only 与全量重建入口；全量重建使用原共享确认对话框。
- 复用生成 HTTP 客户端、CSRF、幂等键、ETag、mutation invalidation、runAction、持久任务 ID 与取消流程。
- 操作完成后刷新视频状态；不复制工作队列或任务状态机。
- 缺模型、离线库、未开启、设置清理/读取失败或提交中禁用；状态请求失败不提供旧数据按钮。
- 未记录覆盖率允许第一次回填；降级不计作已完成。确认提示只修改派生数据。

## 验证

- Docker web 构建成功，含双 TypeScript 检查及 Vite；保留原大 chunk 警告。
- Docker 38 项回归通过（页面 26、API adapter 9、query 3）。首次 adapter 用例漏 ETag，
  现有校验正确拒绝；补上真实合同要求的测试 header，以只读挂载最新测试文件复跑通过。
- `make arch-check` 通过。
- 实际 app 镜像 manifest-list：`sha256:f7e47b7a8cd9fc2a00e73d1c7b8246fb669c8b1baea46bc6233f497104c99818`。
- 本地 `foliopath-unrecorded`、合成库 `Unrecorded synthetic library`：模型恢复前按钮禁用，恢复后自动可用。
- 浏览器点击补齐 → 状态从未知到 0/2 处理中 → 任务页显示补齐操作完成、2/2 已处理 → 覆盖率 1/2、降级 1。
- 浏览器点击重建 → 专属确认对话框 → 确认提交 → 任务页显示重建和补齐两条完成记录 → 覆盖率恢复 1/2、降级 1。
- 取消确认不提交、正确 library/mode/CSRF/幂等参数与任务追踪已有定向测试；运行中取消和跨库并发未做本轮实际浏览器验收。

结束后停止两个本地容器，保留隔离数据卷/合成夹具；不使用 server222、真实媒体或推送。
本轮没有完整七项发行检查、全量 E2E、容量矩阵；人物生产接入仍未完成，不宣称全 AI 完成。
