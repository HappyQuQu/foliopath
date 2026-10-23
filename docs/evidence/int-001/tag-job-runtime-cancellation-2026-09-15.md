# 标签生成：非空任务实际等待取消与恢复

CR-2026-030 / POST-MVP-5 / FR-INT-004、FR-INT-016、NFR-INT-007。
Owner app/semantic/resourcecontrol。扩展验证脚本，不改生产逻辑或阈值。

重建主应用镜像并启动本地 foliopath-multilib：
`sha256:aa55d825b91c06ea31b6c1c79fee16b194faeb886e48a0b91ad1a6e0d450fc7d`。
同一只读合成图库、私网真实 AI 服务，不使用 server222 或真实媒体。

mixed-admission-probe.mjs 新增 tag 目标。发布含既有测试词条的临时受控词表，
断言标签生成 totalItems=1。后台额度 1，视频先运行、AI 暂停后领取标签任务再取消。
标签任务 aio_dffdf2c3a87c16c1bcb9030114d11d9e 在 15235 ms 后 cancelled、completedItems=0，
视频仍 running、AI 仍 paused。恢复后视频成功，标签 missing 重试 succeeded，
标签覆盖 1/1、complete=true；图片/视频覆盖也各 1/1。
前后正式 curation 对象一致，取消历史保持 cancelled。

```sh
# 设置隔离管理员 FOLIOPATH_TEST_PASSWORD 后执行
FOLIOPATH_TEST_CANCEL_TARGET=tag node spikes/int001-remote-ml/mixed-admission-probe.mjs
node --check spikes/int001-remote-ml/mixed-admission-probe.mjs
make arch-check
```

实际脚本、语法与架构检查均通过。结束恢复原词表成员和后台额度、停止两容器；
词表 revision 正常推进，原正式测试标签保留。
本例“非空”指有 1 个待处理候选，不要求重试产生新建议（正式标签会抑制重复建议）。
受控启动顺序不等于直接观测信号量；未覆盖已开始文本推理/评分后的取消或非空最终提交交错。
未执行全套七项检查、浏览器 E2E 或压力测试。跨库审核账本、质量、人物、容量及发行仍待完成。
