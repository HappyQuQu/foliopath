# 共享激活调度与主应用适配

2026-09-15；POST-MVP-5 / CR-2026-030；本地 Docker Linux/arm64。

## 实现

- 内部 StartDeploymentActivation 与旧准入复用私有 owner：同一请求 hash、幂等记录、信号与队列。
- ActivationWorker 组合期可安装一次 DeploymentActivationRuntime。service 不读本地 manifest、文件或 ONNX；未配置失败关闭，加载/提交复用原 worker 流程。
- app.deploymentActivationRuntime 只接受已绑定 HTTP 客户端及匹配 SpaceIdentity，加载前重新核对身份；提交携带固定空间，交给既有 SQLite 原子提交。
- VerifyIdentity 复核复用同一 HTTP 客户端和两在途请求预算，无额外无界连接池。
- databaseService 补充该提交接口透传；run.go 未创建新适配，正式启动行为不变。

## 验证

Docker 构建 `--target remote-ml-regression` 成功；随后执行：

```sh
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache \
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm regression \
  '/out/aimodel.test -test.timeout=3m && /out/app.test -test.run "TestDeploymentActivation|TestSemantic" -test.v -test.timeout=3m && /out/remote-image-probe.test -test.skip Probe -test.timeout=3m'
```

联合命令 exit 0，各测试二进制启用 race；aimodel 全包、app 相关子集、remoteml 全部非 probe 通过。
新增覆盖：服务请求幂等、未配 runtime 拒绝、禁止替换已配 runtime、服务 load/commit 路由；
app 身份重核、运行时改变拒绝、空间错配/未绑定客户端拒绝、提交保留固定空间。
worker 路由为合成桩、app HTTP 为 httptest；本轮不是完整 Run 循环或真实模型推理证据。
既有取消/失败测试随 aimodel 全包执行，未新增专门的服务中途取消集成场景。

宿主静态 `make arch-check` 和 `git diff --check` 通过。未跑全仓 lint/生成/E2E 或 SQLite 全包。
没有 server222 操作、真实媒体读取、应用数据库变更、提交/推送或发布。
仍待：部署配置读取、启动实例化/服务不可用恢复、完整应用 API/UI 联调与后续发行验证。
