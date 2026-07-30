# S5-009A Release Candidate 当前判断

## 结论

**No-Go — 当前提交不能进入 Release Candidate。**

`S5-009A` 已完成首次统一 RC 审计和失败关闭入口，但 `S5-009` 尚未完成。Stage 5 的八个
前置 Gate 中 `S5-002`、`S5-003`、`S5-004`、`S5-005` 与 `S5-008` 通过，另外三个仍有可验证的硬阻断；
八项发布阻断风险中六项处于缓解中、`R-008/R-011` 已关闭，没有开放或正式接受项；缓解中
风险仍未达到发布关闭条件。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-009A`、`S5-009`
- 需求/质量：冻结 scope 的全部 FR/NFR 与 `MVP-AC-001～008`
- owner：发布负责人拥有 RC 决策快照；各 Gate/risk owner 仍拥有关闭条件和原始证据
- 输入：`docs/task-list.md`、`docs/risk-register.md`、全部 `s5-*.md` Gate、
  `docs/testing-strategy.md`
- 机器快照：`docs/releases/MVP-2026-07-23-rc-readiness.json`
- 自动化：`tests/architecture/release_readiness_test.go`、
  `make release-readiness-check`、`make release-ready`
- 架构影响：只增加版本级 Gate 聚合和 fitness check，不改变产品、部署或信任边界

## RC 前置 Gate

| Gate | 当前状态 | 关闭条件摘要 |
| --- | --- | --- |
| S5-001 最终镜像 | Blocked | 从干净提交重建，生成 provenance，并签署最终不可变 digest |
| S5-002 双架构矩阵 | Passed | 操作者指定原生 amd64 服务器与本机原生 arm64 均通过完整矩阵；CI 计费失败不冒充运行结果 |
| S5-003 发布 HTTP 安全 | Passed | 已完成；最终 Compose 仍由 S5-001/002 复核 |
| S5-004 恢复/升级 | Passed | 原生 amd64/arm64 均以不同前一候选 image ID 通过向前升级及旧镜像＋升级前备份配对回滚 |
| S5-005 产品容量 | Passed | 原生 amd64 通过真实媒体和 100k/10k 目标档、100k 全量派生、cache 低水位及持续健康；本机三引擎 FPS/RSS 通过并冻结预算 |
| S5-006 质量矩阵 | Blocked | Safari 26.5.2 真机、Chrome normal/forced-colors、五桌面项目 200% 等效重排及 Chrome 151 物理 Mac 原生 200% 纵向链通过；真实品牌 Firefox、物理读屏/触摸/移动设备、Safari/Firefox 缩放和最终视觉签署仍缺失 |
| S5-007 供应链 | Blocked | 干净候选提交 `5c3b3c7` 的原生 arm64 已绑定完整 smoke、不可变 digest、SPDX/notices、provenance 和 `all` 策略 `0 Critical / 0 High`；仍缺同提交原生 amd64、paired summary 与安全/合规签署 |
| S5-008 发布文档 | Passed | 已完成并由 `make release-docs-check` 防漂移 |

任一 Blocked 项都足以保持 No-Go，不允许以总通过率、候选本机结果或“CI 已接线”替代证据。
2026-07-28 对当前已提交 HEAD 的 GitHub Actions run `30314930003` 复核确认：账户付款
失败或 spending limit 使全部 job 在 runner 分配前失败。`S5-001C` 已补齐同 commit/run
双架构 JSON 证据契约。操作者已指定原生 amd64 服务器替代本轮计费阻断的 CI，因此
`S5-002` 已由实际双架构矩阵关闭；`S5-001` 仍由最终不可变 digest 和供应链签署阻断。

同日人工传输的候选已在原生 linux/amd64 完整通过 runtime matrix，并以 31,899 项真实
ZFS 媒体发现并关闭 FFprobe 参数缺陷；证据见
[S5-002A/S5-005C](s5-native-amd64-real-media.md)和
[S5-005D](s5-native-amd64-capacity.md)。后续服务器原生构建与完整 smoke 已和本机 arm64
结果配对，关闭 `S5-002`；同一服务器随后完成全量媒体吞吐和 cache 水位复测，关闭
`S5-005`。最终物理浏览器签署仍未完成。

## 发布阻断风险

风险登记明确要求 `R-002`、`R-003`、`R-006`、`R-008`、`R-010`、`R-011`、`R-014`、
`R-017` 在首个镜像发布前处置。2026-07-30 的最新快照为：

- `R-002/003/006/010/014`：`mitigating`，均有 owner 与下一条关闭条件；
- `R-008`：`closed`，原生双架构完整候选矩阵及运行闭包已审阅；
- `R-011`：`closed`，原生双架构升级与配对回滚证据已归档；
- `R-017`：`mitigating`，干净候选提交的原生 arm64 已为 `0 Critical / 0 High` 并
  生成绑定证据；原生 amd64 配对复扫与最终签署尚未完成；
- `closed`：2；
- `accepted`：0。

正式接受不是把状态文字改为 accepted。每个接受项必须记录具体 CVE/包或残余风险、适用
版本与部署范围、可达性证据、决策者、到期时间和复审/撤销条件。严重安全、原媒体写入、
路径逃逸或不可恢复数据损坏不能用一般风险接受绕过。

## 失败关闭自动化

机器快照只聚合当前决策；各 capability Gate 和风险登记仍是原始事实来源。

- `make release-readiness-check` 校验完整八 Gate、八项发布风险、owner、证据链接、关闭条件
  以及汇总结论一致；准确记录 No-Go 时该检查应通过。
- `make release-ready` 使用相同 manifest，但要求所有前置 Gate 为 passed 且所有发布阻断
  风险为 closed/accepted。当前命令必须非零失败。
- `make arch-check` 默认包含一致性测试，因此删除阻断条件、丢失证据或把 No-Go 误写为 Go
  会阻断普通 CI。

2026-07-28 本机结果：

```text
make release-readiness-check
ok github.com/HappyQuQu/foliopath/tests/architecture

