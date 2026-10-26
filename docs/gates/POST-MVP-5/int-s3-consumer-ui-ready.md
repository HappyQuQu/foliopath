# INT-S3：Consumer/UI Ready

- 日期：2026-09-02
- 目标版本：`POST-MVP-5` revision 2
- 范围：`INT-301～311`，A～E 合同消费者与管理/审核 UI
- 判断：**Consumer/UI Ready / Release No-Go**
- 前置：S2A、S2B、S2C Backend Ready

## 完成证据

- 管理界面经生成客户端消费模型、语义、人脸和任务合同，提供按库 ETag 开关、覆盖/失败/generation、
  missing/rebuild、derived/manual clear、managed/direct、固定 `/models` 扫描、兼容拒绝、空间失败和真实
  operation 轮询/取消。离线模型获取按 [CR-2026-023](../../changes/CR-2026-023-s3-offline-model-ui-contract-alignment.md) 对齐。
- 文件名、图片语义和视频语义模式由唯一 URL codec/query-key owner 管理；语义结果复用共享虚拟媒体集合、
  预览/查看器，展示覆盖率、不完整索引、游标失效及视频命中时间与 4/10 帧故事板。
- 标签审核把 AI 建议/置信度与人工标签分离，批量最多 100 项，写入携带 suggestion/curation revision、
  CSRF 和幂等键。人物工作台覆盖已命名人物搜索、匿名 core/edge、批量选择、逐脸归类/排除、建人物、
  人物媒体、移动/移除/合并/重命名、重名 ID 后缀及 revision conflict。
- 单图多人选择同时提供粗略百分比框与键盘可访问等价列表。中英文文案明确匿名相似分组不等于现实身份
  识别，不联网查人、不训练/发布模型；清除不修改原始媒体。
- Vitest 覆盖 adapter 与关键交互；Storybook 全局 axe 为 error，并构建智能审核状态。视觉合同固定
  390/768/1265/1440、light/dark、简中/英文及 reduced-motion；页面保留 DOM 顺序、焦点轮廓和语义控件。

本次实际成功执行：`make fmt`、`make arch-check`、`make generate-check`、`make lint`、`make test`、
`make test-integration`、`make test-e2e`；前端另执行类型检查、生成/架构/视觉引用检查、50 文件 181 tests、
Storybook 静态构建，以及上述四档真实 Chromium axe/水平溢出矩阵。`git diff --check` 通过。

## 发布边界

本 Gate 不宣称 AI 可发布。S4 仍须提供最终审核模型、governed semantic/tag/video/face 质量、真实 native
amd64+arm64 配对镜像、联合容量、SBOM/签名 provenance/notices/VEX，以及 privacy/compliance/security/
release 批准。`groupAssignmentAllowed=false` 时 UI 禁止匿名 core 整组自动归类，只允许逐脸复核。
