# P1 人脸代次安全接入验收

复查更正：以下为首次验收记录，不再单独支持“P1 无遗漏完成”。后续发现激活本地 busy 和服务端 busy 两处遗漏，已在[专项修复与复验](face-busy-repair-2026-09-16.md)中处理并通过；以最新进度清单为准，保留原始通过及撤回事实。

2026-09-16；CR-2026-030 / FR-INT-009 / NFR-INT-002、004；POST-MVP-5 S2。
Owner：face / app / SQLite / API。范围：剩余清单 A01～A03，不含 P2 非空人物业务与质量验收。

## 实现

- A01 完整部署兼容身份和原子激活：[证据](face-deployment-commit-2026-09-16.md)。
- A02 可选配置、启动核验、离线恢复：[证据](face-startup-2026-09-16.md)。
- A03：`run.go` 组合 ContentService 安全读取 → imagevips → 远程 FaceRuntime → face.Processor → 代次绑定 JobProcessor → SQLite。分析/清理各一个 worker，均复用全局后台预算；HTTP 与 CLIP 共用有界客户端。服务不会获得媒体或数据库挂载。
- 开启既有每库设置、分析和清理控制路由及取消 owner；设置读取不再依赖尚未装配的人物结果服务。未新增公共 API 合同或数据库表。
- 每次推理前后验证部署指纹、代次、维度、活动状态和可用性；任务准入、结果提交、聚类切换和成功终态在事务内再次检查 service 模型可用性。
- 远程故障按模型不可用终止任务，不计作单张图片失败；本地请求槽忙保留 claim，复用既有租约与最多三次尝试恢复。
- 增加等待预算时的取消清理钩子；仅有持久化用户取消才提交 cancelled，进程退出保留 claim。
- 修复实际强杀测试发现的恢复冲突：通用 AI 操作恢复原先会把人脸操作直接标成 `operation_interrupted`，现在四类人脸操作由各自队列恢复。

## 本地 Docker 证据

最终主应用镜像 `foliopath-app:face-worker`，Docker image ID：
`sha256:dd9c5f35e7079081d474c91595c504ca24e09fc6b97f6411803a3e5eb5b5ca83`。
最终竞态测试镜像 `foliopath-remote-ml-regression:local`，Docker image ID：
`sha256:b7930cdb9bbdecda165342bb98ee4f095a0fda242ddce212f0eea43307360842`。

构建命令（本机省略外部 Dockerfile syntax frontend 下载，其他源不变）：

```sh
sed '1{/^# syntax=/d;}' Dockerfile | docker build --pull=false -t foliopath-app:face-worker -f - .
sed '1{/^# syntax=/d;}' Dockerfile | docker build --pull=false --target remote-ml-regression -t foliopath-remote-ml-regression:local -f - .
node spikes/int001-remote-ml/face-startup-probe.mjs
```

探针使用仓库纯色 PNG 和视频 fixture，只读挂载单一 `/library`；只分析一张图片，预期零人脸。真实 buffalo_l 推理后 completed=1、failed=0、state=ready；不把零人脸当作非空质量证据。

检查项：首次服务离线时核心就绪、CLIP/face 独立激活、停服标记保留模型、恢复和应用重启保留身份；正式鉴权 API 建库/扫描/开启/分析、幂等重放；冻结临时 AI 容器并在任务 running 时强杀应用，重启后等待真实租约到期，同一 operation 恢复 succeeded、代次不变、覆盖率保持 1/1。

强杀恢复修复后成功跑通一次；最终镜像同一探针复验也全部通过（exit 0），包含同一 operation 在真实租约到期后恢复 succeeded。自有临时容器、网络和合成数据卷已清理。P1/A01～A03 全部关闭。

## 回归

主进程组合版本在 Docker 成功运行：face、aimodel、resourcecontrol 全包；remote Face（跳过 opt-in Probe）、app Deployment/Face/SemanticContentSource、API Face；SQLite `Face|AIOperationRecovery`。
最终忙状态修正后镜像又执行并通过：

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/face.test -test.timeout=3m && /out/remote-image-probe.test -test.run Face -test.skip "Probe$" -test.timeout=3m && /out/sqlite.test -test.run "TestFaceBusy|TestFaceServicePublicationFence|TestFaceAdmissionCancellationPreservesShutdownClaim|TestAIOperationRecovery" -test.v -test.timeout=3m && /out/app.test -test.run "Deployment|Face|SemanticContentSource" -test.timeout=3m && /out/api.test -test.run Face -test.timeout=3m && /out/contract.test -test.timeout=3m'
```

其中新增：服务离线/退役事务拒绝（3 场景）、等待预算取消与停机区分（2 场景）、四类人脸操作运行/取消中恢复归属（8 场景）、本地忙后可恢复、传输错误分类及安全媒体跨库拒绝。代次绑定 11 场景和原激活矩阵继续通过。

`make fmt-check`、`make arch-check`、`git diff --check`、探针 `node --check` 通过；修改的 Go 文件执行 gofmt。
face 全包中的 100k 容量测试未启用，明确跳过。未运行完整七项发布检查、全量浏览器回归、非空人脸质量、原生双架构或容量验收；这些不属于 P1 的关闭结论。

首次强杀探针失败是真实发现，已修复，未删除失败事实。测试只创建、移除自有临时容器/网络/合成数据卷；未修改真实媒体、未访问 server222、未提交推送。
