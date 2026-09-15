# 本地 Docker 回归（2026-09-15）

## 全包复验最终结果：通过

在同一本地 Linux/arm64 测试镜像中执行 `/out/sqlite.test -test.v -test.timeout=15m`，
未传入 `-test.run`、未删除用例、保留 race 检测。完整 SQLite 包最终输出 PASS，exit 0。
首次 5 分钟超时记录保留在下方；本次补足全包证据，不再处于“待复验”。
镜像：`sha256:414ea32782482617a70105ae69b638dd75d8bb30b59f3e5679bed4aa8acdd89f`。
实际容器 network=none、readonly=true、2 GiB、无 host mounts。包括旧代次回归、
新空间绑定/流式回填成功与提交前退役拒写、历史迁移、库重叠并发约束，均通过。
Dockerfile 默认 SQLite 预算同步改为 15 分钟并开启逐项输出；此命令已对现有镜像显式验证。
此次只复跑 SQLite；semantic/imagevips 与真实 ML probe 的通过记录来自本报告之前的同轮测试。
这仍不代表 app 完整接入独立 AI、浏览器 E2E 或发行验证完成。

范围：CR-2026-030 / POST-MVP-5，FR-INT-017/019/020、NFR-INT-007/009/010。
用户要求后续测试在本地 Docker 运行。本次未访问 server222、未挂载用户媒体或正式数据库。

## 可重复入口

`Dockerfile` 新增测试专用 `remote-ml-regression` target，继承已有 native 构建，
编译 semantic、imagevips、SQLite 三个包的 `-race -tags=libvips` 测试二进制。
生产 runtime stage 不依赖此 target。Compose 新增 test profile 的 regression/encoded-probe；
命令见 `spikes/int001-remote-ml/README.md`，无需在宿主机安装 libvips。

## 环境与已执行结果

- Docker：本地 Linux/aarch64。
- 测试镜像构建成功：`foliopath-remote-ml-regression:local`。
- 实际 regression 容器：network=none、readonly=true、无挂载、2 CPU / 2 GiB，
  /tmp 为 512 MiB tmpfs；数据库和图片均为临时合成数据。
- semantic、imagevips 回归已通过。损坏图片用例产生 native warning，包结果 PASS。
- SQLite 全包竞态回归：5 分钟超时，exit 2；当时正在 TestScanContractMigrationUpgradesVersionThreeSchema。
  未证明是功能缺陷或死锁，也不视为通过。另行运行本轮代次/绑定/流式回填聚焦测试，不替代全包结果。
- 聚焦 SQLite 竞态测试 8 项（含生命周期子场景）全部通过：TestGenerationSpace、
  TestBoundEncoderUsesPersistedGeneration、TestStreamBackfillPersistedLifecycle、TestSemanticBackfillProcessor。
- 单独重跑 TestScanContractMigrationUpgradesVersionThreeSchema：PASS，1.99 秒。
  说明全包超时位置的单例可以完成；仍不替代全包成功，后续须增加全包预算或分片复验。
- encoded-probe 通过：PNG/JPEG/GIF/WebP → 正式编码入口 → Go HTTP → 固定离线 ML，
  四格式与文本均为有效 512 维向量，总耗时 1.53 秒（单次 smoke，非性能承诺）。
- 前后 49 个模型缓存文件指纹相同：`c8be0544ab43ffd0c3f81e550218c833410d4e593a3b282fc7844f811606211b`。
- ML 服务已停止；测试 run 使用 --rm 自动移除，模型缓存保留。

## 限制

数据库任务测试与真实 AI 推理测试仍是分开的链路，不能声称主应用完整接入。
没有浏览器 E2E、amd64、发行验收或实际用户媒体质量验证。此测试镜像不是正式部署镜像。
