# Gate MVP-2026-07-23 / Stage 0 / Current

- 日期：2026-07-23
- 目标版本：`MVP-2026-07-23`
- Scope revision：1
- Roadmap 阶段：0
- 需求：冻结 manifest 中全部 FR/NFR；本 Gate 只判断能否进入产品实施
- 决策职责：产品范围已由用户确认；架构证据由当前仓库评审；发布签署尚不适用
- 输入与证据：
  - [Scope manifest](../../releases/MVP-2026-07-23-scope.md)
  - [开发就绪评审](../../development-readiness.md)
  - [FS-01 路径边界](../../spikes/fs-01-path-boundary.md)
  - [FS-02 SQLite/generation](../../spikes/fs-02-sqlite-generation.md)
  - [FS-03 媒体矩阵](../../spikes/fs-03-media-matrix.md)
  - [FS-04 容量基线](../../spikes/fs-04-capacity-baseline.md)
  - [FS-05 运行与恢复](../../spikes/fs-05-runtime-recovery.md)
  - [供应链与许可证审查](../../supply-chain-review.md)
  - [S0-105 Gate allocation record](s0-105-gate-order.md)
  - [S0-106 容量证据 Gate 分配记录](s0-106-capacity-gate-order.md)
  - [S0-109 风险复审](s0-109-risk-review.md)
  - [权威 OpenAPI 契约](../../../api/openapi.yaml)
  - [架构适配度检查](../../architecture/fitness-functions.md)
  - [测试策略](../../testing-strategy.md)
  - [风险登记](../../risk-register.md)
- 已验证事实：
  - FS-01：Darwin 与原生 Linux amd64/arm64 路径矩阵、Linux `openat2` 同/跨设备及 self-bind mount
    拒绝、真实 HTTP test harness 的 Stage 0 路径可行性范围通过
  - FS-02：当前 SQLite/generation 正确性 scope 通过
  - FS-03：合成 FFmpeg 子矩阵和隔离 govips/libvips 图片 fixture 已在原生 amd64/arm64
    通过，S0-103/S0-104 关闭；完整结论仍为 Conditional
  - FS-04：Linux/arm64 四核、4 GiB 的扫描/索引主档、Linux RSS 与三档趋势通过；
    S0-106 的 Stage 0 可行性范围关闭，完整产品/发布结论仍为 Conditional
  - FS-05：同一 Debian slim Dockerfile 的原生 linux/amd64、linux/arm64 构建和
    runtime/recovery/failure fixture 通过；已验证非 root、零 capability、只读根和
    `/library`、健康检查、优雅退出、完整数据卷离线恢复、重复 migration 与磁盘满/只读/
    损坏 DB 失败关闭
  - 供应链：固定 Syft digest，可重复生成 source/npm/image SPDX 2.3；关键依赖可追踪，
    Debian FFmpeg 的 GPL、libx264/libx265 组合义务已被明确记录
  - `api/openapi.yaml` 已为 HTTP 结构权威；不依赖 Ruby/Node/网络的 Go YAML 解析、引用/
    结构/ECMAScript pattern、AST/Schema 与 scanner/migration 选择性关键不变量检查
    （`queued`、`animated`、可空 `startedAt`）通过；这不是完整领域实现一致性证明。Redocly
    无结构错误，保留两条 health endpoint 4xx 规则 warning
  - Node/npm 版本与 lockfile 已固定；TypeScript 类型和唯一 Web API client 可确定性生成，
    strict typecheck、npm high-severity audit、摘要锁与语义兼容自比较在本地通过
  - [PR #1 CI](https://github.com/HappyQuQu/foliopath/actions/runs/29985018814) 的原生
    amd64/arm64 Go/race、合成媒体、mount-boundary 和 Web contract 7 个 jobs 全部通过；
    Web contract 对真实 base branch 的语义兼容比较通过
  - [Stage 0 最终 CI](https://github.com/HappyQuQu/foliopath/actions/runs/29990480565)
    覆盖双架构 Go/race、mount、govips/FFmpeg、FS-05 runtime/recovery、契约生成/兼容与
    SBOM/license；该 run 必须为 success，否则本 Gate 自动回退为 `Conditional Go`
- 已延期且继续阻断后续 Gate 的条件：
  - FS-03：ICC/更多敌意输入与资源限制、生产任务隔离、浏览器直放、最终发布镜像
  - FS-04 后续 Gate：代表性存储/最终镜像；完整媒体队列、FTS/keyset、生产 HTTP 和前端并发
  - FS-05 后续范围：在线备份、真实已发布版本升级、运行期 NAS 断连、最终 multi-arch
    manifest、镜像体积/漏洞/attestation
  - 契约/工程：`sqlc` 生成、`test-e2e`；生产运行骨架、feature handler、认证、React 产品
    前端和可发布容器仍不存在，它们是 Stage 1～5 的交付物，不是 Stage 0 的伪前置条件
- Fallback：失败时缩小已承诺的平台、格式或规模并记录变更/Gate；不得降低原文件只读、路径
  失败关闭、失败扫描不清理旧索引或认证边界
- 结论：`Go — 允许进入 Stage 1`
- 决策边界：
  - 只授权 [任务清单](../../task-list.md) 的 Stage 1，且严格后端优先：先运行骨架，再认证
    后端；认证 `Backend Ready` 前不得实现认证产品 UI
  - 不授权 Stage 2～5、共享预览、稳定发布或公网暴露；每个纵向切片仍必须独立经过
    `Architecture Ready → Contract Ready → Backend Ready → Frontend Ready → Integrated Done`
  - FS-01 生产 HTTP/auth 由首个受保护文件 API Backend Gate 强制；发布挂载/运行期 unmount
    由 Release Gate 强制
  - FS-04 生产容量证据按 S0-106 分配到 Backend、UI、Performance/Release Gate
  - S0-109 中每个开放/缓解中风险由其 Owner 在最迟 Gate 提交证据；不得把本次 Go 解释为
    风险已关闭
- 下一步获准范围：`S1-001`～`S1-106` 的后端工作，以及仅在认证 Backend Ready 后开始的
  `S1-201`～`S1-206`。任何新增需求先走 scope change，不直接堆叠到 MVP。
