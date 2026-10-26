# 自动标签非空建议、审核与清理实际闭环

CR-2026-030 / POST-MVP-5 / FR-INT-004、FR-INT-016 / Owner app/semantic/curation。
本轮扩展本地验证脚本，不改生产代码或阈值。

查看合成 gamma.png 确认为蓝紫渐变图；过长的描述词条首次被标签创建 API 以 400 拒绝，
改用长度合规的 `blue purple gradient`。未降低 0.65 候选阈值。
真实 CLIP 生成建议 confidence=0.6619495，生成任务 succeeded，覆盖 1/1。

正式 HTTP 链路验证：

1. 使用当前建议版本、curation revision 接受建议，返回 accepted。
2. 相同幂等键与请求重放，结果一致，正式 tag_2 存在。
3. accepted 列表包含该建议；取得审核 ETag 后提交清理。
4. 清理任务 aio_bbc600477d90f3ad923aef3fb283e590 成功、完成项 1；accepted 列表为空。
5. 清理前后资产 curation 对象一致，正式标签未被删除。

生成任务 aio_9129236a3a11ed9a714036da1b508d98，建议 ais_6a9624e1e1dacd28431cc4c55f2f2541。
沿用镜像 `sha256:60579ee74a087a12cf30a31b3cf75641c8f5638399ac103121c1ffb55325f079`，
本地隔离 Docker 项目 foliopath-multilib、真实离线模型、只读合成媒体。
不接触 server222 或真实图片，不修改原始媒体。

```sh
# 设置隔离管理员 FOLIOPATH_TEST_PASSWORD 后执行
node spikes/int001-remote-ml/tag-generation-probe.mjs
node --check spikes/int001-remote-ml/tag-generation-probe.mjs
make arch-check
```

闭环脚本退出 0，语法与架构检查通过。结束恢复原词表成员（revision 正常推进），停止两容器。
测试正式标签刻意保留以证明清理边界；重复运行需未分配该标签的新合成库状态，
脚本不自动移除正式标签。此证据不代表一般图片识别质量或阈值已校准。
尚未覆盖 dismiss、跨库混合审核账本、非空实际取消交错、标签生成取消矩阵、浏览器流程、
人物、容量或发行；未执行完整七项检查。
