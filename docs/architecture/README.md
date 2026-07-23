# FolioPath 系统架构档案

## 目的与状态

本目录把 FolioPath 的产品范围转化为可实施、可验证、可演进的系统架构。它不是为了增加文档数量，
而是让每个能力、状态、数据、接口和质量门槛都有唯一所有者，避免功能无限扩张、模块互相穿透或同一
规则在多个源码位置分别实现。

当前状态必须区分：

- **Target**：已接受的目标架构，实现必须朝此方向收敛。
- **Current**：仓库当前真实存在的代码、工具和部署能力。
- **Spike validated**：只在报告所述平台与 scope 获得证据，不能等同于产品完成。
- **Planned gate**：尚未落地，进入相关阶段前必须实现。

目前模块化 Go spike、SQLite/generation、Darwin 与原生 Linux amd64/arm64 `openat2` 路径边界已有局部
证据，`api/openapi.yaml` 已成为 HTTP 结构权威，TypeScript 类型、唯一 Web API client、
摘要锁、语义兼容检查和双架构 CI 工作流已建立；这些 scope 仍不能等同于产品完成。最小
`cmd/foliopath` 已建立并只调用 `internal/app.Run`；`internal/app` 已拥有经过验证的固定
`/library`、`/app/data` 和认证前回环监听配置，以及进程根取消、唯一组合入口、顺序启动、
失败回滚、运行故障传播、反向关闭和有界停机。当前已接入单 HTTP listener、服务端 request ID、
统一安全错误、JSON 日志与在途请求排空，但尚无健康/状态路由、数据库或业务 handler，因此仍
不是可用的产品服务。认证、React 产品应用、
完整媒体 adapter、Docker 和发布运维链路仍未建立。
详细状态见[开发就绪评审](../development-readiness.md)。

## 架构档案导航

| 关注点 | 权威入口 | 回答的问题 |
| --- | --- | --- |
| 系统边界与运行形态 | [系统上下文与运行视图](system-context.md) | 系统与用户、NAS、代理、卷、SQLite 和媒体工具如何交互 |
| 模块与运行规则 | [模块、数据与运行时边界](modules.md) | 谁拥有规则、接口、事务、任务、错误、配置和并发 |
| 范围与交付治理 | [交付与架构治理](delivery-governance.md) | 什么进入版本、如何变更、为何后端优先、何时能进入下一 Gate |
| 需求到实现 | [需求—架构追踪](traceability.md) | FR/NFR 落到哪个 capability、契约、数据、风险与验证 |
| 前端子系统 | [前端架构](frontend.md) | 组件、样式、状态、API、列表和测试如何保持唯一所有权 |
| 自动约束 | [架构适配度检查](fitness-functions.md) | 哪些架构不变量由本地/CI/发布检查执行，当前是否真的落地 |
| 决策历史 | [ADR 索引](../adr/README.md) | 为什么选择当前结构，如何替代已有决定 |

顶层[架构概览](../architecture.md)保留技术栈、核心流程与部署摘要；数据、安全、API、部署、UI 和测试
细节仍由各自专项文档负责。本目录只定义它们之间的系统关系和治理，不复制第二份事实来源。

## 架构原则

1. **范围先于结构**：没有目标版本、FR/NFR 和验收结果的能力不进入生产代码。
2. **一个概念一个所有者**：业务规则由 capability 拥有，外部访问由 adapter 实现，交付层只组合。
3. **契约先于消费者**：OpenAPI、迁移、错误和状态语义先稳定，业务前端再消费生成客户端。
4. **安全失败关闭**：路径、认证、扫描清理和不可重建数据不以可用性或交付速度换取弱化。
5. **资源始终有界**：请求、队列、goroutine、数据库事务、媒体进程、缓存和 DOM 都有上限与取消。
6. **运行时简单，源码边界清楚**：保持一个部署单元，不用微服务弥补模块职责不清。
7. **计划不冒充证据**：Current、Target、Spike 与 Released 状态分开记录。
8. **规则能够执行**：重要依赖、契约、迁移、安全、前端一致性和发布条件逐步变成 fitness function。

## 唯一事实来源

| 决策对象 | 唯一事实来源 |
| --- | --- |
| 冻结版本、需求/非目标/验收 ID | [Scope manifest](../releases/MVP-2026-07-23-scope.md)；产品需求解释语义，不能静默改变 manifest |
| 运行拓扑、核心技术和结构性取舍 | 已接受 [ADR](../adr/README.md) |
| HTTP schema 与公开错误 | `api/openapi.yaml` 已是权威；[API 设计](../api-design.md)只解释动机与实现参数 |
| 数据演进 | 只追加 migration；[数据模型](../data-model.md)解释语义 |
| 视觉、交互和可访问行为 | [UI 设计](../ui-design.md)；代码 token/Story 在落地后作为可执行伴随物 |
| 源码放置与依赖方向 | [项目结构](../project-structure.md)、[模块边界](modules.md)与根 `AGENTS.md` |
| 验证和发布判断 | [测试策略](../testing-strategy.md)、fitness functions 与[有证据的 Gate 记录](../gates/README.md) |

发现冲突时暂停受影响切片，按[交付治理](delivery-governance.md)处理；不得让“当前代码已经这样写了”
自动成为架构决定。

## 架构变更触发器

以下任一变化必须先做影响分析，并按需要新增 ADR：

- 新增进程、容器、数据库、队列、外部服务或新的持久化边界；
- 改变信任边界、身份模型、原文件只读承诺或路径解析策略；
- 改变 capability 所有权、允许依赖方向、关键事务或任务一致性；
- 改变公开 API 兼容策略、数据迁移策略或不可重建/可重建数据分类；
- 引入第二套前端状态、HTTP、设计系统、主题或集合实现；
- 改变目标平台、容量档、备份恢复承诺或发布拓扑。

普通实现仍须通过对应 Architecture/Contract/Backend/Frontend/Release Gate；“不需要 ADR”不等于可以绕过
需求、测试或所有权检查。
