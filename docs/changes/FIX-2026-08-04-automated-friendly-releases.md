# FIX-2026-08-04：自动版本与用户友好更新日志

## 状态

Accepted / Implemented

## 归属

- 需求：`FR-DEP-001～004` 发布与部署
- 目标：当前候选发布工具链；不改变产品功能范围
- Owner：根 GitHub workflow、Release Please 配置与发布文档
- Gate：既有 Stage 5 发布文档、双架构镜像与供应链 Gate

## 问题

原流程在 `main` 上只发布 `latest`/`sha-*`，正式版本号与版本文件需要人工创建，且更新日志
容易直接复制技术提交。版本、GitHub Release 和 Docker 标签可能因此分离。

## 决定

- `main` push 继续自动发布 Docker `latest` 与 `sha-*`。
- Release Please 根据 Conventional Commits 维护可审核的 Release PR。
- 合并 Release PR 后自动创建语义化版本、GitHub Release 和对应 Docker 版本标签。
- 用户日志固定使用带少量 Emoji 的“新功能、改进、修复、注意事项”分类；纯技术类型隐藏。
- 不连接、不重启、不更新任何实际部署实例。

## 证据

- `tests/architecture/TestDockerHubPublicationKeepsTagAndPlatformBoundaries`
- `tests/architecture/TestFriendlyReleaseAutomationContract`
- `make arch-check`
- `make release-docs-check`
