# 222 删除与回收站候选部署

2026-10-30；CR-2026-058 / POST-MVP-DELETE；Owner：app / files / deployment。

用户明确要求“按简单方案做，把多余的内核隔离撤掉，不要动 PVE，弄好后升级 222 的 9090，我要能直接用删除和回收站”。[ADR-0044](../adr/0044-simple-media-deletion.md) 取代 ADR-0043；这是指定实例的候选升级和启用授权，不代表正式发布、双原生配对或容量门禁通过。

当前已撤掉 Landlock、变更子进程、JSON 管道和私有命令入口，改为既有主进程后台 worker 直接调用 files。认证、相对路径、openat2、库根/跨挂载保护、无覆盖恢复和有界批次保留。删除期间外部程序不能移动目标目录及其父目录。

## 前一候选的准备记录

- 在 222 原生 amd64 上使用原 Dockerfile 构建 `foliopath:delete-preview-20261030`，版本 `v1.9.2-delete-preview.20261030`；镜像配置 ID `sha256:e4b7ba592d155c77ab27e9d692896e20cfb4959dddf15eff9ac42fbbdfcd4d80`。构建使用临时独立 BuildKit 和既有网络代理，没有修改全局 Docker 配置或发布标签。
- 源码归档 SHA-256 `be211ad9bdf7cf9eaf0c9358d1fe9016639cf908386422f68047d601b930845c`；远端源码、构建元数据和脱敏实例基线保存在 `/root/foliopath/previews/delete-20261030`。未读取或输出凭据。
- 从上一发行的已停机完整备份恢复到独立数据目录；旧镜像 `database-check` 通过。候选镜像在无网络、空只读媒体挂载、无能力及 no-new-privileges 的容器启动，healthcheck 和 database-check 通过。停止该副本后读取结果：schema 48→49，3 库、30,432 资产、452 目录保留，integrity ok、外键违规 0。没有用宿主机读取在线数据库，也没有挂载真实媒体做验证。
- 原 9090 容器内 `database-check` 通过。原应用、AI、数据和媒体挂载未切换；没有删除任何原件。升级覆盖草案仅保存在预览目录，未应用。

## 已撤回的内核隔离部署方案（历史）

222 为 PVE 200 上的 LXC，继承 `7.0.14-14-pve` 内核。PVE 和 222 的 Landlock ABI 查询均返回 EOPNOTSUPP；`CONFIG_SECURITY_LANDLOCK=y`，但 `CONFIG_LSM="lockdown,yama,integrity,apparmor"` 未包含 landlock。标准受限 Docker 内的文件变更边界测试因此失败关闭。没有通过关闭 seccomp、增加能力或绕过 Landlock 启用删除。

PVE 当前使用 systemd-boot；仅准备 `/root/foliopath-landlock-plan-20261030/cmdline.proposed`，内容是在现有 `root=ZFS=rpool/ROOT/pve-1 boot=zfs` 后加入 `lsm=landlock,lockdown,yama,integrity,apparmor`，保留原安全模块。原文及当前运行状态也保存在该准备目录。**未修改 `/etc/kernel/cmdline`、未执行 boot refresh、未重启宿主机或任何业务服务。**

应用该参数需重启 PVE，影响当时运行的 14 个 LXC：210、211、212、213、214、215、218、219、221、222、224、225、228、231。201 模板、217、220、229 及 VM 240 当时停止。用户收到重启询问后要求解释原因，未批准重启；继续等待明确决定，不执行主机变更。

该方案已撤回，不再申请或执行宿主机重启。原计划为：先备份并校验启动配置、核验原运行集合，再启用参数并重启/核验全部服务；重测标准容器内隔离后，才停止 9090 完整备份并恢复验证，再切换候选镜像与显式可写单一媒体挂载。应用删除仍由认证管理入口授权；原库不用作删除测试。schema 49 回退必须恢复配套升级前完整数据，不能让旧镜像直接打开升级后数据。

