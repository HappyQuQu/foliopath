# CR-2026-008：后台任务中心

## 状态

- 状态：Confirmed Direction / Scope Proposed
- 变更等级：C2
- 目标版本：`POST-MVP-3`
- 提出日期：2026-07-30
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- Feature：[FTR-OPS-001](../features/task-center.md)
- 风险：[R-020](../risk-register.md)

## 产品决定

产品用户确认继续把已验证原型中的“扫描与缓存任务中心”转换为后端优先的开发规格。首个
范围只包含扫描和可重建派生缓存任务，不包含系统维护、数据库备份、诊断包或智能媒体能力。

该决定不修改冻结 MVP、`POST-MVP-1` 或 `POST-MVP-2`。为避免没有 scope budget 的功能
插入当前发布，目标暂记为 `POST-MVP-3`；只有 `OPS-001` 接受 scope manifest 后才冻结。

## 价值与范围

- 统一展示高层 scan/derived runs、状态、阶段、计数、历史与允许操作；
- 复用既有 scanner、thumbnail、jobs 状态机；
- 增加 missing/all derived batch 的 durable parent run 和有界 admission；
- 提供取消、重试、ETag 详情和稳定 keyset 历史；
- 不暴露逐资产 `media_jobs`，不提供清空失败或任意并发调整。

## 架构影响

- 候选新增 `internal/operations`，只拥有跨 capability 投影、筛选、公共 task ID 和动作 dispatch。
- `internal/scanner` 继续拥有 scan；`internal/thumbnail` 拥有 derived run；
  `internal/jobs` 拥有 lease/claim/fairness/retry；SQLite 只实现这些接口。
- 预计需要只追加 operation-run migration；不修改已发布 migration。
- 不增加部署单元、外部数据库、消息队列、WebSocket 或任意媒体写入能力。
- 预期无需 ADR；若 S0 改变核心技术、任务一致性、事务 owner、信任或持久化边界，则先新增 ADR。

## 安全与可靠性

- 所有写 operation 需要认证、CSRF 与适用的幂等键；
- API 只接受 ID 和受控枚举，不接受路径；
- offline、失败、取消、盘满和重启保留可靠索引与 ready cache；
- rebuild 不先清空旧缓存，新文件仍使用临时发布和原子替换；
- 任务错误只使用稳定 code 和脱敏 message。

## Gate 与决定

- 原型：已完成，可继续作为交互参考。
- S0 规格与 spike：Go。
- 生产 OpenAPI/migration/backend：Blocked，等待 `OPS-S0` 与 `OPS-S1`。
- 生产前端：Blocked，等待 `OPS-S2 Backend Evidence Ready`。
- 最终结论：Conditional Go。
