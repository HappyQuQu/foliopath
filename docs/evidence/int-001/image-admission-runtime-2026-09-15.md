# 图片等待资源取消：实际双向混合任务验收

CR-2026-030 / POST-MVP-5 接入验证阶段 / FR-INT-016 / NFR-INT-007。
Owner semantic/resourcecontrol/app。沿用现有资源预算、取消通知和任务合同。

## 环境与脚本

本地 `foliopath-multilib` 双容器，alpha 合成视频、gamma 合成 PNG，媒体挂载只读。
重新构建并启动的主应用镜像：
`sha256:7e0327c4f1157055d054f19446b9d305b8b110e4db98a48ce084907a38b30e90`。
两容器健康；不访问 server222 或真实媒体。AI 无公开端口、媒体或数据库挂载。

扩展 `spikes/int001-remote-ml/mixed-admission-probe.mjs`，
`FOLIOPATH_TEST_CANCEL_TARGET=image|video` 指定等待并取消的类型，默认 video。
固定本地实例，后台额度暂设为 1，先运行持有任务，AI 暂停时再领取另一类任务并取消。
仍由 finally 恢复 AI 和原 backgroundConcurrency，无服务端测试状态伪造。

## 结果

图片方向：取消等待的图片任务用时 15208 ms，处理数 0，视频任务仍 running，
AI 仍 paused=true。恢复 AI 后视频 succeeded，图片 missing 重试 succeeded，
图片/视频 completed 均为 1，原图片操作仍 cancelled。未等待 120 秒租约恢复。

视频方向复跑也通过：15213 ms 取消，图片继续运行；恢复后持有任务及重试成功，
两类覆盖率均为 1/1，历史取消保留。两次脚本均退出 0；语法和架构检查通过。
这采用受控启动顺序和实际 HTTP 观测，未直接探测信号量队列；
结合前轮占满资源槽的组合测试，不推导为所有调度交错或性能证明。

```sh
# 设置隔离测试管理员 FOLIOPATH_TEST_PASSWORD 后执行。
FOLIOPATH_TEST_CANCEL_TARGET=image node spikes/int001-remote-ml/mixed-admission-probe.mjs
FOLIOPATH_TEST_CANCEL_TARGET=video node spikes/int001-remote-ml/mixed-admission-probe.mjs
node --check spikes/int001-remote-ml/mixed-admission-probe.mjs
make arch-check
```

未执行全套七项检查、浏览器 E2E 或压力测试。扫描、清理等类型等待期取消、
图片取消与最终提交/失败交错、人物生产接入及发行验收仍未完成。
测试结束停止容器，保留合成夹具和数据卷供复跑。
