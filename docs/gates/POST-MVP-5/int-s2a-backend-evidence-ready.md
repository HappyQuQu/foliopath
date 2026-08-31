# INT-S2A：模型基础与图片语义搜索 Backend Evidence Ready

- 日期：2026-08-28
- 目标版本：`POST-MVP-5` revision 1
- 范围：A 模型基础 + B 图片语义搜索后端
- 复审任务：`INT-216`
- 当前判断：**No-Go**

## 结论

S2A 已实现到当前获批合同与可用输入允许的边界，但不能签署 Backend Evidence Ready，也不授权生产
消费者 UI。阻塞原因不是继续补普通 Go/SQLite 代码即可解决：production text tokenizer/encoder 仍受
未接受的 ADR-0014 阻断，内建 release catalog 仍为空，合法代表性质量集和原生 Linux/amd64 runner
尚未到位。

继续增加 Linux/arm64 合成排列不会改变该判断。后续只允许解除下列精确阻塞、修复已批准实现中的
真实缺陷，以及维护现有证据；不得开始 C 标签建议、D 视频搜索、E 人脸人物库或生产 AI UI。

该判断现由 `make arch-check` 可执行保护：ADR-0014 仍为提议且本 Gate 为 No-Go 时，生产 composition
必须使用空 reviewed catalog、保留已批准的 Semantic 管理/回填/清理边界，并且不得注入
`SemanticSearch`。Gate 转换必须在同一变更中显式更新该 fitness check，不能静默接通搜索 route。

## 已实现且通过定向证据的后端边界

- migration 22～24：model、operation、generation、embedding/progress、durable backfill/clear 与
  hashed-idempotency request 状态。
- 固定 `/models:ro` 有界扫描、opaque candidate、reviewed catalog fail-closed、managed/direct 安装、
  activation、availability refresh 与完整 managed orphan 恢复。
- managed publication 原子 rename、真实进程强杀、真实 Linux/arm64 `ENOSPC`、direct nested mount、
  corruption/restart/recovery 与诊断脱敏。
- production ORT image `Run`、context termination、single-resident generation session、hard timeout、
  idle unload、无敏感标识的 resident/active/load/run/unload 资源计数，以及 Linux/arm64 native inference
  process-kill/reload。
- libvips semantic image preprocessing、固定 tensor/embedding codec、SQLite binary16 权威行、source
  fingerprint 失效、短事务批量提交和 generation isolation。
- durable semantic backfill/clear、fair multi-library claim、lease/retry/cancel/restart、exact vector
  scope、coverage snapshot、stable encrypted cursor、bounded asset projection、auth/CSRF/rate-limit 和
  query/log 脱敏。

## 阻塞矩阵

| Blocker | 阻塞任务 | 解除证据 | 未解除时行为 |
| --- | --- | --- | --- |
| ADR-0014 仍为 Proposed；package v2 与 SentencePiece production owner 未获接受 | `INT-203/207/208` | 接受 ADR；production parser/runtime/FD lifecycle 与生成合同进入正式包 | semantic search route 保持不注册 |
| 最终审核 catalog/model、许可和供应链签署缺失 | `INT-202/203/209/214/215` | 固定 package digest、书面再分发结论、SBOM/VEX/notices/provenance 与 catalog entry | production catalog 为空；安装/激活失败关闭 |
| 合法代表性图片质量集缺失 | `INT-203/209/210` | 冻结数据 manifest 与 Top-20/中英质量报告 | 不宣称语义质量，不发布 Slice B |
| 原生 Linux/amd64 runner 缺失 | `INT-202/203/210/215` | 与 arm64 同合同的原生数值、取消、强杀、RSS/容量报告 | 不声明双架构完成 |
| 真实 100k×768 全进程容量未通过 | `INT-207/210` | 最终模型下 backfill + exact search + browse 并发、RSS、DB/WAL/backup 报告 | 不签 Backend Ready |
| 最终模型的完整 app backfill + SQLite + native inference 强杀链缺失 | `INT-209/215` | durable claim 后在 native Run 中 kill，新进程 lease recovery 且 active/embedding/media invariants 不变 | 仅承认各子边界证据 |

ADR-0014 已于 2026-08-29 按其六类接受门槛逐项复审，结论为
[保持提议 / Blocked](adr-0014-acceptance-audit-2026-08-29.md)：特殊 token 合同通过，其余五类仍为
Partial/Blocked。该审计明确禁止把 QEMU amd64、incomplete SBOM 或 tripwire 结果升级解释为接受证据。
同日的 [glibc 官方状态与当前 distroless 复核](../../evidence/int-001/glibc-security-status-refresh-2026-08-29.md)
确认双架构最新 `cc-debian13` 子镜像仍含 Debian 标记 vulnerable 的 `libc6 2.41-12+deb13u3`；没有可供
重建和重扫的 trixie 修复基座，供应链 blocker 不变。

