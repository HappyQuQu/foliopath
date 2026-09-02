# POST-MVP-5 原生证据远端就绪审计（2026-08-31）

## 目的

只读确认当前 `aifeature` 是否已经有可运行的远端候选提交、智能媒体原生 workflow 或同 source SHA 的
artifact。该审计不提交、不推送、不触发 workflow，也不把历史 CI/发布镜像结果解释为 POST-MVP-5 证据。

## 实际检查

在 `/Users/qu/Documents/GitHub/foliopath` 执行：

```sh
git branch --show-current
git rev-parse HEAD
git rev-parse --abbrev-ref --symbolic-full-name '@{u}'
git ls-remote --heads origin aifeature
git status --porcelain=v1 | wc -l
gh workflow list --all --json id,name,path,state
gh run list --limit 30 --json databaseId,name,workflowName,headSha,status,conclusion,event,createdAt,url
```

结果：

- 本地分支为 `aifeature`，HEAD 为 `6c2a9b21beff24fc2e7fae44eba01743379e7c9e`；
- upstream 为 `origin/aifeature`，远端同名分支也仍指向该 HEAD；
- 工作树有 213 个 modified/untracked path，POST-MVP-5 当前实现与 evidence workflow 不属于该远端 SHA；
- GitHub 远端 workflow 清单只有 `.github/workflows/dockerhub.yml`（`Publish Docker image`）；
- 最近 30 个 run 只有旧 `CI` 与 `Publish Docker image`，没有 `Intelligent media native evidence`，也没有
  与当前未提交 POST-MVP-5 工作树匹配的原生双架构 artifact。

## 判断

## 2026-09-01 只读复核

远端状态已经比上述首次审计前进一步：本地 `HEAD` 与 `origin/aifeature` 均为
`fdede8c99b3dcef52cc2d9851551521fc0340652`，且 `git ls-tree` 证明该远端提交已包含
`.github/workflows/intelligent-media-native.yml`。`gh workflow list` 默认只列默认分支 workflow，不能再据此
断言分支文件不存在。

但当前工作树仍有 96 个 modified/untracked path，包含后续 S2C 后端、严格 evidence verifier 和 Gate 证据；
它们不属于远端候选 SHA。GitHub 最近 30 个 run 仍只有旧 `CI` 与 `Publish Docker image`，从未运行
`Intelligent media native evidence`，也没有可下载的 paired native artifact。远端 workflow 当前只产生 baseline
native summary，不产生最终模型所需的 `model-evidence.json`，因此即使对旧 SHA 运行，也不能满足
`verify-intelligent-media-native-model-evidence` 或最终 S2 aggregation。

当前准确结论是：**baseline workflow 骨架已在远端分支，但当前 S2 增量、最终审核模型/质量输入及严格模型
证据仍不在可运行的同一候选提交中，且没有对应 run**。历史 CI/发布镜像属于不同 commit 和产品范围，
不能复用。

解除顺序：

1. 由操作者明确授权并形成包含当前 S2 增量的可审查候选提交/推送；
2. 最终审核模型、合法质量集与供应链签署到位；
3. 在同一候选 SHA 上运行原生双架构 workflow 并生成严格 model evidence；
4. 依次运行 quality、native-model、supply-chain 与最终 S2 aggregation verifier；
5. 由对应 owner 复审 Gate。

本审计只确认远端状态，不构成上述任一授权或证据。

## 2026-09-01 workflow 注册复核

进一步通过 GitHub API 复核 workflow 的实际可调度状态：

- 仓库默认分支为 `main`，其 HEAD 为 `f2d6e4863e4a2298340c4ae0ec6cea1f5d36b981`；
- 本地 HEAD 与 `origin/aifeature` 均为 `fdede8c99b3dcef52cc2d9851551521fc0340652`；
- API 可从 `aifeature` 读取 `.github/workflows/intelligent-media-native.yml`，blob SHA 为
  `63a8735980b6fe7b4867916714225b1522335a74`；
