# FolioPath 文档索引

这里保存 FolioPath 在编码前与开发过程中的产品、交互和工程约束。项目仍处于规划与早期开发阶段；标为“提案”或“待确认”的内容不是已经发布的能力。

## 从哪里开始

| 读者 / 任务 | 首先阅读 | 继续阅读 |
| --- | --- | --- |
| 了解项目 | [项目 README](../README.md) | [产品需求](product-requirements.md)、[可行性研究](feasibility-study.md)、[路线图](roadmap.md) |
| 确认产品范围 | [需求确认清单](requirements-checklist.md) | [产品需求](product-requirements.md)、[用户流程](user-flows.md) |
| 设计界面 | [界面设计规范](ui-design.md) | [用户流程](user-flows.md)、[API 设计](api-design.md) |
| 开发后端或扫描器 | [架构](architecture.md) | [目录与依赖约束](project-structure.md)、[数据模型](data-model.md)、[API 设计](api-design.md)、[安全模型](security.md) |
| 开发前端 | [界面设计规范](ui-design.md) | [用户流程](user-flows.md)、[API 设计](api-design.md) |
| 部署和运维 | [部署](deployment.md) | [安全模型](security.md)、[测试策略](testing-strategy.md) |
| 判断能否开工 | [开发就绪评审](development-readiness.md) | [可行性研究](feasibility-study.md)、[风险登记](risk-register.md) |
| 修改架构 | [Agent 约束](../AGENTS.md) | [ADR](adr/)、[架构](architecture.md) |

## 产品与体验

- [产品需求](product-requirements.md)：愿景、用户、范围、需求编号和验收标准。
- [需求确认清单](requirements-checklist.md)：已确认基线和仍需要产品决策的事项。
- [用户流程](user-flows.md)：创建媒体库、扫描、浏览、搜索、查看和异常恢复流程。
- [界面设计规范](ui-design.md)：信息架构、页面、组件、响应式、状态、可访问性和动效边界。
- [路线图](roadmap.md)：不承诺日期的实施阶段、依赖与阶段出口条件。
- [术语表](glossary.md)：统一 `/library`、媒体库、目录、递归浏览、扫描和派生数据等词义。

## 可行性与开工准备

- [可行性研究](feasibility-study.md)：产品、技术、性能、媒体、SQLite、安全、运维、跨架构和许可证的条件 Go 结论。
- [风险登记](risk-register.md)：概率、影响、触发信号、缓解、fallback、Owner 角色和发布阻断风险。
- [开发就绪评审](development-readiness.md)：开工门槛、阶段 0 顺序、Definition of Ready 与 Definition of Done。
- [路线图](roadmap.md)：从需求/spike 到发布安全的阶段依赖和出口条件。

## 工程与交付

- [架构](architecture.md)：运行拓扑、技术栈、代码布局和核心流程。
- [目录与依赖约束](project-structure.md)：后端、前端、生成代码和测试文件的放置规则。
- [数据模型](data-model.md)：SQLite 领域对象、索引、扫描一致性和迁移语义。
- [API 设计](api-design.md)：计划中的 `/api/v1` 资源、请求约定和错误模型。
- [安全模型](security.md)：路径、网络、媒体解析、容器和日志的信任边界。
- [部署](deployment.md)：计划中的单容器部署、权限、备份、恢复和升级。
- [测试策略](testing-strategy.md)：测试层次、风险覆盖、测试数据和发布门槛。
- [Agent 约束](../AGENTS.md)：代码边界、不可破坏的产品约束和修改规则。

## 架构决策记录

- [ADR-0001：采用 Go、React 与 SQLite 的模块化单体架构](adr/0001-go-react-sqlite.md)
- [ADR-0002：以单一允许根目录管理多个媒体库](adr/0002-library-path-model.md)
- [ADR-0003：使用扫描代次保证索引最终一致性](adr/0003-scan-consistency.md)

新的架构决策使用连续编号。已接受 ADR 不直接改写；方向改变时新增 ADR，并将旧记录标记为被替代。

## 状态与优先级

文档使用以下含义：

- **已接受 / 已确认**：当前实现必须遵守；改变架构约束需要 ADR，改变产品基线需要需求确认。
- **MVP 计划**：首个可用版本的目标，但在实现并验证前不能描述为已经可用。
- **提案 / 推荐默认**：用于评审的具体建议，等待确认后才转成基线。
- **未来**：明确不属于 MVP，不应为其预先增加部署或架构复杂度。

发生冲突时不要静默选择其中一份文档。先检查已接受 ADR 和已确认需求，再同步修改所有受影响文档；不确定时把问题加入[需求确认清单](requirements-checklist.md)。

## 变更同步表

| 变更类型 | 至少同步更新 |
| --- | --- |
| 用户可见功能、格式或配置 | `README.md`、产品需求、相关用户流程 |
| 媒体库路径或删除语义 | 产品需求、架构、数据模型、安全模型，必要时新增 ADR |
| API 路径、字段或错误 | API 设计；开始编码后以 `api/openapi.yaml` 为准 |
| 页面、组件、响应式或动效 | 界面设计规范、相关用户流程 |
| 数据表、索引或迁移语义 | 数据模型，必要时新增 ADR |
| Docker、权限、备份或升级 | README、部署、安全模型 |
| 测试门槛或支持矩阵 | 测试策略、README |
| 可行性假设、spike 或风险状态 | 可行性研究、风险登记、开发就绪评审 |
