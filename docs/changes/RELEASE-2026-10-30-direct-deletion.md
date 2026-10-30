# 原文件删除与文件管理发布

2026-10-30；用户授权“更新readme 提交推送合并发布”。范围：直接永久删除、一次确认、后台文件任务与暂停/继续、默认只读与显式可写覆盖；目录操作/右键、全部展开收起、删除完成刷新与消息中心结果；媒体库选择记忆。回收站和输入确认词按用户决定撤回。

中英文 README 和 Docker Hub 概览同步当前操作方式、隐藏/非媒体范围、不可恢复、单媒体根及外部程序移动约束。保留已知自动发现反复扫描问题与原处理指引。没有 AI 模型/身份变化，不修改正式版本数字或手写生成更新日志，由 Release Please 确定下一版本。新增 migration 49–50，旧稳定版回退必须恢复升级前备份。

沿既有 PR → main → Release Please → 原生验证 → 同 digest 推广流程；DELETE-S4 证据见[验收记录](../gates/POST-MVP-DELETE/delete-s4-release.md)。不提交普通测试、截图、日志、生成 Vite 产物、运行数据库或缓存，不扩大 25 项发行输入白名单。

## 发布前验证

`CGO_ENABLED=0 GOMAXPROCS=4 make fmt-check arch-check generate-check lint test test-integration`、`make test-e2e`、固定 Node 24.22.1/npm 11.19.1 的 `npm --prefix web run check` 已通过；运行日志在本地 `/tmp/foliopath-direct-release`。通知、选择记忆和目录删除的真实合成后台及多语言/主题/四档响应式证据见现有修复记录；最终焦点修正另有查询刷新回归通过。

## 容量复核修正

初轮并行压力下 100k/10k 常规浏览及宽搜索超出 250ms 预算；串行复核仍有 browse 279.664ms、宽搜索 266.524/284.483ms，故未把两轮记为通过，也未放宽预算。定位到无活动变更时的逐行子树过滤及回收索引参与正常数据选择；新增 migration 50，仅在有活动删除的库内执行原子树过滤，两个回收索引改为非空部分索引。保持已有 migration 49 不变（222 已部署过该版本）。对应无活动、活动子树和其他库可见性真实 SQLite 回归覆盖。

永久删除独立合成 ZFS 树：1 万/10 万非媒体文件后台全流程约 1.21/14.35 秒，命令行 rm -rf 对照约 0.25/2.68 秒；深度 64 目录和三个顶层目录通过。200 项批量五组提交 p95 507.33ms，期间浏览 p95 23.47ms，峰值约 163.5MiB。只说明该机器/文件系统/合成负载，不承诺所有 NAS 同速；未在真实媒体上测试。

migration 50 的原生 amd64 候选 `foliopath:release-perf-20261030`（仅合成验收，未替换 9090）在同一 100k/10k 档通过全部原预算：扫描 111 秒，浏览及四种搜索 p95 213.818/161.918/159.710/213.780/225.133ms；峰值约 181.4MiB。完整 Go 检查、接口主分支兼容和 Linux 可见性/删除恢复回归通过。容量 JSON 在服务器 `release-checks/evidence/capacity-optimized.json`，相关本地日志留在 `/tmp/foliopath-direct-release`。

旧 v1.9.1/schema 48 → 候选 schema 50、同版本重启、完整停机备份及旧镜像配对恢复回退全部通过；合成原件哨兵保持。测试使用专用 Linux named volume，未打开或复制线上数据库。服务器 Docker 27 的测试 stop 参数改用兼容的 `-t` 后执行通过；该普通测试源仅本地保留，不改变部署流程。

## 正式发布结果

