# 多库视频调度、取消隔离与检索验收

CR-2026-030 / POST-MVP-5 接入验证阶段 / FR-INT-004/016。
Owner semantic/SQLite/app；沿用现有合同和全局单视频 worker，不更改并发预算。

## 本地实际环境

独立 Compose project `foliopath-multilib`，数据卷 `foliopath-multilib_candidate-app-data`。
只读媒体根 `/tmp/foliopath-multilib.uEIluH` 下有 alpha/beta 两个普通目录，
各含一段复制自既有合成夹具的 3 秒视频，不读取真实媒体。
模型缓存只读，AI 无媒体/数据库挂载或公开端口；主应用仅 localhost:6091。
主应用镜像沿用已验收取消修复版本
`sha256:39d8d0d74b6ab6f90c49e38b07382b99e7d13cf6665fd75ab207a4bba8f89193`。

## 自动化实际 HTTP 结果

脚本：`spikes/int001-remote-ml/multilib-cancel-probe.mjs`。
通过正式 setup/login、建库、扫描、故事板修复、开关、任务、取消和检索 API；
使用真实 AI 容器生成向量，不是数据库 stub。脚本限制固定项目及只读临时媒体挂载。

- 两个库任务共存：A running、B queued。当前生产组合只有一个视频 worker，
  不把排队共存描述为两个视频同时推理。
- 暂停本地 AI 后取消 A；无需恢复 AI，A 即进入 cancelled，完成数 0。
- B 保持 queued/running、完成数不变、错误未改变，没有随 A 取消。
- 恢复 AI 后 B succeeded、覆盖率 1/1，A 仍 0/1。
- A missing 重试 succeeded、覆盖率 1/1；A 原任务仍 cancelled，B 仍 succeeded。
- 按库视频语义检索各返回唯一资产，并核对资产 ID 为该库合成视频。

初次测试错误假设两个视频任务会同时 running，等待失败；人工恢复 AI 后清理
又遇到 already-unpaused，掩盖初始超时。已按实际单 worker 修正验收断言，
并让清理先检查暂停状态。没有为满足错误断言修改生产调度。
同时修正测试中命中资产 ID 的读取为合同定义的 `items[].asset.id`。

## 复跑

在该项目和挂载已启动且健康时：

```sh
# 首次不设置 FOLIOPATH_TEST_RESUME；密码通过环境提供，不写入脚本。
FOLIOPATH_TEST_RESUME=1 node spikes/int001-remote-ml/multilib-cancel-probe.mjs
make arch-check
```

需设置 `FOLIOPATH_TEST_PASSWORD` 为该隔离测试管理员密码。
修正后的实际流程成功两次，最后一次包含最终清理检查版本，全部断言 PASS。
`make arch-check`、`node --check` 及本轮已跟踪文档的 `git diff --check` 均通过。
测试容器已停止，合成夹具及隔离数据卷保留；未运行完整七项验证或浏览器 E2E。
脚本重新请求重建和取消，仅用于指定合成测试库。浏览器多库切换、吞吐/压力、
跨任务类型资源竞争和完整发行矩阵不属于本证据。人物生产接入仍未完成。
