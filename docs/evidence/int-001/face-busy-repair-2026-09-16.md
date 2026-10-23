# P1 复查遗漏：busy 修复与复验

2026-09-16，CR-2026-030 / FR-INT-009 / NFR-INT-002、004，POST-MVP-5 S2。
Owner：app / remoteml / face；既有 P1 的缺陷修复，不新增产品范围、部署单元、API 或数据库状态。

## 撤回与修复

首次“P1 全部完成”的结论不够充分：测试未覆盖激活时本地忙、推理时服务端忙。保留[首次验收记录](face-worker-p1-2026-09-16.md)，不把遗漏改写成当时已通过。

1. 激活适配原来把本地 `ErrBusy` 映射成模型不可用，消耗三次持久化激活机会。现在单一激活 worker 通过 `FaceClient.WaitVerifyIdentity` 等待既有共享槽；占不到槽不退出加载、不提交失败。等待响应上层取消/停机；取得槽后的网络请求仍有既有超时。启动后台探活保留非阻塞语义，不复制重试状态机。
2. 服务端合法 `503 {"error":"busy"}` 原来被映射成离线。现在仅当 runtime/cache 响应身份一致、错误体不超过 1024 字节、仅包含受支持字段且没有尾随 JSON 时才返回 `ErrBusy`，由既有 FaceRuntime/JobProcessor 的忙状态路径处理。真正离线、错误身份、超大/畸形错误不能冒充忙。
3. 分析忙状态仍使用既有租约恢复，最多三次 claim。增加数据库回归：第一次忙不改变失败计数、不进入 awaiting_model；租约恢复后同一 operation 在第二次 claim 成功，无需重新启用图库。

## 已执行的专项证据

- Docker `TestFaceDeploymentAdapterIdentityAndCommit`：实际占满两条共享 HTTP 槽，验证激活不立即返回失败；等待可取消，释放槽后同一次加载成功。
- Docker `TestFaceRuntimeRemoteBusyContract`：busy、unavailable、错误身份、缺少身份、超长、畸形、未知字段、尾随 JSON、错误 HTTP 状态共 9 场景通过，均核对请求槽释放。
- Docker `TestFaceBusyKeepsBoundedRecoverableClaim`：同一操作忙→租约重排→成功通过（2.47 秒）。此测试使用合成分析器；前一项通过真实本地 HTTP stub 验证传输到 FaceRuntime 的映射，不冒充真实模型拥塞压测。
- `make fmt-check`、`make arch-check`、`git diff --check` 通过。

镜像：

- `foliopath-app:face-worker`：`sha256:d20dc2f204c737b81fb40af1269d3aa8fb9b94bc5b5a79aac21fd74a52e5d271`
- `foliopath-remote-ml-regression:local`：`sha256:c82fa3476012361cd7a85e1a241db311301686ca7f4de5f245616df20e745b6e`

最终组合回归命令：

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/face.test -test.timeout=3m && /out/remote-image-probe.test -test.run Face -test.skip "Probe$" -test.timeout=3m && /out/app.test -test.run "Deployment|Face|SemanticContentSource" -test.timeout=3m && /out/aimodel.test -test.timeout=3m && /out/api.test -test.run Face -test.timeout=3m && /out/sqlite.test -test.run "Face|AIOperationRecovery" -test.timeout=8m'
node spikes/int001-remote-ml/face-startup-probe.mjs
```

上述组合回归与最终镜像探针均通过（exit 0）。探针包含离线启动、双模型激活、服务恢复、正式 API 分析/幂等重放以及强杀后的真实租约恢复；自有临时容器、网络和合成数据卷已清理。两处复查问题关闭，A02/A03 重新关闭，P1 按既定范围通过；不是“已证明不存在任何缺陷”。
所有运行时检查只在本地 Docker；探针只读使用仓库合成媒体和临时独立数据卷，无 server222、真实媒体修改、提交或推送。
非空人脸、质量、容量、原生双架构、完整七项发布检查不在本次通过范围；face 的 opt-in 100k 容量项未启用。