make release-ready
FAIL: release candidate is No-Go; unresolved gates and risks remain
```

第二个失败是本 Gate 的预期安全结果，不是未处理的测试故障。

## 2026-07-30 供应链续审

原生 linux/arm64 已从干净提交 `5c3b3c7` 重建并通过完整 release smoke、SPDX、
notices、SLSA provenance 与 Trivy `all` 零发现验证，镜像 digest 为
`sha256:8a88d26b6579afea21e4d3d85a1df7b5d45b5f851466c4afd6067d025516457d`。
GitHub Actions
[run 30551526321](https://github.com/HappyQuQu/foliopath/actions/runs/30551526321)
的原生 amd64/arm64 job 在执行任何 step 前因账户付款失败或 spending limit 被拒绝，
因此 paired job 跳过。该运行不改变 Gate 结论：它既不是产品失败，也没有提供原生
amd64 通过证据；`S5-007` 与 Release Candidate 继续 No-Go。

## 2026-07-30 浏览器质量续审

新增的 200% 等效重排护栏已在 Chromium、Firefox、WebKit、品牌 Chrome Stable 与
forced-colors 通过，覆盖媒体卡焦点入口、查看器主焦点、缩放/信息/关闭控件、页面横向
溢出和 axe serious/critical。Google Chrome 151 随后在物理 Mac 以原生 `200%` 页面
缩放完成真实候选的扫描、浏览、预览、Viewer、快捷键和媒体缩放纵向链，并确认只读挂载
及媒体 SHA-256 不变，证据见
[`docs/evidence/s5-006b`](../../evidence/s5-006b/README.md)。Mozilla 官方 Firefox
153.0.1 下载在当前网络反复断流，没有形成可校验的品牌版应用，因此真实 Firefox 纵向链
仍未执行；物理读屏、触摸、移动设备、Safari/Firefox 真实缩放和最终视觉签署也仍缺失。
`S5-006` 状态保持 Blocked。

## 允许的下一步

只允许推进上表的关闭条件、发布阻断缺陷、安全/数据完整性修复及其文档证据。当前尚未进入
RC feature freeze 状态；当 manifest 首次满足 `release-ready` 后，仍需人工审阅证据并完成
`S5-009`，不能由自动化自行宣布 RC 或稳定发布。`S5-010` 必须继续保持未完成。
