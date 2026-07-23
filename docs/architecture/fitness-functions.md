# 架构适配度检查与质量门禁

## 状态

本文定义系统架构如何由可重复检查持续约束。状态分为：

- **本地已执行**：仓库已有可运行入口，并在当前环境实际通过；若 CI 尚未建立，不代表持续强制。
- **部分执行**：已有局部测试，但尚未覆盖目标平台或完整场景。
- **计划门禁**：必须在相关产品代码合入前落地，当前不能声称通过。

架构规则必须同时具备规范、所有者、检查入口和失败处理。只写在文档中的规则是设计约束，
但在自动检查落地前不视为已完整执行。

## 统一入口

根 `Makefile` 是本地与 CI 的共同入口：

```sh
make arch-check
make contract-check
make generate-check
make lint
make test
make test-integration
make test-e2e
```

除 `test-e2e` 外这些入口已存在；`generate-check` 当前覆盖 OpenAPI TypeScript 产物，尚未
覆盖未来的 `sqlc`。缺失入口必须明确报告，不能被已有 Go 测试替代。

## Fitness function 清单

| ID | 不变量 | 检查方式 | 当前状态 | 最晚落地时间 |
| --- | --- | --- | --- | --- |
| AF-001 | Go 依赖只向内，handler 不依赖 SQLite/files，能力包不依赖 adapter/app | `make arch-check` 解析实际 Go import graph | **本地与 CI 已执行**：首次原生 amd64/arm64 PR CI 通过 | 持续强制 |
| AF-002 | 不出现无所有权的 `utils/common/helpers/base` 或无外部消费者的 `pkg/` | `make arch-check` 检查 Go package | **本地与 CI 已执行**：首次原生 amd64/arm64 PR CI 通过 | 持续强制 |
| AF-003 | 冻结 scope revision 不被静默改写；每项工作关联需求、目标版本、capability 与 Gate | scope manifest/revision、Change Record、Gate/PR 链接检查 | 计划门禁 | 首个产品 PR 前 |
| AF-004 | OpenAPI 是唯一 HTTP 结构契约，生成客户端无漂移 | `make contract-check`、OpenAPI lint、生成 diff、摘要锁、兼容性检查 | **部分执行**：离线 Go 契约检查、确定性 TypeScript 生成、唯一 client、摘要锁、Redocly、语义自比较和真实 PR base-branch 比较均通过；生产 request ID、统一安全错误、health/readiness/status handler 已有契约形状测试，其余业务路由实现一致性尚未证明 | 首个 handler 前 |
| AF-005 | 已发布 migration 只追加且可从空库／上一版本升级 | migration checksum、升级测试、外键和完整性检查 | **部分执行**：正式应用已有空目录迁移、重复启动、冲突 schema 失败关闭、外键和完整性测试；migration checksum 与真实上一版本升级仍待实现 | 首个预览版前 |
| AF-006 | 所有媒体路径经过唯一策略和 `internal/files`，不越界、不泄露 | 恶意路径矩阵、Linux mount/openat2、HTTP E2E | **Stage 0 范围已执行**：Darwin 与原生 Linux amd64/arm64 路径矩阵、同/跨设备及 self-bind mount、HTTP harness 已通过；生产 handler/auth 由首个受保护 API Backend Gate 强制，发布 volume/unmount 由 FS-05/Release Gate 强制 | 按 S0-105 持续分层强制 |
| AF-007 | 失败、取消、离线或中断扫描绝不清理旧索引 | generation 故障矩阵、重启与竞态测试 | 部分执行；FS-02 当前 scope 通过 | 持续，发布前补强杀/磁盘故障 |
| AF-008 | 后台任务、数据库写入和媒体工具全部有界 | 队列/并发配置测试、压力指标、取消与超时测试 | 计划门禁 | 对应 worker 合入前 |
| AF-009 | 前端依赖方向、共享组件唯一所有权与 token 单一来源 | TypeScript boundary/cycle lint、受限基础库/import 位置 allowlist、token lint、组件目录检查 | **部分执行**：strict TypeScript、唯一生成 client 边界及禁止散落 `fetch` 的架构测试已通过；React、组件/token/cycle 门禁尚未建立 | 首个业务 feature 前 |
| AF-010 | 前端稳定原语在主题、语言、宽度和异步状态下行为一致 | component workbench build、Testing Library、axe、聚焦视觉回归 | 计划门禁 | 行为/axe 在首次消费前；视觉基线在 API 稳定或第二消费者前；完整矩阵在 RC 前 |
| AF-011 | 大列表只使用游标分页和统一虚拟化模式 | API 契约测试、前端 pattern 测试、E2E DOM/请求上限 | 计划门禁 | 浏览切片前 |
| AF-012 | 单容器、非 root、`/library:ro`、本地 `/app/data` 运行 | 双架构容器 smoke、安全挂载和健康检查 | **Stage 0 probe 已执行**：原生双架构 FS-05 通过；正式应用镜像仍待验证 | 首个预览镜像前 |
| AF-013 | 认证、会话、CSRF 与代理信任覆盖全部业务 API | 路由清单测试、安全 E2E、配置测试 | 计划门禁 | 首个可共享预览版前 |
| AF-014 | 备份、恢复、升级、磁盘满和强杀不破坏不可重建数据 | 故障注入与恢复演练 | **部分执行**：FS-05 离线恢复、重复 migration、只读/满盘/损坏失败关闭通过；在线备份、强杀和真实版本升级未测 | Release Candidate 前 |
| AF-015 | 目标规模内资源和交互不越过实测预算 | 10 万媒体／1 万目录／4 核／4 GiB 基准与趋势比较 | **部分执行**：Linux/arm64 tmpfs 的扫描/索引子范围通过；完整媒体/HTTP/前端与代表性存储未测 | 阶段 0 FS-04 与发布前复测 |
| AF-016 | 镜像依赖、许可证与漏洞可追溯 | SBOM、license policy、镜像扫描 | **部分执行**：source/npm/image SPDX 与关键 codec/license 审查通过；最终 digest attestation、漏洞与 notices 未完成 | Release Candidate 前 |

