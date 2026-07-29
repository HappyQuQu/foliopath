# Gate POST-MVP-2 / WCH-S0 / Architecture Ready

- 日期：2026-07-29
- 目标版本：`POST-MVP-2`
- Roadmap 切片：`WCH` 媒体库自动发现
- 需求：`FR-SCN-010～014`、`NFR-REL-002`、`NFR-PERF-003`
- 验收：`WCH-AC-001～012`
- 决策角色：产品负责人（产品用户）；架构/capability/安全与数据负责人（FolioPath maintainers）
- Change Record：[CR-2026-005](../../changes/CR-2026-005-automatic-library-discovery.md)
- Scope：[POST-MVP-2 revision 1](../../releases/POST-MVP-2-scope.md)
- Feature：[FTR-SCN-001](../../features/automatic-library-discovery.md)
- Spike：[WCH-001](../../spikes/wch-001-linux-watcher.md)
- ADR：[ADR-0011（已接受）](../../adr/0011-linux-inotify-hints-and-anchored-reconciliation.md)
- 风险：R-002、R-003、R-005、R-013、R-016、R-019
- 结论：**Go**

## 输入完整性

| S0 输入 | 结论 |
| --- | --- |
| 目标版本与稳定需求 | `POST-MVP-2` revision 1 已冻结；不改变 MVP/POST-MVP-1 |
| 用户结果与非目标 | 自动近实时发现、完整扫描兜底及网络盘/新服务非目标已固定 |
| Capability owner | `internal/scanner`；files/jobs/store/API/Web 边界明确 |
| 数据与 API 影响 | durable incremental queue、content revision、settings/library 状态；精确合同归 S1 |
| 正常/边界/失败/恢复 | `WCH-AC-001～012` 与质量属性场景已定义 |
| Fixture 与可行性 | WCH-001 在 Linux/arm64 观察完整事件、ELOOP 安全拒绝、root replacement、真实 overflow 和 10k watches；amd64 模拟按 ENOSYS 失败关闭 |
| 资源与 fallback | R-019 已登记；每目录 watch、有界合并、degraded/full-scan/disable fallback 已固定 |
| ADR 判断 | ADR-0011 已接受；事件只作 hint，增量清理资格和完整扫描兜底已固定 |

## 已收敛架构方向

1. Linux 目录事件只是 hint；`internal/scanner` 通过 `internal/files` 锚定定向校准后才写索引。
2. delete/unmount 不直接清理；只有根身份和父目录可靠枚举成功才能清理明确范围。
3. overflow、ENOSPC、watch invalidation、权限或 I/O 错误进入 degraded/offline 并合并完整扫描。
4. 每目录而非每文件 watch；所有事件、dirty、任务、事务和刷新路径有界。
5. 增量工作进入 `internal/jobs` 所有的 durable lease 协议；完整 generation 仍是最终一致基线。
6. 独立 content revision 只触发客户端重新获取，不替代 cursor generation/revision。
7. 前端首版轻量轮询，不增加 WebSocket；共享 query/invalidation 只有一个 owner。

## 剩余证据分配

- WCH-S1 必须冻结 durable queue 复用/新表、content revision、full-scan 并发、错误码和资源上限。
- WCH-S2 必须补原生 linux/amd64、受控 ENOSPC、mount namespace unmount/nested mount、
  慢写稳定窗口、强杀和生产组合证据。
- 10k watches 实测为约 273ms、RSS +2,364 KiB、FD 不增长；只能作为 S1 候选预算输入。
- amd64 模拟层 `openat2` 返回 ENOSYS 并失败关闭，不记录为原生 amd64 通过。

## 下一步获准范围

允许执行 WCH-S1 Contract Ready：

- 固定 capability use cases、watch/degraded 状态机、事件合并和删除确认合同；
- 修改并评审权威 OpenAPI 源；
- 选择只追加 migration 和 durable queue/content revision 事务模型；
- 冻结 app/内核资源上限、错误码、可观察性与测试 fixture；
- 记录 `WCH-S1 Contract Ready`。

本 Gate 仍不授权生产 watcher、migration 执行、handler 或 UI。只有 WCH-S1 Go 后才能开始
后端实现，只有 WCH-S2 Go 后才能开始前端接入。
