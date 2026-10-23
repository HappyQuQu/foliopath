# ADR-0026：原生双架构构建与已验证 digest 发布

- 状态：已接受（2026-10-23，用户确认优化方向）
- 范围：`NFR-COMP-001`、`NFR-OPS-001`；S5 发布供应链维护
- Owner：发布工具维护者
- 记录：[原生发布构建维护](../changes/FIX-2026-10-23-native-release-builds.md)

## 背景

旧流程在一个 AMD64 runner 上通过 QEMU 构建两个架构。项目包含原生媒体依赖和
Go/CGO 编译，模拟执行成本高；用户在接近一小时后取消了 run 39914307958。
BuildKit 的 GHA layer cache 不自动持久化 `RUN --mount=type=cache` 内容。
编号保留后续 AI 开发中的 ADR 空间，避免改动独立 AI 分支。

## 决定

1. `ubuntu-24.04`、`ubuntu-24.04-arm` 并行运行原生架构，每架构只构建一次最终镜像。
   Dockerfile、依赖版本、补丁、编译参数和部署单元保持原约束。
2. image plan 绑定仓库、完整源码 SHA、嵌入 VERSION、OCI labels 和最终标签。
   单架构按 digest 推送，在相同原生 runner 拉取同一 digest，检查平台、revision、
   version，然后以禁止重建模式运行现有发布镜像 smoke。
3. 仅两份来源、版本、仓库和架构匹配的成功记录可以发布标签。发布脚本检查各原生
   index 的 SPDX SBOM、SLSA provenance，以及合并后 descriptor 完整保留。
   合并只使用已验证 digest，不再构建最终镜像。
4. layer cache 按架构隔离；Go modules、Go build、npm cache mounts 按架构和
   Dockerfile/go.sum/package-lock 身份持久化。自有脚本在同一 builder 导入导出，
   读取 Dockerfile 已固定基础镜像，仅操作三个允许目录，不引入第三方缓存 Action，
   不挂载实际媒体或应用数据。
5. 同一发布队列不自动取消在途构建；原生任务各限时 45 分钟，准备和发布独立限时。
   保持已有 Release Please 和标签策略；PR 不获得发布 secrets。

## 替代方案与影响

QEMU 保留模拟执行瓶颈；交叉编译 CGO 需要额外工具链和媒体依赖维护；第三方
cache-dance Action 增加供应链来源。原生 runner 加自有薄缓存脚本复用现有工具链。
缓存可丢弃，不能替代 digest 验证。失败可能留下无最终标签的暂存镜像；冷缓存仍需
完整编译，不承诺固定分钟数。GitHub Release 与容器发布仍为既有两个阶段，前者成功
不能解释为容器已发布。本决定不替代其他 S5 readiness 门。

## 验证

`make arch-check` 执行 workflow 契约和 Python 协议回归；actionlint 检查 workflow。
`tests/release/build_cache_smoke.py` 用专用临时 builder 验证三个真实缓存往返。
远程冷/热构建、同 digest runtime smoke 和双架构 index 结果见关联维护记录。
