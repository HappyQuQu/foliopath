# POST-MVP-5 原生双架构证据入口

- 日期：2026-08-31
- 状态：**Implemented locally / remote evidence pending**
- 类型：已批准 POST-MVP-5 S2 证据入口与失败关闭维护
- Requirement：`FR-INT-001～010`、`NFR-INT-001～010`、`INT-403`
- 目标版本与阶段：`POST-MVP-5` revision 2，S2 Backend Evidence / S4 Release evidence
- Owner：release infrastructure / QA
- 关联 Gate：[INT-S2A](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)、
  [INT-S2B](../gates/POST-MVP-5/int-s2b-backend-evidence-ready.md)

## 变更

新增手动 `.github/workflows/intelligent-media-native.yml`，为同一 source SHA 提供原生 linux/amd64 与
linux/arm64 执行入口。矩阵固定 GitHub-hosted `ubuntu-24.04` x64 和 `ubuntu-24.04-arm` ARM64 runner，
并在运行开始时核对 kernel、machine、Go 和 Docker daemon architecture；任何身份不符都立即失败。

每个架构执行：

- 格式、架构、生成、lint、单元、集成和 E2E 仓库检查；
- production libvips adapter 测试；
- 两库 order-first 查询正确性/性能证据矩阵；
- 强制 4 CPU、10k 目录/100k 媒体容量基线。

workflow 只有 `contents: read`，支持直接 `workflow_dispatch` 和同仓只读 `workflow_call`，不配置 QEMU、
Docker platform override、push/pull_request 自动触发或写权限。失败时仍上传 identity、step outcome 和有界日志，保留 14 天；
`outcomes.complete` 只有全部步骤成功才为 true。

同一 workflow 现在还以 `paired-evidence` job 下载当次两份 artifact，并调用
`make verify-intelligent-media-native-evidence`。本地 verifier 强制：

- 恰好一份 amd64 与一份 arm64 identity，且 source SHA、workflow run ID/attempt 与验收输入一致；
- runner label、`uname` machine、Go arch 和 Docker daemon arch 精确匹配原生 x64/ARM64；
- `qemuAllowed=false`；repository、libvips、search matrix、capacity 及 identity 五项均为 `success`；
- 只有全部条件满足才原子写出 paired `result=passed` summary。

缺 artifact、重复架构、跨 run/attempt、QEMU、错误 runner 或容量失败均使 verifier/job 失败。paired summary
默认模式仍只证明这些 baseline 检查。另新增严格的
`make verify-intelligent-media-native-model-evidence`，要求每个架构同时提供 `model-evidence.json`，并验证：

- 同一最终 model package、质量集、ranking/tie fixture，且两个架构 Top-20 集合 digest 相同；
- 参考 ranking、质量与 tie fixture 均通过，数值最大绝对误差不超过 `1e-3`；
- 10 万媒体/1 万目录、RSS 不超过 3.2 GiB、语义 P95/P99 不超过 750/1500 ms、普通浏览退化不超过
  20%、派生数据不超过 500 MiB；
- index rebuild、restart recovery 与 runtime load/cancel/kill/ENOSPC/offline/unload/long-soak failure matrix
  的聚合结果均通过。

质量摘要、Top-20、数值、runtime 与 index 五份聚合报告不是只填 hash 字符串：严格 verifier 从每个原生
artifact 目录实际打开普通文件并复算 SHA-256，拒绝绝对路径、目录逃逸、任一路径组件 symlink、非普通
文件和内容篡改。合法审核数据本体仍不上传；其 governed manifest 只通过固定 hash 与质量摘要绑定。

严格模式只校验结构化结果与跨架构绑定，不生成模型运行结果；当前 baseline workflow 不会调用未获准的
最终模型。只有 ADR、最终 package 与 production runtime 获准后，由原生 job 真实生成这些文件，才能把
严格 verifier 的通过结果用于 `INT-403`。

## 失败关闭与证据口径

workflow 文件和本地 architecture test 只证明入口没有明显漂移，**不证明双架构已运行或通过**。只有同一
source SHA 的 amd64/arm64 两份实际成功 artifact 经对应 Gate 复核，才能用于 `INT-403`。失败后 artifact
只用于诊断，不能因为“上传成功”升级为通过。

2026-09-02 在不合并默认分支的前提下，为默认分支已经注册的 `dockerhub.yml` 增加 branch-ref bridge：
手动输入精确 sentinel `intelligent-media-native-evidence` 时，发布 job 由互斥条件强制跳过，只以
`contents: read` 调用同一 commit 的 native workflow。GitHub 官方的
[手动运行合同](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manually-run-a-workflow?tool=cli)
允许 `workflow_dispatch --ref` 选择非默认分支，[复用工作流合同](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows)
规定同仓 `./.github/workflows/...` 使用调用者相同 commit。architecture test 和
`actionlint v1.7.7` 同时锁定“不发布”条件与 reusable workflow 关系。

该桥接只解决“workflow 未注册因而无法调度”的入口问题；baseline 运行仍不产生最终 model evidence，
也不会把候选质量、供应链或批准标成通过。

同日只读
[远端就绪审计](../evidence/int-001/native-remote-readiness-audit-2026-08-31.md)确认：`origin/aifeature`
仍停在现有 HEAD，当前工作树有 213 个 modified/untracked path，GitHub 远端只有 Docker 发布 workflow，
没有智能媒体 workflow 或匹配 artifact。故当前不是“待触发”，而是尚无可运行的远端候选提交。

## 验证

- `go test ./tests/architecture/...`：通过；
- `go test ./tests/release/intelligent_media_native_evidence ./tests/architecture -count=1`：通过；
- `actionlint v1.7.7` 校验两个 workflow：通过。

完整仓库验证在本记录落地后执行并单独如实报告。
