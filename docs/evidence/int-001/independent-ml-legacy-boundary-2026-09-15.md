# 独立 AI 接入前：旧运行时隔离回归

日期：2026-09-15。范围：POST-MVP-5 / CR-2026-030 / ADR-0019 兼容修补。

发现：旧图片、文本工厂仅验证 active 与 768 维。空间绑定已经可以持久化，
但同维度的绑定代次仍可能通过旧入口；这是接入前发现的潜在误路由，不是已发生的生产混用。

修复：semantic.RequireLegacySigLIPGeneration 唯一拥有旧准入策略，两个 app 工厂共同使用。
仅 active/768/空指纹允许继续旧模型验证；任何非空指纹均拒绝。旧模型来源验证保持不变。

## 执行结果

本地 Docker Desktop Linux/arm64，测试镜像 `foliopath-remote-ml-regression:local`，
本次构建 config digest：`sha256:46c894f7d0d1bee1ef9bccb3d747e6839f7883df2ab91af2e8e89a27c04d57d5`。
测试容器无网络、只读根文件系统、无媒体/缓存/数据库宿主挂载；临时文件使用 tmpfs。

```sh
docker build --target remote-ml-regression -t foliopath-remote-ml-regression:local .
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache \
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm regression \
  '/out/semantic.test -test.timeout=3m && /out/app.test -test.v -test.run "^TestSemantic" -test.timeout=3m'
```

构建成功；semantic 全包 PASS；app 的 `^TestSemantic` 子集 PASS；联合命令 exit 0。
两类测试二进制以 `-race -tags=libvips` 编译。环境变量仅满足 Compose 插值，regression 无缓存挂载。

- 新策略测试覆盖 5 种状态 × 2 种维度 × 3 种指纹。
- 新工厂测试覆盖图片/文本已绑定拒绝及旧空绑定继续模型检查，避免缺依赖导致假通过。
- 既有会话复用、关闭、异常向量和取消来源校验等语义回归通过。
- 本地宿主静态架构检查 `make arch-check` PASS；`git diff --check` PASS。

本轮未跑全仓 lint/生成/E2E，也未重跑 SQLite 全包；此前全包证据保留在
[Docker 回归](local-docker-regression-2026-09-15.md)。未使用 server222 或真实照片。
尚未完成：部署配置/身份事实核验、主应用独立服务组合、激活和完整索引查询；未提交、推送或部署。
