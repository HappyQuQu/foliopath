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

当前不是“workflow 已在远端、只差点一次运行”，而是**尚无包含该 workflow 和当前 S2 代码的远端候选
提交**。因此不能取得同 source SHA 的 native amd64/arm64 evidence，也不能运行 strict model 或最终 S2
aggregation。历史 CI/发布镜像属于不同 commit 和产品范围，不能复用。

解除顺序：

1. 由操作者明确授权并形成可审查的候选提交/推送；
2. 最终审核模型、合法质量集与供应链签署到位；
3. 在同一候选 SHA 上运行原生双架构 workflow 并生成严格 model evidence；
4. 依次运行 quality、native-model、supply-chain 与最终 S2 aggregation verifier；
5. 由对应 owner 复审 Gate。

本审计只确认远端状态，不构成上述任一授权或证据。
