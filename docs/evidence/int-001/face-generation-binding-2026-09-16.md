# 人脸任务代次绑定：A03 增量

2026-09-16；CR-2026-030 / FR-INT-009 / NFR-INT-004；POST-MVP-5 S2。
Owner：face / SQLite。仅使用本地 Docker、合成数据；未访问 server222 或真实媒体。

## 实现和边界

- `face.DeploymentJobProcessor` 每次 claim 建立独立代次绑定；复用原 `JobProcessor` 的进度、租约和终态逻辑，不另建状态机。
- 推理前后复核 generation ID、完整部署指纹、512 维、active 状态和模型可用性；中途切换/离线时不返回结果，空结果不例外。
- 空任务的控制检查也验证部署身份；聚类阈值来自同一份已验证部署 identity。
- SQLite 单条关联查询读取代次与模型；仅 service 人脸模型、source_identity 与 content_hash 一致才提供指纹。旧文件模型不能被误用为远程模型。
- 不新增 API、表或部署单元。以上代码尚未接入主进程 worker；前后检查不替代最终写入事务隔离。最终提交的可用性复核、worker/资源预算/安全媒体组合及正式入队恢复仍属 A03 未完成工作。

## 执行结果

构建 `foliopath-remote-ml-regression:local` 成功，配置摘要：
`sha256:7e3ec50381dfb2a6b34915cada5cd0e0e752acc27af7723a5a329f32c24140a4`。

```sh
sed '1{/^# syntax=/d;}' Dockerfile | docker build --pull=false --target remote-ml-regression -t foliopath-remote-ml-regression:local -f - .
docker run --rm --network none foliopath-remote-ml-regression:local '/out/face.test -test.v -test.timeout=3m && /out/sqlite.test -test.run TestFaceDeploymentActivationAtomicIdentity -test.v -test.timeout=3m'
make arch-check
git diff --check
```

- face race 二进制全包通过；其中新增绑定测试 11 场景：正常、错误 ID/指纹/维度、退役、离线、读取失败、推理中切代、空结果中离线、输出维度不符、上下文取消。
- `TestClusterFaces100KCapacity` 因未设置专用环境变量而跳过，不是容量通过证据。
- SQLite 原激活矩阵 10 场景通过（22.59 秒），成功场景额外验证新 service 指纹及旧文件代次无远程指纹。
- `make arch-check`、`git diff --check` 通过。新增/修改 Go 文件已执行 gofmt。
- 本轮未跑完整七项检查、真实非空模型、人脸正式 API/UI 或压力测试。

结论：A03 安全前置子步骤通过，A03 未关闭；清单仍为 **2/34 已关闭、32 项未关闭**。
