# FolioPath 文档

本目录保存 FolioPath 的产品基线、架构约束、交付计划和验收证据。文档按用途分层：
先从本页找到当前权威入口，再进入专题索引；不要从文件数量或日期推断文档是否仍然有效。

当前状态：MVP 功能切片与 `FTR-UIF-001` 已完成，Release Candidate 仍因物理设备辅助功能、
供应链处置和最终不可变 digest 等 Gate 而处于 No-Go。当前冻结范围是
[MVP scope revision 4](releases/MVP-2026-07-23-scope-r4.md)。

## 首先阅读

| 目的 | 当前入口 | 说明 |
| --- | --- | --- |
| 了解产品 | [项目 README](../README.md) | 面向用户的能力、部署和限制 |
| 确认产品范围 | [产品需求](product-requirements.md) | 需求语义；冻结范围以 scope manifest 为准 |
| 了解系统结构 | [架构索引](architecture/README.md) | 模块、运行视图、前端边界、追踪和适配度检查 |
| 查看 API | [OpenAPI](../api/openapi.yaml) | 已实现 API 的权威合同 |
| 设计或修改界面 | [界面设计规范](ui-design.md) | 导航、组件、响应式、可访问性和动效 |
| 部署或运维 | [部署](deployment.md) | 容器、卷、权限、备份、恢复和升级 |
| 查看用户更新 | [更新日志](../CHANGELOG.md) | 自动维护的用户可见新增、改进、修复和注意事项 |
| 查看当前工作 | [开发任务清单](task-list.md) | 当前进度、阻断和下一步 |
| 判断能否发布 | [版本与 readiness](releases/README.md) | scope、候选状态和发布说明 |

## 按文档类型浏览

| 类型 | 索引 | 用途 |
| --- | --- | --- |
| 架构 | [architecture/](architecture/README.md) | 当前系统结构、模块所有权和治理 |
| ADR | [adr/](adr/README.md) | 只追加的架构决策及替代关系 |
| Feature | [features/](features/README.md) | 独立用户能力的范围、合同和交付阶段 |
| 变更记录 | [changes/](changes/README.md) | Change Record 与已批准 slice 内修复 |
| Gate | [gates/](gates/README.md) | Architecture、Contract、Backend、UI 和 Release 判断 |
| 版本 | [releases/](releases/README.md) | 冻结 scope、readiness 和发布说明 |
| 验收证据 | [evidence/](evidence/README.md) | 可复核的视觉、浏览器和容量证据 |
| Spike | [spikes/](spikes/README.md) | 可行性实验；不能代替生产 Gate |

## 当前权威文档

### 产品与体验

- [产品需求](product-requirements.md)
- [需求确认记录](requirements-checklist.md)
- [用户流程](user-flows.md)
- [界面设计规范](ui-design.md)
- [品牌标识规范](branding.md)
- [路线图](roadmap.md)
- [术语表](glossary.md)

### 工程与运行

- [系统架构](architecture.md)
- [目录与依赖约束](project-structure.md)
- [数据模型](data-model.md)
- [API 设计说明](api-design.md)：解释资源边界；wire 合同以 OpenAPI 为准
- [安全模型](security.md)
- [部署](deployment.md)
- [测试策略](testing-strategy.md)
- [供应链与许可证审查](supply-chain-review.md)

### 计划与风险

- [开发任务清单](task-list.md)
- [后端开发清单](backend-task-list.md)
- [前端开发清单](frontend-task-list.md)
- [开发就绪评审](development-readiness.md)
- [可行性研究](feasibility-study.md)
- [风险登记](risk-register.md)

已完成计划仍可作为交付记录，但不应优先于当前产品、架构、OpenAPI、migration 或 Gate
事实。计划状态发生变化时，应更新计划本身或对应专题索引，避免创建新的平行总清单。

## 历史文件如何处理

- scope revision、ADR、Change Record、Gate 和发布证据是审计历史，采用只追加策略。
- 被替代的文件保留原路径，并在文件头或所属索引中指向当前替代项。
- 已完成任务清单保留任务与证据映射；后续工作进入新的 feature 或版本清单。
- `docs/evidence/` 和 `web/qa/` 中相同内容可能代表不同验收状态，不能仅按哈希去重。
- 只有未被任何当前或历史记录引用、且不承担审计含义的生成物或草图才可删除。

## 状态含义

- **已接受 / 已确认**：当前实现必须遵守。
- **冻结**：只追加新 revision，不原地改写范围。
- **Go**：适用检查已经实际执行并通过。
- **Conditional Go**：只允许记录中明确列出的有限下一步。
- **No-Go / Pending**：不得宣称切片或版本完成。
- **提案 / 未来**：不授权进入当前生产实现。
- **已被替代**：保留历史，但实现应遵循其替代文件。

发生冲突时，依次检查冻结 scope、已接受 ADR、OpenAPI/migration、当前 Gate 和产品需求；
不要静默选择其中一份，也不要通过删除旧记录消除冲突。

## 变更同步表

| 变更类型 | 至少同步更新 |
| --- | --- |
| 用户可见功能、格式或配置 | `README.md`、产品需求、相关用户流程、Change Record |
| 媒体库路径或删除语义 | 产品需求、架构、数据模型、安全模型，必要时新增 ADR |
| API 路径、字段或错误 | `api/openapi.yaml`、生成客户端、相关合同测试 |
| 页面、组件、响应式或动效 | 界面设计规范、相关用户流程、视觉证据 |
| 数据表、索引或迁移语义 | 数据模型、migration，必要时新增 ADR |
| Docker、权限、备份或升级 | README、部署、安全模型 |
| 测试门槛或支持矩阵 | 测试策略、readiness、README |
| 可行性假设、spike 或风险状态 | 可行性研究、风险登记、开发就绪评审 |
