# 取消修复后的本地运行镜像回归

CR-2026-030 / POST-MVP-5 接入验证 / FR-INT-016 / NFR-INT-007。
Owner semantic/scanner/resourcecontrol/app；本轮重建与验证，无生产代码变更。

主应用实际运行镜像为
`sha256:1c80669f213cf6ce5c55c61eabe2a5eeb64e8e0a4e29ae69492d011b7dce5ca0`，
已包含图片与清理终态取消优先修复。Docker inspect 确认主应用只绑定
127.0.0.1:6091，AI 无主机端口，两容器根文件系统只读。
继续使用 foliopath-multilib 隔离数据卷、合成视频/PNG 和只读媒体挂载，
未使用 server222 或真实媒体。

运行 mixed-admission-probe.mjs 的 image/video/scan/clear 四种目标：
后台额度 1、先运行持有任务、暂停 AI、领取等待任务并取消，恢复后验证成功及历史保留。
脚本 finally 恢复暂停状态和原并发设置；清理恢复是重新启用/回填，不是继续删除。

四项运行全部通过，整体退出 0：图片取消 15143 ms、视频 15151 ms、扫描 15219 ms、
清理 15204 ms。各次取消时持有任务仍 running、AI 仍 paused；恢复后持有任务及
重试/重新启用回填成功，图片和视频覆盖均为 1/1，历史取消记录保留。
脚本语法检查、make arch-check 通过。结束后停止两个容器，保留合成夹具与测试卷。

```sh
# 已设置隔离测试管理员 FOLIOPATH_TEST_PASSWORD
for probe_target in image video scan clear; do
  FOLIOPATH_TEST_CANCEL_TARGET="$probe_target" node spikes/int001-remote-ml/mixed-admission-probe.mjs || exit
done
node --check spikes/int001-remote-ml/mixed-admission-probe.mjs
make arch-check
```

此轮是受控启动顺序的实际 HTTP 回归，不直接观测信号量，也不能声称在真实服务
中精确重现了最终事务交错。精确边界证据分别见
[图片处理器](image-processor-terminal-2026-09-15.md)与
[清理批次](clear-batch-cancellation-2026-09-15.md)。
未运行全套七项检查、浏览器 E2E、压力容量或其他任务类型完整取消矩阵。
人物生产接入及发行仍待完成。
