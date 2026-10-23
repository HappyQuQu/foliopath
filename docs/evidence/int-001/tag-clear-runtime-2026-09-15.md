# 标签审核清理：本地运行镜像验收

CR-2026-030 / POST-MVP-5 接入验证 / FR-INT-016 / NFR-INT-007。
Owner semantic/SQLite/resourcecontrol/app。新增验证脚本分支，无生产合同变化。

已重建并启动本地主应用镜像，包含标签审核清理准入与终态修复：
`sha256:f0c39a5f209377d29c3eda7f532d63cb7411bc934f3ceeb0f7528ddd460f14d4`。
沿用 foliopath-multilib 合成媒体和隔离数据卷、只读媒体挂载、loopback 6091、私网 AI。
未访问 server222 或真实媒体。

mixed-admission-probe.mjs 新增 tag-clear 目标：从正式标签建议接口读取审核 ETag，
用正式清理 API 提交；后台额度 1，视频持有资源、AI 暂停时取消已运行的标签审核清理。
15215 ms 后 cancelled、完成数 0，视频仍 running、AI 仍 paused。
恢复 AI 后视频 succeeded，重新提交清理 succeeded，原取消历史保留；图片/视频覆盖各 1/1。
等待任务 ID：aio_bcc76feeec324e8a8385dd306870347e。

```sh
# 设置隔离测试管理员 FOLIOPATH_TEST_PASSWORD 后执行
FOLIOPATH_TEST_CANCEL_TARGET=tag-clear node spikes/int001-remote-ml/mixed-admission-probe.mjs
node --check spikes/int001-remote-ml/mixed-admission-probe.mjs
make arch-check
```

脚本退出 0、语法与架构检查通过，恢复原额度后停止两个容器，保留测试卷。
本轮未预置审核记录，不能算非空审核批次的实际服务删除验收；
非空与正式标签保留证据见[组合测试](tag-clear-batch-2026-09-15.md)。
受控顺序的 HTTP 观测不等于直接信号量观测或全部并发交错证明。
未执行全套七项检查、浏览器 E2E 或容量测试；其余任务矩阵、混合审核请求账本、
人物生产接入与发行仍待完成。
