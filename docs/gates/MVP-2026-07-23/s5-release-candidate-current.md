# S5-009A Release Candidate 当前判断

## 结论

**No-Go — 当前提交不能进入 Release Candidate。**

`S5-009A` 已完成首次统一 RC 审计和失败关闭入口，但 `S5-009` 尚未完成。Stage 5 的八个
前置 Gate 中 `S5-002`、`S5-003`、`S5-004`、`S5-005` 与 `S5-008` 通过，另外三个仍有可验证的硬阻断；
八项发布阻断风险中五项处于缓解中、`R-008/R-011` 已关闭、`R-017` 仍开放，没有任何一项
有时限地正式接受。

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
| S5-006 质量矩阵 | Blocked | Safari 26.5.2 真机及 Chrome 150 normal/forced-colors 通过；真实 Firefox、读屏/缩放/触摸/移动物理设备签署仍缺失 |
| S5-007 供应链 | Blocked | 双架构 SPDX/notices 和 fail-closed provenance 入口已建立；仍须处置/逐项接受 1 Critical、8 High，并生成最终 clean-commit statement、完成安全/合规签署 |
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
`R-017` 在首个镜像发布前处置。当前快照为：

- `R-002/003/006/010/014`：`mitigating`，均有 owner 与下一条关闭条件；
- `R-008`：`closed`，原生双架构完整候选矩阵及运行闭包已审阅；
- `R-011`：`closed`，原生双架构升级与配对回滚证据已归档；
- `R-017`：`open`，1 Critical / 8 High 尚未消除或逐项正式接受；
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

## 允许的下一步

只允许推进上表的关闭条件、发布阻断缺陷、安全/数据完整性修复及其文档证据。当前尚未进入
RC feature freeze 状态；当 manifest 首次满足 `release-ready` 后，仍需人工审阅证据并完成
`S5-009`，不能由自动化自行宣布 RC 或稳定发布。`S5-010` 必须继续保持未完成。
