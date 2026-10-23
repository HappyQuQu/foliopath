# 原生发布构建维护

- 状态：实现、本地验证、主分支和 1.4.0 镜像发布完成
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

- `python3 -B -m unittest discover -s scripts -p 'test_native_release.py' -v`：12 项通过；
  覆盖来源/版本/架构混用拒绝、失败不生成记录、同 digest smoke、不重建 promotion、
  SBOM/provenance 保留及缓存目录/基础镜像协议。
- `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 -shellcheck= .github/workflows/dockerhub.yml .github/workflows/native-images.yml`：通过。
- `python3 tests/release/build_cache_smoke.py`：三个真实 BuildKit cache mounts 往返通过，
  专用临时 builder 完成后移除。

## 远程证据

- [首次冷构建 39920923695](https://github.com/HappyQuQu/foliopath/actions/runs/39920923695)：
  AMD64 构建 7 分 10 秒、ARM64 构建 5 分 51 秒；两边相同 digest 的 runtime smoke
  均通过，但缓存导出遇到 Linux 非 root exporter 写入 Go 模块 0555 目录失败，故未发布。
  修复仅对导出副本增加目录 owner 写权限，并把只读模块目录加入真实缓存往返回归。
- [修复后运行 39922276637](https://github.com/HappyQuQu/foliopath/actions/runs/39922276637)：
  完整流程 2 分 38 秒，AMD64/ARM64 layer cache 命中后的构建分别 11 秒、17 秒；两边
  runtime smoke 分别 74 秒、75 秒。导出、成功记录上传、SPDX/SLSA 检查、双架构 index
  合并和 Docker Hub Overview 全部成功。测试标签 `native-check-20261023` 未更新 latest。
  该轮证明 layer cache 和发布链路，首次失败时未成功保存的编译 cache mounts 不算热缓存证据。
- [main 首次运行 39923007181](https://github.com/HappyQuQu/foliopath/actions/runs/39923007181)：
  两架构镜像构建和 runtime smoke 通过；AMD64 成功保存 457,760,922 字节编译缓存。
  ARM64 缓存导出停滞，因此加入缓存操作独立超时及失败警告后继续的规则；不能让可丢弃
  缓存阻止已验证镜像发布。每次缓存 builder 调用限 60 秒，四个缓存步骤各限三分钟。
  超时回归确认旧快照仍保留。该轮已取消，最终结果见后续成功运行。
  日志显示停在 local exporter 的逐文件复制阶段；后续改为单一 tar 流导出并通过
  Python 仅解包普通文件/目录，拒绝路径逃逸、符号链接和特殊文件；失败保留旧快照，缓存布局保持兼容。
  GHA cache 受分支可见性约束，不能假定 main 直接复用测试分支缓存。
- [main 最终运行 39925599194](https://github.com/HappyQuQu/foliopath/actions/runs/39925599194)：
  完整流程 3 分 29 秒；两架构构建、同 digest smoke、缓存及合并发布全部成功。
  AMD64 恢复先前完整编译缓存；两种架构的 layer cache 构建分别 8 秒、15 秒。
- [1.4.0 正式发布 39926110946](https://github.com/HappyQuQu/foliopath/actions/runs/39926110946)：
  完整流程 5 分 20 秒，包含 VERSION 变化引起的应用构建、双架构 runtime smoke、
  SBOM/provenance 校验及 Docker Hub Overview。AMD64/ARM64 构建分别 70 秒、93 秒；
  完整缓存 tar 导出分别 29 秒、17 秒，ARM64 逐文件导出停滞已消除。
  `1.4.0`、`latest`、`sha-37bcfc4` 推送同一 index：
  `sha256:f6eda8161967a175b58f49940903cc035cc2f09ae5210e891c97a475c015f301`。
  原生成功记录的 source 均为 `37bcfc454dcd10d24096b053e144f7d98dbe8580`，version 均为 `1.4.0`。
  `git diff --exit-code v1.4.0 HEAD -- Dockerfile go.mod go.sum cmd internal migrations web`
  已证明维护提交的生产构建输入与 `v1.4.0` 一致。

本记录的最终证据更新仅修改文档，使用 `[skip ci]` 避免再触发相同生产输入的镜像构建；
源码、双架构真实验收及发布结果由上述成功运行固定，不把文档提交解释为新的构建证据。

旧 QEMU run 普通取消后仍在进行，已补发 force-cancel 并确认最终 `cancelled`，不重启。