正式 S4 仍未通过；本次成功构建和迁移副本冒烟不改写 S3 的历史命令结果。技术依据见 [Linux Landlock](https://docs.kernel.org/userspace-api/landlock.html) 和 [Proxmox 启动配置](https://github.com/proxmox/pve-docs/blob/master/system-booting.adoc)。


## 简化实现验证

- `CGO_ENABLED=0 GOMAXPROCS=4 make arch-check generate-check lint` 和 `make test test-integration` 通过；`git diff --check` 通过。没有修改 HTTP/SQL 合同或前端交互，因此复用 S3 已通过的 86 文件 / 570 项前端及四档真实页面证据，不冒称本轮重跑浏览器矩阵。
- 222 原生 amd64 文件边界测试通过回收/恢复冲突、600 文件分批删除、库根与路径保护、取消、回收冲突及清理；14 个真实 SQLite 删除测试（含分阶段强杀、重试、暂停、保留策略和库移除阻止）通过。只使用临时合成媒体。
- 新程序在固定 Go 1.26.5 原生 amd64 工具链以 CGO/libvips 编译，沿用同日已完整构建的原 Dockerfile 候选镜像中的原生依赖和运行时，仅替换主程序及版本标签。候选 `foliopath:delete-simple-20261030` 配置 ID `sha256:158eb8ca90431960c75cb419af48aa021ee50f9a807d5ca92b7ac2061328cad1`；源码归档 SHA-256 `c159d93a32d587f43bb709fac7d96c04e90331b9fefc48c6b9c1d7dfbfa7ceff`。没有正式发布或重建原生依赖。
- 该原生镜像在 222 独立容器跑完整 HTTP 验证通过：真实 setup/login 会话、CSRF 拒绝、默认关闭、显式开启、回收、同请求重放、回收站阻止移库、恢复保留目录 ID、关闭回收站后永久删除非媒体文件，以及库根保留；独立数据和合成媒体均已清理。
- 一次性本机管理程序在旧图库独立恢复副本上调用真实 deletion 设置服务通过：`enabled=true recycle=true available=true manual_retention=true`。该程序不加入发行镜像、不增加公开 API，不读取或重设管理员凭据。


## 9090 升级结果：已完成

- 主应用停止后完整备份至 `/root/foliopath/backups/pre-delete-simple-20261030T064958Z`：包含原 Compose、有效配置和约 4 GB 的完整 data 归档。`tar -d` 逐文件比较通过，SHA-256 为 `d55176d969db36cbfd7be442d27921604eae6d740329710983c9aebc2a8717f0`；解包到独立副本，在原镜像内执行 database-check 通过后才迁移原数据。
- 新鲜停机备份基线为 schema 48、3 个媒体库、128,036 个媒体、2,454 个目录、1 个管理员。迁移到 49 并启用后数量逐项一致。先前 30,432/452 为上一发行备份的冒烟样本，不是本次升级基线。
- 一次性迁移容器首轮未提供可写临时目录，较大数据库迁移返回临时路径 I/O 错误；原数据完整性检查通过。给该容器提供有界 128 MiB tmpfs 后迁移和同一设置服务调用成功，未重建数据或修改迁移。最终 `enabled=true`、`recycle=true`、`available=true`、保留期限为手动清空。
- 仅将主服务切换到上述原生 amd64 简化候选镜像，保持 9090 和原数据目录。持久 Compose 固定该本机镜像，并显式设置 `FOLIOPATH_MEDIA_MUTATIONS=true`、现存唯一媒体挂载 `/library` 可写；有效旧环境值和 AI 服务定义逐项保留。原配置已备份，当前 Compose 采用 Docker Compose 支持的 JSON 格式。
- 运行中主容器 version、healthcheck、database-check 全通过，Docker 状态 healthy。`verify-deployed-web.mjs` 将全部静态资源及 `/files`、`/settings/about`、`/libraries/lib_1/browse` 与本轮嵌入 SPA 逐字节比较通过。原账号和会话数据保留，不创建或重置正式管理员。
- AI 容器 ID、镜像、启动时间、重启次数和挂载全部与准备阶段基线一致。没有重启 222 或 PVE，没有修改 PVE 启动配置、内核、Docker 全局代理、AI 模型或公网发行标签。原图库未用于删除测试；实际媒体写入仅为授权启用时创建回收站保留目录。
- 临时编译容器、合成 HTTP 测试容器/数据、镜像运输 registry 和 SSH 转发已清理。候选镜像、源码、管理辅助程序及脱敏 `result-simple.json` 保存在预览目录；辅助程序未加入运行镜像。原镜像另保留 `foliopath:before-delete-20261030`。

恢复旧版必须先停止新版、保留当前数据副本，再恢复配套完整 data 与原配置并固定旧镜像、恢复只读挂载；不能让 schema 48 的旧程序直接使用升级后数据。应用数据备份不撤销用户后续的永久媒体删除。通用双原生和容量发行门禁仍另行完成，本次没有正式发布。
