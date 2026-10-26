# 语义源读取取消响应修复

范围 CR-2026-030 / FR-INT-002 / NFR-INT-002、003 / POST-MVP-5 P4，Owner imagevips/semantic；既有有界任务取消语义的例行修复，不新增 API、部署或持久化合同。

回归先行复现：在第一次 Read 取消 context，原实现仍调用 Read 共 33 次才返回 context canceled；读取错误与取消同时发生时还可能返回 invalid image。原 `io.ReadFull` 无块间取消检查。

方案：保留一次精确缓冲分配，每次最多读取 64 KiB，读取前后检查 context；取消优先于普通源错误，取消后不再发起读取。连续 100 次无进展则失败关闭，避免畸形 reader 忙循环；不新开 goroutine，也不关闭调用者拥有的流。成功输出不变，沿用 remote-image-v2 指纹。

限制：这是同步 Read 返回后的协作取消，不能强制中断已经阻塞的系统调用或正在执行的 libvips 内核，不宣称硬实时中断。

验证：回归先失败（`before.log`：33 次读取），修复后首次读取取消即停止；`fail=false/true` 两种取消场景通过。固定测试二进制带 race，在 1 CPU / 1 GiB / 无 swap / 无网络容器运行 9 项 `TestRemoteSemantic` 全通过，包括 108 MP、渐进 JPEG、源预算及大小变化。容器退出 0、OOMKilled=false；日志 `test-results/p4-cancellation/imagevips.log`。gofmt 检查、`make arch-check`、小样本结果门禁 Node 3 测试、`git diff --check` 通过；未运行全部七项仓库检查。

本地候选 `sha256:6325266035ffb7329d8e11ada7d84e1497dd8bc4ae5d407a6caa5affa52d09d3`（linux/arm64）使用既有 runtime/frontend，仅更新 Go 二进制。p4-large（app 1024 MiB / AI 2304 MiB，CPU 1+3，无 swap）下同一 100 匿名副本功能探针退出 0、functionalPass=true：

- 图片语义 90/90，失败 0，16125 ms。
- 人脸 90/90，失败/降级 0，45307 ms。
- 视频 10/10，失败/降级 0，10081 ms。
- 标签 90/90；中英文图片/视频搜索 8 次成功，34～55 ms（非 P95）。
- 样本 hash 前后不变；期间两容器 OOMKilled=false、RestartCount=0；结束内存快照 app 267 MiB / AI 2.017 GiB（非峰值），测试环境已清理。

报告 `test-results/p4-cancellation/sample.json`，日志同目录。真实样本仅为此前授权的匿名副本，默认自动化回归仍为合成数据。本轮不访问或停服 server222，也不提交推送/部署。该最新候选尚未重跑 amd64、升级/回退或全仓库发布检查；此前这些项目通过的是上一固定镜像。质量、长期稳定性及发行仍待验收，P4 0/8、总计 18/34 不变。
