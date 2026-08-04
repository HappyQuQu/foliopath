# POST-MVP-1 Scope Manifest — Revision 8

Revision 8 完整继承 [revision 7](POST-MVP-1-scope-r7.md)及其全部安全、非目标和验收约束，
并通过 [CR-2026-018](../changes/CR-2026-018-explicit-resource-limits.md)修订
`FR-SET-001`、`NFR-PERF-004`：

- 配置页不再要求用户在三档资源模式之间选择；
- 用户直接设置后台任务并发数 1～4，以及原图/视频读取并发数 1～16；
- 默认仍为 2 / 8，既有档位迁移到等价数字；
- 后端硬上限、共享后台预算、独立内容读取预算及降低限制不取消现有工作保持不变；
- 不增加逐库并发、无界并发、带宽整形、部署单元或外部队列。

- 版本：`POST-MVP-1`
- Scope revision：`8`
- 状态：`Scope Frozen`
- 冻结日期：2026-08-04
- 产品负责人：产品用户
- Scope-budget exception：用户明确接受的既有切片交互简化；不提高资源上限或扩大架构边界
