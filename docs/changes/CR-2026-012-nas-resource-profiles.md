# CR-2026-012：NAS 资源模式

## 状态与范围

- 状态：Confirmed / Implemented locally
- 变更等级：C2（用户可见设置与实例级资源策略）
- 目标版本：`POST-MVP-1` / `Post-MVP/1`
- Scope revision：[POST-MVP-1 revision 4](../releases/POST-MVP-1-scope-r4.md)
- 基线事件：2026-07-31 产品用户确认
- 需求：`FR-SET-001`、`NFR-PERF-004`
- Owner：`internal/resourcecontrol`（实例级并发策略）、`internal/settings`
  （持久化与更新编排）、`internal/app`（组合与启动恢复）
- 合同：`Settings.resourceProfile`、migration 16 `settings.resource_profile`
- 证据：profile/动态收缩/取消测试、SQLite 默认值与 CHECK、设置 HTTP、生成 client、
  Web type/test/Storybook、仓库格式/架构/生成/测试检查

## 决定

管理中心“扫描与缓存”提供三个预设，不暴露每种内部任务的任意并发数字：

| 模式 | 共享后台媒体源操作 | 原图/视频内容读取 |
| --- | ---: | ---: |
| `eco`（NAS 友好） | 1 | 4 |
| `balanced`（均衡，默认） | 2 | 8 |
| `performance`（性能） | 4 | 16 |

共享后台预算同时覆盖完整扫描、自动发现定向校准和缩略图/视频派生，防止各能力分别未超限
但合计压垮 NAS。内容读取保留独立上限，避免长时视频流占尽后台任务许可。降低模式不取消
正在运行或传输的工作；新工作等待现有持有者释放许可。durable queue、lease、重试、扫描
generation 和原媒体只读语义不变。

不提高既有每类 worker、libvips native concurrency 或内容读取硬上限；不增加带宽整形、
逐库并发、任意自定义数字、新部署单元或外部队列。该变更不需要 ADR，因为没有改变部署、
信任、持久化所有权、任务一致性或依赖方向。
