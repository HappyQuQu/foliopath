# server222 P6 候选部署（2026-09-17）

## 结果

当前 `aifeature` 工作树已部署到 `server222` 的 `192.168.2.222:9090`。部署采用正式
`compose.ai.yaml` 双服务拓扑，主应用和 AI 服务均在原生 linux/amd64 构建、启动、重启后保持
healthy；局域网首页返回 200，未认证会话接口返回 401。该部署是私有功能候选，不改变 P6
供应链审计的 Release No-Go 结论，也不构成模型质量、许可或正式发行批准。

## 固定输入

- 源分支：`aifeature`；同步时基准提交 `7d2c05f415467a11f1b8a0d07e6d7731a78335fd`。
- 当前脏工作树快照指纹：`a3f8cce03931769f20c8bb0e260d6eee2aef3a54a7ad87626a249075f10dccae`；不能冒称仅由上述提交可复现。
- 应用镜像：`sha256:16be768e970b9ba286fd876b302ff5bb44a79e1d94a642f58263cd4cbccc2e3d`，amd64，81,392,756 bytes。
- AI 镜像：`sha256:241bc447b3d192e26dd04da9081e66016741e33a08a29c78f434144a678fb0e3`，amd64，1,290,040,122 bytes，包含 CLIP 与 buffalo_l 合同。
- 模型缓存：复用 `/root/immich/model-cache`，只读挂载；按镜像清单验证 49 个文件、802,152,765 bytes，大小及 SHA-256 全部匹配。
- 媒体根：`/mnt/r0data/picture` 只读挂载到 `/library`，未修改媒体原件。

构建因 server222 直连 Docker Hub/GitLab 间歇失败，显式使用部署者提供的
`http://192.168.2.231:7890` 代理取得固定 digest/校验和依赖；最终两个运行镜像环境中均未保留
HTTP/HTTPS/ALL proxy 变量。构建器下载失败轮次不计通过，最终固定源码构建成功。

## 迁移和回退

切换前先停止旧应用并复制完整数据目录、旧 Compose 和旧镜像 ID。第一次副本用于 19090 迁移
演练，演练后的副本不作为回退备份。正式切换前再次停写并创建新鲜回退备份：

`/root/foliopath/backups/pre-p6-cutover-20260917-200102`

该备份约 1.2 GiB。旧镜像和旧 `docker-compose.yml` 均保留；没有删除旧数据、旧镜像或媒体。
正式配置保存在 `/root/foliopath/compose.ai.yaml` 和权限 0600 的
`/root/foliopath/ai.env`。AI 配置明确标记缓存许可未完成审核，人脸阈值 profile 标记为
`server222-uncalibrated-20260917`，不得解释为质量校准通过。

## 实际检查

- AI 预启动：固定缓存校验完成；语义和人脸 identity 与离线镜像身份一致。
- AI 推理：真实文本 `portrait` 返回 HTTP 200 和 CLIP 向量；未将健康检查冒充推理成功。
- 迁移演练：备份副本、127.0.0.1:19090，主应用和 AI 均 healthy；首页 200、未认证接口 401。
- 正式切换：两个容器均 healthy；首页 200、未认证接口 401。
- 重启复验：`docker compose restart` 后两容器再次 healthy；局域网
  `http://192.168.2.222:9090/` 返回 200。
- 安全边界：两个容器均只读根、`cap_drop: ALL`；应用仅发布 9090；AI 无宿主端口、无媒体或数据库挂载；`/library` 和 `/cache` 均只读。
- AI 日志有 Matplotlib 因只读根改用 `/tmp` 的非阻断警告；随后正常完成启动。应用日志没有迁移或启动错误。

## 当前边界

服务已经可用，但仍沿用 [P6 复核](p6-final-verification-2026-09-17.md) 的 No-Go：模型质量、
长期容量、最终双架构配对、漏洞处置、完整 notices、许可、干净 provenance 和外部签署没有因此
自动通过。本次没有推送镜像或源码，没有公开发布。
