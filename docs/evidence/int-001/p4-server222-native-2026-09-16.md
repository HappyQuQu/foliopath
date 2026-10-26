# P4 server222 原生测试维护记录

范围：CR-2026-030 / POST-MVP-5 P4 E01/E03/E04 增量；Owner app/测试。用户明确指定 server222，并随后授权“停掉然后测试”。授权仅限测试维护，不更新现有部署、不改动原媒体或现有数据卷。

## 最终结果

第四次执行退出 **0**，记录 `test_exit=0 / services_restored=1`。基础容量、模型缓存、指定业务与恢复探针通过。结束后另行 SSH 复核原 FolioPath、Immich server、Postgres、Redis 四容器均 healthy，运行列表无测试容器；前后镜像 ID 文件 `cmp` 完全相同。没有把测试镜像部署到原环境。

恢复镜像：FolioPath `1bafc5349167a581d5a64dcc3235ba46489d25e103f89098b50495c609d008d2`；Immich server `8c6b230769c61601016067d898e76271a27069d42c51b4aa708fce8ef7c865e7`；Postgres `86a9b06ef82525cd385caaa029b5fb21e8206ddf12a5441e66e9dec4348609c7`；Redis `7cb874b6878d1a539328fa947200bfae55d8af78a3884209a99733eef7d0d970`（均为 SHA-256）。测试镜像和独立临时证据目录保留用于复核，不包含真实媒体或原应用数据库。

## 原环境与恢复约束

- 原生 `x86_64`，4 CPU、4 GiB，无 swap，空闲磁盘约 193 GiB。
- 测试前运行且健康：`foliopath-foliopath-1`、`immich_server`、`immich_postgres`、`immich_redis`。
- 维护脚本只接受显式 `FOLIOPATH_ALLOW_P4_MAINTENANCE=server222`，确认这四个服务运行且无第五个运行容器；记录镜像 ID。
- 先正常停止 FolioPath/Immich，再停止数据库/Redis。退出/失败/信号均先清理本次 run label 的测试容器，再恢复原四个服务，等待健康并核对镜像 ID。不是重建或升级原容器。
- 所有源码、合成媒体与结果放入独立 `/tmp/foliopath-p4-native.*` 目录。现有模型缓存只读；不挂载现有媒体或数据库到测试应用。

## 首次尝试与保护验证

第一次维护：四服务停止后可用内存约 3.8 GiB。构建 frontend 无法直连 Docker Hub token endpoint（连接拒绝），测试未开始。退出保护实际恢复四服务健康，记录 `test_exit=1 / services_restored=1`。不能把这次启动当作容量结果。

显式测试先前用户提供的 `http://192.168.2.231:7890` 代理访问固定 Docker Hub token endpoint，返回 200。后续使用独立 BuildKit 的构建专用代理，不修改 daemon 全局配置、不给测试应用或模型推理继承代理。

## 网络修复与执行过程

第二次仅构建器设置代理，客户端发起的认证仍直连失败，退出保护再次恢复四服务健康。第三次为 Docker 构建客户端和独立 BuildKit 同时设置显式代理，认证及固定依赖下载成功。此前失败没有被计为测试通过。

- 构建器 amd64 镜像：`sha256:260cc297a47c57183fe53fb963885068c30e976060fabc90e32af04919dbd0bf`；仅工具，不是产品镜像。
- BuildKit 限制 3 GiB、stage 并行度 1；构建结束后停止构建器再测容量。
- 上传的第三次独立源码快照 1,275 个文件已逐个 SHA-256 与当前本地文件核对；manifest SHA-256 为 `f2c41bbfa034e0ca4b5d1da57b0d277c0a349824a530393a84353af09a3fc3b3`，目录 `/tmp/foliopath-p4-native.S9GT8S`。未依赖旧服务器 checkout 或已部署旧镜像。
- 原生构建当前应用、固定上游的独立 AI 镜像和测试工具镜像。没有使用 QEMU 构建/执行这些远端步骤。
- 实际按顺序执行：模型缓存 hash 审计 → 100k/10k 基础容量 → 当前双容器业务 → 强杀/超时/备份恢复。
- 双容器业务阶段主应用限制 1 CPU/768 MiB、AI 3 CPU/2560 MiB；禁用 swap。两者共 4 CPU/3328 MiB，测试驱动另限 512 MiB，主机仍仅 4 GiB。基础容量阶段仍用原单应用 4 CPU/4 GiB 门槛。
- 本地 arm64 已成功复跑相同资源配置的业务探针（退出 0）：真实图片/视频/非空人脸推理、B05 源增改删与另一库不变、C01 混库审核/冲突/清除隔离及人工标签保留、C02 共享预算等待与取消、浏览不受阻。应用 `sha256:59521dc239c2076b861fa4a9beb9281b2ffcbb504d182fa26aa00a37ce3add3d`，AI `sha256:28f8fb6351234b5e94d09477f9d112ef36f2b41bc8537268d50e6bd1459f0d85`。它仍是小规模功能负载，不替代 100k 联合 AI 回填。

