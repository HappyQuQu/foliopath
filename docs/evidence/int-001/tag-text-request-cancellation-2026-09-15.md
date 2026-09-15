# 新词条文本编码请求取消与恢复

CR-2026-030 / POST-MVP-5 / FR-INT-004、FR-INT-016、NFR-INT-007。
Owner app/semantic/remoteml。本轮新增验证脚本，无生产逻辑变化。

本地 foliopath-multilib、合成 gamma 图、实际 AI 服务，沿用镜像
`sha256:aa55d825b91c06ea31b6c1c79fee16b194faeb886e48a0b91ad1a6e0d450fc7d`。
创建唯一新词条并发布新词表，排除该 snapshot 下已有文本向量缓存；暂停 AI 后
提交 1 候选标签任务，不再安排另一持有任务。任务 running/完成数 0 后发起取消。

这是实际编码链路等待服务返回的受控测试，不直接观测 ONNX 内核执行，也不能声称
已经在内核计算中断点注入取消。恢复服务后重新生成须成功完成 1 项；
取消历史保留，正式标签/收藏/curation 快照不变。

首次运行取消成功（14209 ms），重试也成功，但脚本误把创建词条前的全局 curation
revision 与创建后比较，导致断言失败。已将基线移至词条创建后，保留严格对象比较再复跑。
没有修改产品来迎合测试，也没有忽略版本字段。

最终完整复跑通过，exit 0：任务 aio_bb1cf437dbcb8b2a1f36b18de43f3f0c 在 14220 ms 后
cancelled，AI 仍 paused、完成数 0、建议为空；重试 aio_0a1b08e7cb80fecf3f45377eddbc244a
succeeded/1 项，curation 和取消历史保持。语法及架构检查通过，结束停止两容器。

```sh
# 设置隔离管理员 FOLIOPATH_TEST_PASSWORD 后执行
node spikes/int001-remote-ml/tag-encoding-cancel-probe.mjs
node --check spikes/int001-remote-ml/tag-encoding-cancel-probe.mjs
make arch-check
```

finally 恢复 AI 与原词表成员，正常推进词表 revision；新测试词条保留在隔离实例中。
无真实媒体修改、server222、完整七项检查或压力测试。
跨库审核账本、质量/浏览器/人物/容量与发行仍待完成。