- 默认分支注册的 active workflow 仍只有 `Publish Docker image`；按文件名查询/调度
  `intelligent-media-native.yml` 返回 HTTP 404；
- `aifeature` 的 Actions run 列表中没有任何该 workflow 的运行记录；
- 当前工作树另有 107 个 modified/untracked path，仍不是远端候选提交。

因此“workflow 文件存在于功能分支”不等于 GitHub 已注册可手动调度的 workflow。要获得真实原生证据，
入口必须先经正常审查进入默认分支，或由默认分支上已注册且等价受控的 workflow 调用目标 ref；该外部状态
变化仍需要操作者授权，不能由本地 verifier 或文档替代。即使注册完成，最终模型 evidence 生成、合法质量
输入和供应链签署仍是独立前置条件。

## 2026-09-01 收口复核

再次只读查询 GitHub API：默认分支仍仅注册 `dockerhub.yml`，仓库级 self-hosted runner 数为 `0`，最近运行
仍没有智能媒体 workflow。`aifeature` 与 `origin/aifeature` 仍同为
`fdede8c99b3dcef52cc2d9851551521fc0340652`，而当前工作树共有 131 个 modified/untracked path，故刚通过的
人脸 production boundary、format v3 proposal、严格 verifier 与 Gate 更新均没有同 SHA 的远端执行入口。
结论和解除顺序不变；本次复核未提交、未推送、未触发 workflow。

## 2026-09-02 候选人脸原生预检入口

当前工作树已把固定 YuNet/AuraFace/公开 JPEG 的完整 production-boundary candidate smoke 加入
`intelligent-media-native.yml`，并扩展 paired verifier 严格读取每个架构的 `face-candidate.json`。入口会
拒绝 machine/Go/Docker arch 不一致，在模型 SHA 校验后断网、只读、有界运行，只记录 candidate count 与
单向量化指纹；它不能生成最终 `model-evidence.json`，三个批准 flag 固定为 false。

这仍只是本地未提交增量：默认分支未注册该版本 workflow，当前 source SHA 没有远端 run 或 artifact。
即使未来两个 native job 都通过，也只关闭候选 ABI/functional 预检，不关闭最终模型质量、`1e-3` 成对
数值报告、100k 联合容量、供应链或 owner 签署。原解除顺序不变。

2026-09-01 follow-up：候选镜像还会在每个原生 runner 的 4 CPU/4 GiB 限额中运行 100k × 512 合成
人脸聚类，并由 paired verifier 校验 `face-capacity.json` 的 workload、非质量 flags 和确定性指纹。该步骤
尚未在远端执行；即使通过也只是聚类子容量，不是最终模型 + SQLite + HTTP/browse 的联合 100k 证据。

## 2026-09-01 最终本地收口复核

再次只读执行 `git ls-remote`、`gh workflow list --all` 与 `gh run list`：默认分支仍为 `main`，HEAD
`f2d6e4863e4a2298340c4ae0ec6cea1f5d36b981`；本地 HEAD 与 `origin/aifeature` 仍同为
`fdede8c99b3dcef52cc2d9851551521fc0340652`。GitHub 仅注册 active 的 `Publish Docker image`，最近 20 次
运行没有智能媒体 workflow。当前工作树共有 168 个 modified/untracked path（排除生成的
`internal/webassets/dist/` 后为 167），所以最新 S2A/B/C 实现、候选原生入口与严格 verifier 仍没有可引用的
远端 source SHA。

本轮还直接执行全部五个最终入口；它们分别以缺少 `QUALITY_INPUT`、`FACE_QUALITY_INPUT`、`EVIDENCE_DIR`、
`SUPPLY_CHAIN_INPUT`、`QUALITY_SUMMARY` 失败关闭。该结果证明入口没有用默认/合成数据伪造通过，也确认当前
没有可聚合的最终质量、原生模型、供应链和 S2 summary。解除顺序与前述判断不变。