## 第三次结果与工具修复

第三次已完成原生应用/AI 构建。模型缓存核对通过：49 文件、802152765 bytes，manifest fingerprint `c8be0544ab43ffd0c3f81e550218c833410d4e593a3b282fc7844f811606211b`，与本地基线相同。容量脚本因测试工具未安装 Docker CLI 以 127 退出，尚未产生容量结果；`docker.io` 在此 Debian 版本不含 CLI。原四服务全部恢复 healthy，`services_restored=1`，镜像 ID 未变。

修复测试工具为显式安装 `docker-cli` 并在构建时验证 `docker --version`。第四次独立目录 `/tmp/foliopath-p4-native.Fzye3p`，只改测试工具/维护脚本；应用与 AI 源码不变，以精确 ID 验证后复用第三次原生构建，而非复用旧部署：

- 应用 linux/amd64：`sha256:b69612e370f080697104ac863c2a8ad0aa8c7e09042d92fc6c08ae2cad80c72f`。
- AI linux/amd64：`sha256:1647c1214bd014f7c36b8577656cfbd2e15f625a59f0ab4e6f1d1058797b3422`。

第四次快照 manifest SHA-256：`9ea6a19c48ee41c1f391aec492584bad87a26f2cb09eca7cc77f1001bf0dbf14`；与第三次逐文件 diff 仅两个上述测试工具文件不同。模型缓存再次核对相同。

第四次基础容量已通过：100000 媒体、10000 目录，扫描 88000 ms（预算 240000）；浏览 P95 87.965 ms；稀疏库内/全局 90.920/77.385 ms；宽库内/全局 153.894/142.690 ms（搜索预算 250）。容器内存峰值 192823296 bytes，数据库 42904576 bytes，缓存 5495808 bytes。每类延迟 30 次。完整原始结果见 [amd64 容量 JSON](p4-core-capacity-amd64-2026-09-16.json)。这只是基础目录容量，不是 AI 联合负载、质量或长跑。

原生 amd64 业务探针已退出 0：真实图片/视频/非空人脸推理；B05 源增改删与旧结果排除；C01 混库审核重放/冲突/清除隔离和人工标签保留；C02 人脸占用共享预算时图片/视频/标签/扫描/清除等待可取消、浏览保留。与本地使用同一 `p4-native` 资源配置及同一模型 manifest。小规模功能一致性不等于数值/排名质量误差验收。

本地同资源配置恢复探针也以退出 0 通过：正常停机/SIGKILL 后租约恢复、推理中断/真实 HTTP 超时失败后恢复重试、身份不匹配拒绝执行且不切代、停机卷备份恢复与派生清除重建，人工关系/标签和合成原图哈希保留。命令为 `FOLIOPATH_P3_APP_IMAGE=foliopath-app:p4-search FOLIOPATH_TEST_RESOURCE_PROFILE=p4-native FOLIOPATH_TEST_RUN_ID=p4-local-arm64-recovery node spikes/int001-remote-ml/p3-recovery-probe.mjs`。本轮 `make arch-check`、维护脚本 `bash -n`、环境模块 `node --check`、`git diff --check` 成功；未执行完整七项发行验证。

原生 amd64 恢复探针全部断言通过：正常停机与 SIGKILL 的原任务租约恢复、推理中断与真实 HTTP 超时失败后重试、错误模型身份拒绝执行且不跨代、停机备份恢复非空人物及人工标签、派生清除重建后人工关系和原图哈希保留。实际日志位于第四次目录 `results/business.log`、`results/recovery.log`。未在这轮运行 `p3-fault-probe.mjs`、浏览器矩阵或配对升级回退，不能把这组探针称作全部 P3 重验。

## 尚不能关闭

上述只证明原生基础目录容量、小规模双容器功能与指定恢复子项；真实质量标注、跨架构数值/排名误差、真实 10k/50k/100k 向量及长期稳定性仍需独立证据，P4 **0/8**、总计 **18/34** 不变。没有部署、提交、推送或发布。
