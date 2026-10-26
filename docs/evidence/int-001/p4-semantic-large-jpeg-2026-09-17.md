# P4 大 JPEG 语义分析修复证据

范围：[修复记录](../../changes/FIX-2026-09-17-semantic-large-jpeg.md)，FR-INT-002、005 / NFR-INT-002、003。首轮仅本地 Docker，后续增加 server222 已授权停服隔离复验与本地合成旧库升级；不部署到线上。真实样本为此前授权的匿名副本，不修改原媒体，不训练或上传。

## 失败与修复过程

1. 仅提前 JPEG shrink、仍使用 32 MiB 源预算：90 图片中 89 完成/1 失败，人脸 90/90、视频 10/10；整体退出 1。剩余 sample-062.jpg 为 58668512 bytes，源字节超限。原始 `test-results/p4-semantic/sample-report.json`，不能写成全通过。
2. JPEG 改为独立 64 MiB 预算，但仍使用 ReadAll：1 GiB race 字节边界组合测试退出 137。原始 `imagevips-final.log`；容器自动删除，未留 OOMKilled 状态，不能将退出码单独当作已确认内核 OOM 证据。
3. 改为已打开的 seekable 流先校验长度，再一次分配准确字节数；ReadFull 并检测额外字节，源增长/缩短拒绝，避免 ReadAll 扩容副本。继续同一 1 GiB 限制复验，不提高容器预算。

## 检查记录

- 初次构建因局部变量名与 PNG 导出参数冲突失败，已修复；保留 build.log。
- 直接 `go test -race -tags=libvips ./internal/app ./internal/semantic`：app 中多个重启型集成测试失败（govips cannot be stopped and restarted），semantic 通过。不是全包通过。
- 默认无 libvips 标记的 app/semantic race 回归通过（67.321s / 1.014s）。该串行命令后续专项构建与源码修改重叠失败，不掩盖；最终固定源码另跑专项。
- 图像适配专项使用带 libvips 的编译后二进制，在 1 CPU / 1 GiB / 无 swap / 无网络 Docker 中执行；包含显式生成的渐进 JPEG。
- 架构检查、结果门禁 Node 3 测试、diff 空白检查通过；没有运行全部七项发布检查。

## 最终固定候选

固定主应用候选 `sha256:e71df56eefddc13026c2d7124f7905f923dea01d3f062415d1c870d66d488c66`（linux/arm64），沿用 p4-oom-fix 原生库/前端，仅替换本次 Go 二进制；不是发行构建。

最终 `imagevips-bounded.test -test.run TestRemoteSemantic`：7 项全部通过，包含 108 MP JPEG、81 MP 渐进 JPEG、JPEG 字节预算、源增长/缩短、倍率及原格式预算；容器明确 `exited 0`、`OOMKilled=false`。仍为 1 CPU / 1 GiB / 无 swap、无网络。日志 `test-results/p4-semantic/imagevips-bounded.log`。

最终固定源码 `-race -tags=libvips -run 'TestSemanticPreprocessingUpgrade|TestDeploymentConfiguration|TestDeploymentBootstrap' ./internal/app` 通过（5.543s）；证明新旧预处理身份不同、配置/启动专项有效，不将其称作所有带 libvips 主应用测试通过。

最终固定 100 样本探针退出 0，`functionalPass=true`，原始 `test-results/p4-semantic/sample-bounded.json` / `.log`。同一远端匿名副本在本地 Docker 复验，资源 p4-large：app 1 CPU/1024 MiB、AI 3 CPU/2304 MiB、无 swap。AI 镜像 `sha256:28f8fb6351234b5e94d09477f9d112ef36f2b41bc8537268d50e6bd1459f0d85`。