2026-08-31 新增了手动
[`Intelligent media native evidence`](../../../.github/workflows/intelligent-media-native.yml) 入口和架构
fitness check。它会分别验证原生 x64/ARM64 runner identity，执行完整仓库检查、production libvips、
两库查询矩阵和强制容量基线，并拒绝 QEMU/平台覆盖。该入口尚未在当前 source SHA 上远程执行；文件存在
不构成 native amd64/arm64 证据，当前已知 production keyset 容量回归也会使 job 失败关闭。因此本 Gate
继续 **No-Go**。

后续 paired job 与 `make verify-intelligent-media-native-evidence` 已固定同 commit/run/attempt、原生
runner identity、无 QEMU、两架构齐全及全部 step success；容量失败无法生成 passed summary。该 verifier
的严格入口 `make verify-intelligent-media-native-model-evidence` 现已额外强制最终 model package/质量集、
跨架构 Top-20、`1e-3` 数值容差、3.2 GiB RSS、750/1500 ms 查询、20% 浏览退化、500 MiB 派生空间、
index rebuild/restart 和 runtime failure matrix。当前没有最终模型生成的结构化文件或真实远程 run，
因此只完善证据验收链，不解除本 Gate。

2026-08-31 的[远端就绪审计](../../evidence/int-001/native-remote-readiness-audit-2026-08-31.md)进一步确认
GitHub 远端尚无包含当前 S2/workflow 的候选提交，workflow 清单也没有智能媒体入口；最近运行不存在可配对
artifact。未经操作者明确授权不提交/推送/触发，因此 native blocker 仍是实际外部状态而非待轮询任务。

## 任务判定

- 完成：`INT-201/204/205/206/211/212/213`。
- 实现主体存在但被上述证据/决策阻塞：`INT-202/203/207/208/209/210/214/215`。
- `INT-216` 复审完成；“复审完成”不等于 Gate Go。

## 已执行检查

本轮复审引用 2026-08-28 已执行成功的：

- `make fmt`
- `make arch-check`
- `go test ./internal/inference/onnx ./internal/files ./internal/aimodel ./internal/app ./internal/store/sqlite -count=1`
- Linux/arm64 tagged direct-model nested-mount、managed-store real-ENOSPC 与 native ORT process-kill tests
- `git diff --check`

完整 `make lint/test/test-integration/test-e2e`、最终模型质量、100k 全进程、原生 amd64 和供应链签署没有
在本 Gate 执行或通过，因此判断必须保持 No-Go。

2026-08-28 后续维护已成功执行 `make test-race`、`make test-libvips` 与 `make spike-ai`；全仓 Go race、
固定源码 libvips production-tag 测试和隔离 AI spike 均通过。前述完整常规验证也已在后续轮次执行通过。
这些结果维护已实现边界的回归证据，不提供最终模型、合法质量集、原生 amd64、100k 全进程或供应链
签署，因此不改变本 Gate 的 No-Go 判断。

同日的合同消费者维护还执行了 `make openapi-lint`、`make contract-check` 与完整 `make web-check`；
OpenAPI 有效、契约通过，生成 client、前端架构/类型/单测和 Storybook 均通过。期间发现的 Web dependency
high advisory 已用兼容补丁关闭并记录于
[Web 依赖高危 advisory 修复](../../changes/FIX-2026-08-28-web-dependency-advisories.md)。该修复不注册
semantic search route，也不改变本 Gate 的模型、质量、双架构、容量或供应链阻塞。

随后 `make test-web-e2e` 也在真实 Docker 后端与锁定 Chromium / Pixel 5 项目下恢复通过（7 passed、
4 skipped），只读媒体 hash/path 前后相同；证据与 harness 修复记录于
[真实浏览器 E2E harness 与当前界面合同同步](../../changes/FIX-2026-08-28-browser-e2e-harness.md)。
这是现有合同消费者与非 AI 纵向回归证据，不是 S2A 最终模型或 semantic search route 证据，No-Go 不变。

同日新增的
[INT-S2A No-Go 生产组合 fitness check](../../changes/FIX-2026-08-28-int-s2a-no-go-fitness.md)
已随 `make arch-check` 通过；它把当前失败关闭状态变成回归约束，不解除任何阻塞。

语义搜索隔离服务随后增加 repository 返回页的 bounded/scope/finite/unique/order/continuation 校验；异常页
在生成公开 cursor 前失败关闭。HTTP hydration 也逐项复核 asset/library ID 与 online 状态，使删除、跨库
ID 复用或 offline 竞态不返回错误资产。定向证据见
[语义搜索边界失败关闭](../../changes/FIX-2026-08-28-semantic-search-result-validation.md)。该修复维护
已批准合同，但 production route 仍未注册，Gate 继续为 No-Go。

