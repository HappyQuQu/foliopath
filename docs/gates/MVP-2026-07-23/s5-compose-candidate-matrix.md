# Stage 5 Compose 与候选镜像矩阵

## 结论

**Go for `S5-001B/S5-002` runtime scope — linux/arm64 本机与操作者指定的原生
linux/amd64 服务器均通过完整候选矩阵。**

仓库根 `compose.yaml` 现在是候选部署配置，`.env.example` 是其参数清单；
`make test-release-image` 对同一个生产 `Dockerfile` 执行镜像、媒体、Compose 和可信代理
smoke。CI 仍为 amd64 与 arm64 原生 runner 配置相同命令及结构化 artifact；但操作者已
明确要求本轮忽略计费阻断的 amd64 CI，改以指定原生服务器执行同一入口。因此本 Gate
关闭 `S5-001B/S5-002` 的运行范围，但不关闭 `S5-001` 的最终不可变 digest、供应链和
发布签署。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-001B`、`S5-002`
- 需求/质量：`FR-DEP-001～004`、`NFR-SEC-001～002`、`NFR-SAFE-001`、
  `NFR-REL-001～002`
- owner：根 `Dockerfile` 与 `compose.yaml` 拥有候选运行配置；
  `tests/release` 拥有候选镜像验收和证据契约；CI 只在原生架构重复同一入口
- 前序 Gate：[发布镜像基础](s5-release-image-foundation.md)、
  [可信代理与发布 HTTP 安全](s5-trusted-proxy-security.md)
- 架构：仍是单容器、单 Go 进程、单 `/library` 和 `/app/data`，未改变既有 ADR

## 已固定的候选默认值

- 进程固定为 UID/GID `65532:65532`，只读容器根，丢弃全部 capabilities，并启用
  `no-new-privileges`。
- `/library` 是唯一只读媒体 bind，`/app/data` 是唯一持久可写 bind；`/tmp` 是
  `16 MiB`、`noexec,nosuid` 的受限 tmpfs。
- 宿主端口只绑定 `127.0.0.1`。应用在容器内非回环监听时必须配置精确
  `FOLIOPATH_TRUSTED_PROXIES`，不能使用隐式或宽泛信任范围。
- Compose 要求显式镜像引用、媒体目录、数据目录和代理 CIDR；示例不会默认为
  `latest` 或匿名 LAN 模式。

## 自动化证据

`tests/release/image_smoke.sh` 构建真实候选镜像并验证：

- 嵌入 SPA、readiness、版本注入、SIGTERM 零退出及媒体 sentinel 哈希不变；
- 镜像内 Go 二进制链接 libvips，FFmpeg 含 WebP 与 H.264 编码支持；
- JPEG、PNG、WebP、GIF、MP4、MOV、MKV 合成、probe 和视频 poster 提取；
- 截断 MP4 被拒绝；
- `compose.yaml` 能启动为 healthy，且只读根、丢弃 capabilities 和 `/library:ro`
  与声明一致；
- 无合法代理声明的直连请求失败关闭；精确可信 peer 的单跳 HTTPS 声明成功并返回 HSTS。
- 恢复子演练验证离线备份/恢复、cache/tmp 重建、同版本 migration 幂等、`SIGKILL`
  后 WAL 恢复、数据盘满及损坏 SQLite 失败关闭。

2026-07-28 本机原生 linux/arm64 实际执行：

```text
make test-release-image
candidate platform=linux/arm64 size=208256665
Stage 5 candidate image smoke passed
```

镜像 digest 由每次构建记录，但本地临时标签在测试退出时删除，不作为发布标识。

同日 [S5-002A 原生 amd64 与真实媒体验证](s5-native-amd64-real-media.md) 在一台
4 CPU/4 GiB 原生 `linux/amd64` 服务器上重复完整 smoke 并通过；后续候选又在该服务器
原生构建并通过媒体、Compose/代理、恢复和升级/配对回滚矩阵。两架构候选返回相同的
`index-BDEL2ued.css` 与 `index-CQHwtgZD.js` 嵌入 SPA asset，SPDX 运行包闭包除架构/候选
tag 外一致。该证据不是 GitHub artifact，也不冒充最终 digest 签署。

CI 的 `Release candidate (amd64|arm64)` job 分别使用 `ubuntu-24.04` 与
`ubuntu-24.04-arm` 原生 runner 执行同一 Make target。每个完整通过的 job 才会写出
`release-image-<arch>.json`，记录冻结 release、source commit、实际 OS/architecture、
镜像 digest/size、smoke suite、结果、workflow run ID/attempt 和时间，并上传为
`release-image-<sha>-<arch>` artifact。`make verify-release-image-evidence` 要求两个文件
属于同一提交和同一 workflow run，且两个架构的完整 smoke 都是 passed；单个成功 job、
不同 run 拼接或手写状态不能通过。

2026-07-28 只读核对当前已提交 HEAD `22eb7fa382abb5523afa1d49347bb23a6e0752aa`
的 [CI run 30314930003](https://github.com/HappyQuQu/foliopath/actions/runs/30314930003)：
所有 job 都在分配 runner 前失败、没有 step/log；GitHub check annotation 明确报告近期
账户付款失败或 spending limit 需要提高。该结果不是产品或测试失败，也不是任何架构的通过
证据。该外部账户状态仍应修复，但按操作者本轮决定，它不再阻断 `S5-002`；未来 CI 运行
仍必须满足下述成对证据合同。

下载同一 run 的两个 JSON 后，评审入口为：

```text
make verify-release-image-evidence \
  EVIDENCE_DIR=/path/to/downloaded/evidence \
  RELEASE_SHA=<40-to-64-character-lowercase-commit>
```

## 本轮退出条件

- linux/arm64 本机和指定原生 linux/amd64 服务器均完成相同的完整 smoke：已满足；
- 两架构嵌入 SPA 标识、运行包闭包和只读/安全运行语义配对：已满足；
- `S5-001B/S5-002` 可勾选；最终不可变 digest、容量、浏览器、供应链和 RC 仍由
  `S5-001`、`S5-005～010` 分别阻断；
- 未来恢复 Actions 后，`make verify-release-image-evidence` 仍拒绝不同 commit/run 或
  单架构结果，不能用本轮人工记录降低后续自动证据合同。