| 项目 | 结果 |
| --- | --- |
| 图片语义 | 90/90，失败 0，16128 ms，期间浏览 17 次 |
| 人脸 | 90/90，失败/降级 0，44323 ms，期间浏览 45 次 |
| 视频 | 10/10，失败/降级 0，10096 ms，期间浏览 11 次 |
| 标签 | eligible/completed=90，失败/降级 0 |
| 中英文图片/视频搜索 | 8 次成功，35～55 ms（不是 P95），图片每次 20 / 视频 10 结果 |
| 样本 | 100 项，副本 hash 前后不变 |
| 运行状态 | 任务期间两容器 running、OOMKilled=false、RestartCount=0；流程正常结束清理 |
| 结束内存快照 | app 266.3 MiB / AI 2.058 GiB，非峰值 |

该轮修复消除了这批输入的 36 项语义失败，仅证明固定样本本地功能闭环。没有质量 ground truth，搜索有结果不等于排名准确。后续原生与升级补验见下文；完整回归和发行验收未完成。原 P4 0/8、总清单 18/34 保持，不据此刷完成率。

## 后续：旧库升级与配对回退

新增 `spikes/int001-remote-ml/p4-semantic-upgrade-probe.mjs`，只用独立合成媒体和测试数据卷。旧镜像 `sha256:0421fb761cbf2d00c847038f566dfdebb031f8b59ca4cf02630995ac86503b64` 创建两库并完成语义/人物处理、人工关系后，停写备份，再换上述新镜像。最终报告 `test-results/p4-semantic/upgrade-final.json` / `.log`，退出 0、pass=true：

- 新旧 semantic generation ID 不同，两个库新代次 completed 均从 0 开始，不混用旧向量完成数；人脸代次不变。
- 两库图片语义回填 complete，搜索非空；视频严格保持旧基线 2 eligible / 1 completed / 1 degraded / 0 failed。
- 人工人物关系、人工标签和合成原图 hash 保留；新镜像重启后新代次及完成状态保留。
- 停止新镜像，用旧镜像＋升级前停写备份启动，恢复旧代次、旧覆盖及同一人工状态。未将升级后的库直接交给旧版本。

保留前两次测试失败：错误地要求共享短视频 fixture 全部 complete，而旧/新实测均有 1 个已知短视频降级，且 `p3-business-probe.mjs` 原基线相同。最终改为断言精确基线而非跳过视频检查。`upgrade.json`、`upgrade-inspect.json` 均 pass=false，不覆盖。不是任意真实旧库或所有升级版本矩阵通过。

## 后续：原生 amd64

原生候选 `sha256:522cf90e00adf6faf2a1eae5f226a1ac935349d686b644d22967e044bd9a46a9`；三个核心修改文件与本地源码 SHA256 一致。7 项带 race 的 `TestRemoteSemantic` 在 server222 1 CPU/1 GiB、无 swap/无网络容器通过，退出 0、OOMKilled=false，日志 `test-results/p4-semantic/native-semantic.log`。

100 样本维护测试位于 `/tmp/foliopath-p4-native.NjvFNg`；同一匿名样本、p4-large（app 1 CPU/1024 MiB，AI 3 CPU/2304 MiB，无 swap）。原始本地归档 `test-results/p4-semantic/native-sample.json`，functionalPass=true：

- 图片语义 90/90、失败 0，37364 ms；期间浏览 38 次。
- 人脸 90/90、失败/降级 0，86658 ms；期间浏览 87 次。
- 视频 10/10、失败/降级 0，15134 ms；期间浏览 16 次。
- 标签 90/90、失败/降级 0；中英文搜索 8 次成功，51～52 ms（非 P95）。
- hash 前后不变；测试期间两容器 OOMKilled=false、RestartCount=0。结束快照 app 207.1 MiB / AI 1.932 GiB（不是峰值）。

维护最终 `test_exit=0`、`services_restored=1`；foliopath、immich_server、immich_postgres、immich_redis 四个原服务全部 healthy，前后镜像 ID 比对一致。只更新独立测试标签，未部署生产镜像或修改原媒体。

本轮没有更改生产代码，仅补升级探针和原生验证。架构检查、升级探针 Node 语法、结果门禁 3 测试及 diff 检查通过；未重跑所有发布检查。完整质量、排名数值跨架构一致性、长期稳定性和发行不在这次通过结论内。
