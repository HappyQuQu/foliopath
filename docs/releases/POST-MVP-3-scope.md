# POST-MVP-3 Scope Manifest revision 1

- 状态：Frozen / In Delivery
- 冻结日期：2026-08-03
- 产品确认：本次对话中的“开搞”确认
- Change Records：[CR-2026-008](../changes/CR-2026-008-task-center.md)、
  [CR-2026-015](../changes/CR-2026-015-operations-observability-and-updates.md)

## 范围

- `FR-OPS-001～006`：高层完整扫描、派生补齐/重建任务中心；
- `FR-OPS-007`：完整扫描与派生失败恢复是两个清晰、独立操作；
- `FR-OPS-008`：安全、分页、可筛选的日志与媒体失败诊断；
- `FR-OPS-009`：管理中心关于页和构建版本展示；
- `FR-OPS-010`：稳定版本检查与用户友好发布历史；
- `FR-OPS-011`：全局消息中心聚合活动、需处理、完成和更新提醒；
- `NFR-OPS-002～004`：任务、诊断、轮询、更新检查和保留策略有界且失败降级。

## 非目标

- 应用内自动升级容器、写 Docker socket 或执行部署命令；
- 浏览器访问原始 stdout/stderr、SQL、stack trace 或宿主路径；
- 删除任务历史、失败事实或审计信息来实现“清除消息”；
- WebSocket、新 worker 服务、Redis 或外部数据库。

## 交付顺序

失败诊断与有界恢复 → 关于/发布信息 → 日志页 → 消息聚合 → parent task history 与完整
missing/all batch。每一纵向切片都先更新 OpenAPI、实现后端并通过证据，再连接生成客户端 UI。

