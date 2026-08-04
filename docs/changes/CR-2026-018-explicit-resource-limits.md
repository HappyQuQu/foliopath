# CR-2026-018：直接设置资源并发数

## 状态与范围

- 状态：Confirmed / Implemented locally
- 变更等级：C2（用户可见设置、API 与实例级资源策略）
- 目标版本：`POST-MVP-1` / `Post-MVP/1`
- Scope revision：[POST-MVP-1 revision 8](../releases/POST-MVP-1-scope-r8.md)
- 基线事件：2026-08-04 产品用户确认
- 需求：`FR-SET-001`、`NFR-PERF-004`
- Owner：`internal/resourcecontrol`（实例级并发策略）、`internal/settings`
  （持久化与更新编排）、`internal/app`（组合与启动恢复）
- 合同：`Settings.backgroundConcurrency`、`Settings.contentReadConcurrency`、
  migration 20
- 证据：动态收缩/扩展/取消测试、SQLite default/CHECK、设置 HTTP、OpenAPI 与生成
  client、Web 表单与保存测试、仓库格式/架构/生成/测试检查

## 决定

配置的“扫描策略”页签不再展示 NAS 友好、均衡、性能三档资源模式，改为两个普通整数输入：

| 设置 | 默认值 | 允许范围 |
| --- | ---: | ---: |
| 后台任务并发数 | 2 | 1～4 |
| 原图与视频读取并发数 | 8 | 1～16 |

后台任务并发仍由同一个实例级预算统一覆盖完整扫描、定向校准和媒体派生；内容读取仍使用
独立预算。降低数值不取消正在执行或传输的工作，新工作等待现有持有者释放许可。既有 worker、
libvips native concurrency、durable queue、lease、重试、扫描 generation 和原媒体只读语义
不变。服务端和 SQLite 继续强制既有硬上限，界面不能配置无界并发。

本记录替代 [CR-2026-012](CR-2026-012-nas-resource-profiles.md) 的三档交互与
`Settings.resourceProfile` wire 合同；migration 16 保留为历史迁移，migration 20 将已有档位
无损映射为 1/4、2/8 或 4/16。该变更不改变部署单元、信任边界、事务所有权、任务一致性或
依赖方向，因此不需要新 ADR。