- 开发提交 `ea8488a4487715ed2dc7b0097d9965d8abd17b50` 和验收记录经 [PR #76](https://github.com/HappyQuQu/foliopath/pull/76) 合并为 `c6448713e8a8aa238bfbb66b9e089192c9daf051`。Release Please 自动合并 [PR #77](https://github.com/HappyQuQu/foliopath/pull/77)，冻结发行源码 `6a9d482e93349e9ad2d991ce978ef38d06244cf4`，发布 [v1.10.0](https://github.com/HappyQuQu/foliopath/releases/tag/v1.10.0)。
- [正式流水线 41030969021](https://github.com/HappyQuQu/foliopath/actions/runs/41030969021) success：main/AI × amd64/arm64 四个原生构建验证，以及两个多平台推广任务全部通过。每架构构建一次、同摘要验证，推广日志确认没有重建并保留 SPDX/SLSA attestations。Docker Hub 概览同步成功。
- 四份原生验证记录的 source/version 一致。公开 Docker Hub tag API 的两组件各四个标签、Linux amd64/arm64 runtime manifest 均与这些原生 index 内的 runtime descriptor 一致。AI 的模型准备、恢复与实际合成推理证据随原生记录归档，没有用普通健康状态替代。
- 主应用 `1.10.0`、`1.10`、`latest`、`sha-6a9d482` 索引：`sha256:3d7623b7db0948fc44588dcf3aacc88ffe200acec5fa8de3cb5b31b478ca677e`。
- AI `1.10.0-ai`、`1.10-ai`、`latest-ai`、`sha-6a9d482-ai` 索引：`sha256:0657319eb4378daefad8c2ffc2c9a6693f7f57ee9620c237be5c40d7b2b8e20e`。

| 组件 | 架构 | 已验证单平台 index | 发布 runtime manifest |
| --- | --- | --- | --- |
| ai | amd64 | `sha256:3008389810c2d0f3dd0006f39ada0cd75b5fcb2749bf648fa5ec96b44b19c381` | `sha256:fe50e2ac6dcc5339c2ecd2deb7f6e43eb8cf886dca9fe398aa8383d0644a7c48` |
| ai | arm64 | `sha256:3ad2efe82daea3c7e113a08e3b9132c97d645f7252cd217252e515fb6a725244` | `sha256:0803a3f206e9b7c87e5b691b621b1f5fc61ded499a92f938a150014513963f05` |
| main | amd64 | `sha256:111822a7a49f80dc9685d772ae096d29198569ddddca5f57510c4346d0ade095` | `sha256:2d5e12a0e55e28139ba54fb2b677afd295ec383bc300e6d905abba0600df33b7` |
| main | arm64 | `sha256:8f2e190e3515e8740354aac7a6fea9886ee04eb8b1e53044844947439341a635` | `sha256:b950cb609c860fddfb37bf4f180f458de3624d7dc7c5a4d1e6242ecfeddd9307` |

本地原始工程日志、正式 workflow 日志、image-plan、四份验证记录及公开标签核对位于 `/tmp/foliopath-direct-release/`；普通测试、日志、截图和产物不提交。此次未再次调整 222 的运行配置、PVE 或原件。收尾只补发行证据，使用 `[skip ci]` 文档提交；不重新构建、不移动既有发行标签。

## 222 正式版升级

2026-10-30 用户要求更新 222，并在执行中明确要求“不用备份”。初始备份校验脚本因 Python 3.10 不支持 file_digest 而退出，旧服务自动恢复；未迁移或切换原数据。用户取消备份后没有重新执行备份；本次已产生、未完成恢复验证的临时归档已撤掉，原有历史备份保留。

- 9090 已固定主应用 `evanqu/foliopath:1.10.0@sha256:3d7623b7db0948fc44588dcf3aacc88ffe200acec5fa8de3cb5b31b478ca677e`，实际 amd64 配置 ID `sha256:890baa9fac1eebc8f01d0031facfaf4a748299ee5e95e1388768b0a04ff9adbb`，版本 v1.10.0、源码 6a9d482e93349e9ad2d991ce978ef38d06244cf4。
- schema 49 → 50；前后 3 库、128,731 资产、2,446 目录、1 用户一致，三个库最终 ready。删除设置 enabled=1、recycle=0、retention=null、revision=4 保持，原账号与整理数据保留。
- 只切换主服务 image，已比较有效 Compose、环境摘要、端口与媒体/数据挂载；AI 容器 ID、镜像、启动时间、重启次数、挂载完全一致。没有修改 PVE、媒体原件或 AI。
- 运行 version、healthcheck/database-check 和容器 healthy 通过；同源码 Vite 构建与 8 项部署文件/SPA 路由逐字节比较通过。真实媒体未用于删除测试。
- 结果和脱敏状态在服务器 `/root/foliopath/upgrades/v1.10.0-20261030`，本地 `/tmp/foliopath-222-v1.10.0/`。磁盘归因结果仅保留本地，不作为性能或容量验收证据。
