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
make release-docs-check
make release-readiness-check
make contract-check
make generate-check
make lint
make test
make test-integration
make test-e2e
```

这些入口均已存在；`generate-check` 同时覆盖固定版本 sqlc 与 OpenAPI TypeScript 产物。
当前 `test-e2e` 是真实后端应用的测试专用容器 smoke，不代表浏览器产品 E2E 或正式发布
镜像验收；缺失层次必须明确报告，不能被已有入口替代。

## Fitness function 清单

| ID | 不变量 | 检查方式 | 当前状态 | 最晚落地时间 |
| --- | --- | --- | --- | --- |
| AF-001 | Go 依赖只向内，handler 不依赖 SQLite/files，能力包不依赖 adapter/app | `make arch-check` 解析实际 Go import graph | **本地与 CI 已执行**：首次原生 amd64/arm64 PR CI 通过 | 持续强制 |
| AF-002 | 不出现无所有权的 `utils/common/helpers/base` 或无外部消费者的 `pkg/` | `make arch-check` 检查 Go package | **本地与 CI 已执行**：首次原生 amd64/arm64 PR CI 通过 | 持续强制 |
| AF-003 | 冻结 scope revision 不被静默改写；每项工作关联需求、目标版本、capability 与 Gate | scope manifest/revision、Change Record、Gate/PR 链接检查 | 计划门禁 | 首个产品 PR 前 |
| AF-004 | OpenAPI 是唯一 HTTP 结构契约，生成客户端无漂移 | `make contract-check`、OpenAPI lint、生成 diff、摘要锁、兼容性检查 | **部分执行**：离线 Go 契约检查、确定性 TypeScript 生成、唯一 client、摘要锁、Redocly、语义自比较和真实 PR base-branch 比较均通过；认证、媒体库与扫描 operation 已有真实 HTTP 证据，S3-001 已锁定 browse root/breadcrumb/scope/cursor/offline 结构，生产 browse handler 仍按 S3 Backend Gate 验证 | 持续强制 |
| AF-005 | 已发布 migration 只追加且可从空库／上一版本升级 | migration checksum、升级测试、外键和完整性检查 | **部分执行**：正式应用已有空目录/重复迁移、冲突 schema 失败关闭、外键和完整性测试；追加认证 migration 已验证单管理员、摘要、期限与级联约束。migration checksum 与真实上一版本升级仍待实现 | 首个预览版前 |
| AF-006 | 所有媒体路径经过唯一策略和 `internal/files`，不越界、不泄露 | 恶意路径矩阵、Linux mount/openat2、HTTP E2E | **媒体库 Backend Ready**：Darwin 与原生 Linux amd64/arm64 路径矩阵、同/跨设备及 self-bind mount、认证 path/create HTTP 与真实 composition、TOCTOU/ABA、权限和错误脱敏均通过；扫描/媒体读取 handler 继续逐切片强制，发布 volume/unmount 由 Release Gate 强制 | 按 S0-105 持续分层强制 |
| AF-007 | 失败、取消、离线或中断扫描绝不清理旧索引 | generation 故障矩阵、重启与竞态测试 | **扫描故障矩阵已自动化**：FS-02、S2-103～106 已验证失败、取消、offline、部分不可读、nested mount、根替换、重启和 SQLite 满页不发布不可靠代次或清理旧索引；startup scan 与晚到期 lease 自动恢复，解除满页限制后的完整扫描可收敛。生产强杀与代表性存储故障仍由 Release Gate 补齐 | 持续，发布前补强杀/存储故障 |
| AF-008 | 后台任务、数据库写入和媒体工具全部有界 | 队列/并发配置测试、压力指标、取消与超时测试 | **S5-005 已通过**：扫描固定 256 active/256 batch/2 worker；媒体固定 2 worker/3 attempt、govips concurrency 1/64 MiB cache。原生目标档关闭 session touch、unchanged fingerprint 与 cache 淘汰写放大，并以 100k 全量派生、80% 水位和持续健康验证边界 | 持续强制并在发布前复测 |
| AF-009 | 前端依赖方向、共享组件唯一所有权与 token 单一来源 | TypeScript boundary/cycle lint、受限基础库/import 位置 allowlist、token lint、组件目录检查 | **本地与 CI 入口已建立**：strict TypeScript、唯一生成 client、禁止散落 `fetch`、app/routes/features/components/lib 依赖方向、共享组件目录、CSS token 与颜色字面量由 `npm run check:architecture` 强制；认证首个业务 feature 通过该门禁 | 持续强制 |
| AF-010 | 前端稳定原语在主题、语言、宽度和异步状态下行为一致 | component workbench build、Testing Library、axe、聚焦视觉回归 | **S5-006A 候选自动化已建立**：既有 Storybook/Testing Library、简中/英文、浅/深主题、390/768/1024/1440 与 Chromium axe 继续执行；Firefox/WebKit 新增查看器键盘、焦点、降级状态、overflow 与 axe 矩阵，固定 Linux Chromium 深色/reduced-motion 视觉基线已独立复跑。最终浏览器版本、Safari 真机和物理设备签署仍待 S5-006B | 行为/axe 在首次消费前；视觉基线在 API 稳定或第二消费者前；完整矩阵在 RC 前 |
| AF-011 | 大列表只使用游标分页和统一虚拟化模式 | API 契约测试、SQLite query 测试、前端 pattern 测试、E2E DOM/请求上限 | **后端查询已执行**：目录/资产默认 50、最大 200，query/generation-bound opaque keyset、完整 tuple、唯一 ID tie-breaker、context cancellation 与生产 SQLite 禁用 `OFFSET` 已由 S3-001/002 固定；S4-002 固定库内 generation 与跨库 catalog-revision search cursor、global name tuple 和 mtime tuple，S4-003 又以 100k 两页 keyset 和取消收敛验证预算。前端统一虚拟化和 E2E DOM/请求上限仍待对应前端 Gate | 浏览/搜索切片持续强制 |
| AF-012 | 单容器、非 root、`/library:ro`、本地 `/app/data` 运行 | 双架构容器 smoke、安全挂载和健康检查 | **候选执行、发布签署待完成**：S5-001A/001B 已用真实 SPA/Go/libvips/FFmpeg 候选验证 UID/GID 65532、只读根/媒体、cap-drop、health、SIGTERM 和安全 Compose；S5-002 又在本机原生 arm64 与指定原生 amd64 服务器关闭完整运行矩阵。S5-001C 的 CI JSON 合同仍拒绝跨 run 拼接；最终不可变 digest 尚待供应链与发布签署 | 首个预览镜像前 |
| AF-013 | 认证、会话、CSRF 与代理信任覆盖全部业务 API | 路由清单测试、安全 E2E、配置测试 | **S5-003 通过**：S1 认证 Integrated Done 加上显式 CIDR、非回环 require-proxy、严格单跳 HTTPS transport、Secure Cookie/Origin/HSTS/客户端限流集成矩阵；Compose 实际网络拓扑已由 S5-001B/002 在双架构复验 | 首个可共享预览版前 |
| AF-014 | 备份、恢复、升级、磁盘满和强杀不破坏不可重建数据 | 故障注入与恢复演练 | **S5-004 通过**：原生 arm64/amd64 已验证离线恢复、强杀/WAL、满盘、损坏失败关闭，以及不同不可变候选间的向前升级和旧镜像＋升级前备份配对回滚 | Release Candidate 前 |
| AF-015 | 目标规模内资源和交互不越过实测预算 | 10 万媒体／1 万目录／4 核／4 GiB 基准与趋势比较 | **S5-005 已通过**：原生 arm64/amd64 的 4 CPU/4 GiB 候选通过 100k/10k 扫描、认证查询、重扫、取消和 offline；指定 amd64 服务器完成 100k 全量派生、cache 水位及稳定性，三引擎 100k FPS/RSS 通过 | 发布前持续复测 |
| AF-016 | 镜像依赖、许可证与漏洞可追溯 | SBOM、license policy、镜像扫描 | **部分执行、RC 阻断**：[S5-007A/G](../gates/MVP-2026-07-23/s5-supply-chain-candidate.md) 已为生产候选建立确定性 source/npm/image SPDX、固定 digest Trivy、双架构 notices 和 in-toto/SLSA provenance 入口；修复来源 GLib 的本机 arm64 候选已达到 `0 Critical / 0 High`。最终原生双架构 clean-commit provenance、全阻断复扫和安全/合规签署未完成 | Release Candidate 前 |
| AF-017 | SQLite 查询以 adapter 内的 SQL 源为唯一事实，生成代码不可手改或漂移 | sqlc 固定版本、临时目录重生成 diff、生成标记与 adapter 重复 SQL 检查 | **本地执行，CI 已接线**：媒体库及 scan claim/lease/recovery 查询由 `queries/` 生成到 `dbgen/` 并被 adapter 消费；复杂 generation finalize 仍在 adapter 内受事务测试约束 | 持续强制 |
| AF-018 | RC 决策与 Stage 5 Gate、风险登记和证据不得矛盾，未处置项必须失败关闭 | `make release-readiness-check` 校验机器快照；`make release-ready` 强制全部 Gate passed、风险 closed/accepted | **S5-009A 已执行、当前 No-Go**：八个前置 Gate 中 S5-002/003/004/005/008 passed；六项发布风险 mitigating、R-008/R-011 closed。普通一致性检查通过，promotion 入口按预期非零失败 | Release Candidate 前持续强制 |

## 前后端门禁顺序

一个产品切片只有在前一层检查通过后才进入下一层：

1. **S0 Architecture Ready**：AF-003 与适用 ADR／风险检查。
2. **S1 Contract Ready**：AF-004、数据迁移设计与错误／认证语义。
3. **S2 Backend Ready**：后端单元、契约、集成、故障与安全测试。
4. **S3 Frontend Ready**：AF-009～AF-011，以及全部适用 UI 状态。
5. **S4 Integrated Slice Done**：当前切片的 E2E、适用容器/故障与追踪证据。

`Release Ready` 是版本级 Gate：所有版本内切片完成后，再统一执行 AF-012～AF-016、全量 E2E、
恢复、容量、安全、合规与发布文档门槛。

Stage 0 Gate 已通过，Stage 1 认证后端和 Stage 2
[媒体库管理](../gates/MVP-2026-07-23/s2-library-backend-ready.md)已经 Backend Ready。
扫描已由 `S2-102～107` 补齐 worker、HTTP、设置、故障和容量证据并达到 Backend Ready；
浏览、媒体读取及正式发布容器继续按各自 Gate 阻断。一个切片通过不能
把相邻切片自动标记为 `Backend Ready`，也不能跳过后端直接实现依赖未完成行为的业务 UI。

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
