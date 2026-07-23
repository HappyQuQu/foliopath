# FS-05：双架构运行镜像与恢复 spike

## 结论

**状态：Conditional（本机 Linux/arm64 通过；等待原生 amd64/arm64 PR CI）**

**验证日期：2026-07-23**

**目标范围：`MVP-2026-07-23` / Stage 0 / `NFR-COMP-001`、`NFR-REL-001`、`NFR-SEC-001`**

`spikes/fs05-runtime` 提供一个隔离的 Stage 0 运行探针。它不是生产 `cmd/foliopath`，
不提供业务 API，也不提前关闭 S1 的正式配置、认证、日志和健康契约任务。探针复用真实
SQLite store 与嵌入 migration，验证最终镜像所需的系统依赖和容器故障语义。

本机 Docker Desktop Linux/arm64 已通过完整矩阵。只有 PR 的原生 amd64/arm64 jobs 都通过后，
才关闭 S0-107。

## 镜像边界

- multi-stage build：Go 1.26.4 builder → Debian 12 slim runtime；
- builder 与 runtime base image 均按多架构 manifest digest 固定；
- runtime 安装 `ffmpeg`、`libvips42`、`ca-certificates` 和 `tzdata`；
- 固定 UID/GID `65532:65532`，无登录 shell；
- 单进程、8080 端口、镜像内 healthcheck；
- 只读根文件系统、`/tmp` 受限 tmpfs、全部 Linux capabilities 丢弃；
- `/library` 只读 bind mount，`/app/data` 为唯一持久可写目录。

本机 arm64 构建解析到：

| 项目 | 结果 |
| --- | --- |
| image | 206,224,898 B；linux/arm64 |
| Go | 1.26.4；`CGO_ENABLED=0` 的探针二进制 |
| Debian | bookworm slim digest `sha256:7b140f…5818` |
| libvips | `libvips42` 8.14.1-3+deb12u3 |
| FFmpeg | 5.1.9-0+deb12u1 |
| tzdata | 2026b-0+deb12u1 |

这些版本是本次 Debian snapshot 解析结果。发布镜像仍必须按版本构建并附 SBOM，不能把
浮动 apt repository 结果当作永久锁。

## 已验证行为

`spikes/fs05-runtime/verify.sh` 只使用临时合成媒体目录和临时 Docker volumes，验证：

1. 空数据卷启动时执行真实 migration，并进入 healthy；
2. 进程 UID/GID 为 65532，`CapEff=0`；
3. `/library` 写入失败，`/app/data` 写入成功；
4. `docker stop` 触发 SIGTERM，5 秒 shutdown deadline 内记录完成；
5. 停机后备份完整数据卷，恢复到新卷，配置记录逐字段可读；
6. 同一恢复卷重复启动 migration 幂等；当前尚无上一发布版本，因此这是现阶段唯一诚实的
   no-op upgrade 路径；
7. `/app/data` 只读、64 KiB 已满 tmpfs、损坏 SQLite 都以非零退出失败关闭，不能进入 ready。

不在本轮声称：

- 生产 `cmd/foliopath`、认证或业务 readiness 已实现；
- 在线 SQLite backup API；
- 已发布版本之间的真实 schema upgrade/rollback；
- 运行期 `/library` unmount 或 NAS 断连语义；
- 缓存、媒体队列和磁盘余量策略已实现；
- 该 spike 镜像可作为稳定版本发布。

上述要求分别继续阻断 S1、对应 Backend Gate 与 Release Gate。

## 复现

```sh
docker build -f spikes/fs05-runtime/Dockerfile \
  -t foliopath-fs05:local --build-arg VERSION=stage0-local .
spikes/fs05-runtime/verify.sh foliopath-fs05:local
```

Compose 结构可用 `spikes/fs05-runtime/compose.spike.yaml` 检查；它仍标明 spike，不是正式发布
Compose。

## 完成条件

- 原生 linux/amd64 与 linux/arm64 构建同一 Dockerfile；
- 两边运行相同 runtime/recovery/failure fixture；
- 记录两边 image、libvips、FFmpeg 版本与大小；
- S0-108 生成 source/npm/image SBOM 并审查 codec/许可证。
