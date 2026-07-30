# Gate MVP-2026-07-23 / UIF-S0 / Architecture Ready

- 日期：2026-07-30
- 目标版本：`MVP-2026-07-23` revision 4
- Feature：[FTR-UIF-001](../../features/frontend-prototype-fidelity.md)
- Change Record：[CR-2026-009](../../changes/CR-2026-009-frontend-prototype-fidelity.md)
- 需求：`FR-AUTH-005`、`FR-BRW-010`、`FR-UI-009～010`、`NFR-UIF-001`
- 验收：`UIF-AC-001～012`
- 风险：R-010、R-012、R-015、R-016、R-021
- 决策角色：产品负责人（产品用户）；FolioPath 架构/capability/API/安全/数据/QA owners
- 结论：**Go — 仅授权 Contract Ready 与共享视觉基础**

## 输入完整性

| S0 输入 | 结论 |
| --- | --- |
| 产品范围 | revision 4 冻结；任务中心、系统维护和 AI 明确排除 |
| 视觉真相 | 原型、QA 截图、UI 规范和四档视口已指定 |
| Capability owner | auth、catalog、thumbnail 与 web shared 唯一 owner 已固定 |
| API 缺口 | account、direct-directory q、cache summary/cleanup 已识别，精确 wire 归 S1 |
| 数据影响 | 优先复用现有表；新索引/run 只允许追加 migration |
| 安全 | 当前密码、session revoke、CSRF、缓存只删派生、原媒体只读 |
| 容量 | 100k/10k、四核/4 GiB；目录 query 和 cache cleanup 必须有界 |
| 验收 | UIF-AC-001～012、failure matrix、视觉 comparison 与完整任务清单已定义 |
| ADR | 无新技术/部署/信任/持久化类型/依赖方向；当前不需要 |

## 架构决定

1. 生产视觉使用现有 React/Vite 与共享设计系统，不复制原型 CSS 到 feature。
2. 业务规则继续由 auth/catalog/thumbnail 拥有；HTTP、SQLite 和 Web 只消费窄接口。
3. 目录关键字必须是可靠索引查询，不允许客户端加载全部 cursor 页。
4. 缓存清理只调用现有 LRU/cache owner，不并入 Post-MVP 任务中心。
5. 账户修改使用现有 Argon2id 和 session 模型，成功撤销其他会话。
6. 原型是视觉真相，OpenAPI/Backend Ready 是功能真相；冲突先修正规格。
7. 共享壳可在 S1/S2 同期实现，但真实新操作必须等待 S2。

## 未关闭条件

- 账户 operation 的 revision/ETag、错误与事务窗口尚未冻结；
- direct-directory `q` 的查询计划和是否需要只追加索引尚未测量；
- cache cleanup 的同步/异步表示和重启语义尚未冻结；
- 视觉 reference manifest 和生产 comparison automation 尚未实现；
- R-021 在 UIF-S4 前保持开放。

## 下一步获准范围

允许执行 `UIF-101～109` 和 `UIF-301～306`：

- 固定账户、目录 query、缓存合同；
- 完成 10k 目录查询 spike；
- 修改并评审 OpenAPI；
- 决定只追加 migration；
- 建立 reference manifest、token 映射和共享壳 Storybook。

本 Gate 不授权新业务 handler、migration 执行、生产账户修改、全量目录过滤或缓存清理 UI。
