# 远程人脸代次原子提交

CR-2026-030 / FR-INT-009 / NFR-INT-004，POST-MVP-5 S2。
Owner：face 兼容身份合同，SQLite 原激活事务；app 后续组合。沿用 ADR-0019/0020，不增加部署单元或数据库表。

## 先行合同

DeploymentIdentity 包含固定 face-v1/buffalo_l/512、运行时、缓存、输入变换及质量规则摘要、detector/embedder hash 和经现有 owner 校验的阈值 profile。
完整合同指纹为确定性 SHA-256，不是信任证明；注册模型的 content_hash/source_identity 必须等于指纹。
FaceGeneration 通过 model_id 绑定模型完整指纹，同时持有组件与阈值 hash；提交时必须全部匹配。
复用原事务完成旧人脸代次退役、新代次插入和操作成功，保留语义代次及各库原聚类指针。
service 与文件入口互斥；取消、旧 revision、身份不匹配或事务中途失败不得改变现有代次。
不自动套用旧 YuNet 阈值，不宣称未校准的 buffalo_l profile 为生产就绪。

## 实现及验证

已实现 face 身份 owner、组件/profile 匹配和 SQLite 原事务的 service 专用入口。无新表，`model_id → ai_models.content_hash/source_identity` 绑定完整身份。

本地 Docker 带 race 的 `face.test`、`aimodel.test` 全包通过；SQLite 定向回归通过：

```sh
/out/face.test -test.timeout=3m
/out/aimodel.test -test.timeout=3m
/out/sqlite.test -test.run 'TestFaceDeploymentActivationAtomicIdentity|TestFaceModelActivation|TestFaceServiceCannot' -test.v -test.timeout=3m
```

新增 10 场景：成功、错完整身份、错组件、错阈值、旧 operation revision、旧 availability revision、已取消、插入阶段失败、文件模型走服务入口、服务模型走旧入口。成功后重复提交拒绝；失败旧人脸代次仍 active、无部分新代次、操作状态保留；所有场景保留 CLIP 活动指针及各库原人脸可见指针。测试使用合成身份/profile，不是模型质量证据。

app 进一步实现 `faceDeploymentActivationRuntime`，验证独立服务身份后提供 Metadata，提交时附带完整 face 身份；配置/启动装配尚未接入，归 A02。真实模型激活和分析入库仍未完成。

app `-test.run Deployment` 四项测试（含人脸适配新增测试）与 remoteml 非显式探针回归全部通过，使用本地 Docker race 二进制；尚无本轮真实模型激活证据。架构检查、Go 格式化和 diff 空白检查通过。没有执行完整七项验证，没有修改运行数据库或 server222。
