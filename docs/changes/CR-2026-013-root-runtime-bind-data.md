# CR-2026-013：root runtime 与零初始化 bind 数据目录

## 状态与范围

- 状态：Confirmed / implementation in progress
- 变更等级：C3（容器信任边界与发布合同）
- 目标版本：`MVP-2026-07-23` / Stage 5 release candidate
- Scope revision：不增加用户能力；修订 `FR-DEP-001～004` 的运行身份合同
- 基线事件：2026-08-01 产品操作者明确要求 root runtime、bind data、无 Compose user
- Owner：发布与运维；实现 owner 为根 `Dockerfile`、`compose.yaml` 和 `internal/app` smoke
- 合同：Dockerfile/Compose 均不声明用户覆盖，镜像 OCI 用户为 `0`、
  `./data:/app/data`、`/library:ro`、ADR-0012
- 证据：架构检查、release image smoke、Compose smoke、双架构 Docker Hub 构建

## 决定

FolioPath 正式镜像默认以 root 运行，使 Docker 自动创建的 root-owned `./data` bind 目录
无需权限初始化即可写入。Compose 与 README 不添加 `user`，不改用 named volume。
原媒体仍必须以 `/library:ro` 单次挂载，路径与 mount-crossing 失败关闭规则不变。

该变更使既有非 root 候选证据失效；新的 root 镜像通过当前 smoke 和目标架构复验前，Release
Candidate 继续 No-Go。风险与缓解记录在 ADR-0012、安全模型和 R-022。
