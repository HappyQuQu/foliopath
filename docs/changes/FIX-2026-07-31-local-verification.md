# FIX-2026-07-31：关闭自动 CI，改为本地验证

- 类型：已批准的交付策略调整
- 关联质量：`NFR-COMP-001`、`NFR-OPS-001`
- 目标版本与阶段：MVP / 持续维护
- Owner：根 `Makefile` 与变更提交者
- 合同：`AGENTS.md`、`docs/testing-strategy.md`、本地验证命令
- 关联风险：R-016
- 不变量：不得因关闭自动 CI 而跳过适用验证；Docker Hub 凭据仅用于发布 workflow

## 行为

删除 PR、`main` push 和手动触发的通用 CI workflow，避免持续消耗 GitHub Actions
配额。格式、架构、生成一致性、lint、单元、集成和 E2E 检查继续由根 `Makefile`
提供，开发者在合并与发布前于本地执行并记录实际结果。

Docker Hub 多架构发布 workflow 保留，在推送到 `main`、推送版本标签或管理员手动运行时
触发。普通 PR 不触发；Actions 分钟只用于镜像发布，不再用于通用测试 CI。

## 风险与证据

- 自动 required check 不再阻止未经验证的合并，R-016 保持“缓解中”。
- 历史 CI、双架构和 Gate 证据不重写；它们只证明当时记录的提交与环境。
- 本次变更以 `make arch-check`、`actionlint` 和适用本地检查的实际结果为证据。