## 前后端门禁顺序

一个产品切片只有在前一层检查通过后才进入下一层：

1. **S0 Architecture Ready**：AF-003 与适用 ADR／风险检查。
2. **S1 Contract Ready**：AF-004、数据迁移设计与错误／认证语义。
3. **S2 Backend Ready**：后端单元、契约、集成、故障与安全测试。
4. **S3 Frontend Ready**：AF-009～AF-011，以及全部适用 UI 状态。
5. **S4 Integrated Slice Done**：当前切片的 E2E、适用容器/故障与追踪证据。

`Release Ready` 是版本级 Gate：所有版本内切片完成后，再统一执行 AF-012～AF-016、全量 E2E、
恢复、容量、安全、合规与发布文档门槛。

Stage 0 Gate 已通过并只授权 Stage 1。生产 handler、认证/错误 envelope 和正式发布容器证据
仍缺，必须按切片 Gate 顺序补齐；本次 Stage 0 Go 不能把真实文件读取 API 标记为
`Backend Ready`，也不能跳过后端直接实现业务 UI。

## 失败与豁免

- 安全边界、原文件只读、失败扫描不清理旧索引、认证保护、迁移完整性不可豁免。
- 其他检查若因工具故障临时跳过，必须记录原因、Owner、到期时间和补跑条件；不得以“本地看起来正常”代替。
- 基准退化必须先复现并判断是否属于环境噪声；确认退化后修复、回退或通过 Change Record 调整已实测预算。
- 架构检查误报应修正规则或提交 ADR，不允许长期散布行内禁用注释。

## CI 分层

- 每个 PR：格式、生成漂移、AF-001/002、静态检查、单元和受影响集成／组件测试。
- 主分支：全量 race、契约、E2E、组件工作台和聚焦视觉回归。
- 候选版本：双架构镜像、目标容量、安全扫描、强杀／磁盘故障、备份恢复和升级演练。

具体测试数据和发布阻断规则仍以[测试策略](../testing-strategy.md)为权威；本文负责把它们与系统架构不变量对应起来。
