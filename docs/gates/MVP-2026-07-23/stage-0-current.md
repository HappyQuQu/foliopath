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
  - [S0-105 Gate allocation record](s0-105-gate-order.md)
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
  - FS-04：Linux/arm64 四核、4 GiB、10 万媒体/1 万目录扫描/索引子范围通过，结论仍为
    Conditional
  - `api/openapi.yaml` 已为 HTTP 结构权威；不依赖 Ruby/Node/网络的 Go YAML 解析、引用/
    结构/ECMAScript pattern、AST/Schema 与 scanner/migration 选择性关键不变量检查
    （`queued`、`animated`、可空 `startedAt`）通过；这不是完整领域实现一致性证明。Redocly
    无结构错误，保留两条 health endpoint 4xx 规则 warning
  - Node/npm 版本与 lockfile 已固定；TypeScript 类型和唯一 Web API client 可确定性生成，
    strict typecheck、npm high-severity audit、摘要锁与语义兼容自比较在本地通过
  - [PR #1 CI](https://github.com/HappyQuQu/foliopath/actions/runs/29985018814) 的原生
    amd64/arm64 Go/race、合成媒体、mount-boundary 和 Web contract 7 个 jobs 全部通过；
    Web contract 对真实 base branch 的语义兼容比较通过
- 未关闭条件：
  - FS-03：ICC/敌意输入与资源限制、生产任务隔离、浏览器直放、双架构最终镜像
  - FS-04：代表性存储、RSS、完整媒体/缩略图、FTS/keyset、生产 HTTP/前端并发与趋势
  - FS-05：双架构镜像、非 root/只读挂载、健康检查、备份恢复、升级和许可证追踪
  - 契约/工程：`sqlc` 生成、`test-e2e`；完整运行骨架、生产 handler、认证、React 产品前端
    和可发布容器均不存在
- Fallback：失败时缩小已承诺的平台、格式或规模并记录变更/Gate；不得降低原文件只读、路径
  失败关闭、失败扫描不清理旧索引或认证边界
- 结论：`Conditional Go`
- Conditional Owner 与复审触发：FS-03/媒体处理负责人、FS-04/技术负责人、
  FS-05/发布负责人、契约与 CI/架构负责人；上述缺失证据写回对应报告、风险和本 Gate 后复审。
  FS-01 的生产 HTTP/auth 要求转入首个受保护 API Backend Gate，发布挂载/运行期故障转入
  FS-05/Release Gate，仍由安全负责人签署
- 下一步获准范围：剩余 spike、完善生成/兼容性/CI 契约护栏、架构检查，以及不进入生产
  import graph且不扩大信任边界的时间盒实验脚手架；**不允许产品功能开发、生产 feature
  handler、业务 UI 或稳定发布**