后续检修还通过 race 复现并修复共享 Raw URL Base64 cursor 的非规范文本别名，且让畸形 semantic
snapshot 以内部错误而非客户端错误失败关闭；证据见
[Cursor 规范编码与语义 snapshot 失败关闭](../../changes/FIX-2026-08-28-canonical-cursor-decoding.md)。
这不改变当前 Gate 判断。

共享 transport limiter 随后补齐 wall-clock 回拨恢复：保留已用额度并重基准固定窗口，避免 semantic
bucket 获得额外配额或满载表长期锁死。证据见
[限流时钟回拨恢复](../../changes/FIX-2026-08-28-rate-limit-clock-rollback.md)；Gate 仍为 No-Go。

2026-08-31 新增
[智能媒体最终供应链证据校验入口](../../changes/FIX-2026-08-31-intelligent-media-supply-chain-verifier.md)：
`make verify-intelligent-media-supply-chain` 会把最终 package/catalog、native 双架构 complete SBOM、签名
provenance、notices、漏洞/VEX 和四方批准绑定到真实文件 SHA-256。当前没有最终输入或签署，工具通过
自身单元测试不构成供应链证据，本 Gate 继续 No-Go。

最终复审还必须运行
[S2 最终证据交叉绑定](../../changes/FIX-2026-08-31-s2-evidence-binding.md)；
`make verify-intelligent-media-s2-evidence` 会拒绝 quality、strict native、supply-chain 的 source/model/image
digest 不一致，也拒绝 baseline native summary。当前没有三份真实 summary，故不改变本 Gate。

共享 AI JSON decoder 还修复了 4 KiB `LimitReader` 对超长尾随空白的 EOF 绕过，第 4,097 个字节现会在
任何 install/settings/backfill/clear 服务调用前失败关闭；合法 `application/json` 媒体类型参数按标准
解析，不再被字符串精确比较误拒。证据见
[AI JSON 请求体硬上限](../../changes/FIX-2026-08-28-ai-json-body-limit.md)；不改变 Gate。

模型激活、operation 取消和 semantic settings/clear 共用的强 ETag parser 随后拒绝 `r01`、`r+1`
等非规范 revision 别名，只让服务端实际签发的 canonical validator 进入 service。证据见
[AI 强 ETag 规范匹配](../../changes/FIX-2026-08-28-canonical-ai-etag.md)；Gate 仍为 No-Go。

公共 AI operation owner 还补齐持久化 state/phase/error-code 组合校验；Get/create/transition 的
repository 返回值全部复用该校验，并绑定请求/创建/转换的 operation identity、owner 和 revision；
损坏、adapter 缺陷或返回另一条合法 operation 都不会进入 worker/API，只返回脱敏内部错误。证据见
[AI operation 持久化状态校验](../../changes/FIX-2026-08-28-ai-operation-state-validation.md)；
安装/激活 admission 结果也在 ManagementService 出口复用 operation 校验，并绑定 kind、model 和
Created/Replayed 语义；不改变 Gate。

2026-08-29 又把 queue work 校验前移至 install/activation admission 的 `Wake()` 之前；candidate/source、
model/availability revision、idempotency/request 和初始 operation 任一不一致都不会唤醒 worker。证据见
[AI admission 唤醒前校验](../../changes/FIX-2026-08-29-ai-admission-prewake-validation.md)；Gate 不变。

随后检修确认 wake 只是可合并提示且 worker 有 durable queue 轮询；真正缺口是 install/activation worker
未复核 claim 返回值。现在两者会在安装器、模型文件源和 native runtime 前绑定 request hash、工作对象与
唯一合法的 claimed operation 形态，损坏 claim 直接使 component 失败关闭。证据见
[AI worker claim 返回值失败关闭](../../changes/FIX-2026-08-29-ai-worker-claim-validation.md)；Gate 不变。

同一终态链还统一了 worker failure 与并发取消的 CAS 收敛：operation owner 在 precondition 竞争后有限
重读一次，`cancelling` 优先完成为 cancelled，只有仍为 running 才写原失败码；安装和激活的全部错误
出口不再留下只能等重启处理的 active operation。证据见
[AI worker 终态 CAS 竞态收敛](../../changes/FIX-2026-08-29-ai-worker-terminal-cas-convergence.md)；Gate 不变。

## 下一次复审触发条件

只有以下输入发生变化才重新打开 `INT-S2A`：ADR-0014 被接受或否决、最终 catalog/model 获批、合法
质量集到位、原生 amd64 runner 可用，或完整容量/组合恢复证据产生。单纯重复 arm64 合成测试不触发复审。

2026-08-31 外部状态与非人脸容量输入再次复核：Debian trixie `libc6` 仍为
`2.41-12+deb13u3`；GitHub 仓库当前只有历史 main 发布 workflow 结果，没有包含本 worktree 的 native
amd64 S2 证据。production 100k keyset 基线还出现 296.212 ms 超预算，虽有快速且等价的
benchmark-only 候选，但尚未通过维护 Gate。以上事实维持本 Gate **No-Go**，不接通 text/search route。
