# 原生发布构建维护

- 状态：实现及本地验证完成，远程验证待执行
- 批准：2026-10-23 用户确认原生并行构建优化方向
- 目标：当前 Post-MVP 1.4 发布工具维护；无新增产品功能
- Requirement：`NFR-COMP-001`、`NFR-OPS-001`；保持 `NFR-SAFE-001`
- Gate：S5-007 候选供应链、S5-008 发布文档；不改变其他 readiness 判断
- Owner：发布工具维护者
- 决策：[ADR-0026](../adr/0026-native-release-image-promotion.md)

## 问题与合同

用户取消了接近一小时仍未完成的
[旧 QEMU 构建](https://github.com/HappyQuQu/foliopath/actions/runs/39914307958)。
改为原生 AMD64/ARM64 并行、每架构构建一次最终镜像，成功验证相同 digest 后才合并发布。

`scripts/native_release.py` 唯一拥有 image plan、成功记录和 promotion 校验；
`scripts/buildkit_cache_mounts.py` 唯一拥有三个允许 cache mounts 的往返协议。
workflow 只编排；`Dockerfile` 继续拥有基础镜像和产品构建。不访问实际部署、媒体、
数据库或原 AI 工作区。

## 本地证据

已成功：`make fmt`、`make arch-check`、`make generate-check`、`make lint`、`make test`、
`make test-integration`、`make test-e2e`、`npm --prefix web run build`、`git diff --check`。
`make test-e2e` 是测试专用容器 smoke，不等同最终发布镜像验收。

- `python3 -B -m unittest discover -s scripts -p 'test_native_release.py' -v`：10 项通过；
  覆盖来源/版本/架构混用拒绝、失败不生成记录、同 digest smoke、不重建 promotion、
  SBOM/provenance 保留及缓存目录/基础镜像协议。
- `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 -shellcheck= .github/workflows/dockerhub.yml .github/workflows/native-images.yml`：通过。
- `python3 tests/release/build_cache_smoke.py`：三个真实 BuildKit cache mounts 往返通过，
  专用临时 builder 完成后移除。

## 远程证据

待分支手动运行独立测试标签，记录冷缓存、热缓存及最终主分支运行。
不得以本地模拟协议测试替代实际双架构构建。旧 QEMU run 已由用户取消，不重启。
