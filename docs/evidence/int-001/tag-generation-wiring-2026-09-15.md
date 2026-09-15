# 自动标签生成：主应用组合与首次实际任务

CR-2026-030 / POST-MVP-5 / FR-INT-004、FR-INT-016 / Owner app/semantic。
合同沿用 OpenAPI 标签生成任务、受控词表、图片代次与 SQLite 标签任务契约。

接入 ControlledTagEmbeddingBuilder（复用当前 textEncoder，独立部署为 BoundEncoder）、
ControlledTagPlanBuilder、TagJobProcessor、单 worker、共享后台额度、服务准入和正式路由。
文本与图片复用同代次，评分读取已入库图片向量；不会自动补齐缺失图片，也不自动写正式标签。
候选默认阈值明确为 confidence=0.65，即 cosine=0.30，最多 5 项；这是未校准的建议过滤值，
不是概率或质量承诺，所有建议仍需人工审核。正式质量校准仍为待办。

## 验证

Docker 回归镜像 `sha256:06efba90af0b850f2651ed9ccb46ce2d4dee687d7fe1fee3706e90a145e81e70`：
SQLite 匹配 3 项、semantic 全包、app 的 Semantic/AI 匹配测试通过，架构检查通过。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "TestTagJob|TestControlledTag" -test.v -test.timeout=3m && /out/semantic.test -test.timeout=3m && /out/app.test -test.run "Test.*Semantic|Test.*AI" -test.timeout=3m'
make arch-check
```

实际新镜像 `sha256:60579ee74a087a12cf30a31b3cf75641c8f5638399ac103121c1ffb55325f079`。
本地 foliopath-multilib 合成 gamma PNG、真实离线 CLIP，通过正式 API 创建测试词条
“a geometric image”、发布词表、提交 all 生成。任务
`aio_337db440c83132f143b91c98f82c94a0` 为 succeeded，1/1；列表 coverage complete=true，
items=[]。零建议是本次真实结果，不宣称已有非空建议或审核闭环。

脚本 `spikes/int001-remote-ml/tag-generation-probe.mjs` 通过，结束恢复原词表成员；
词表 revision 正常推进，测试 tag_1 保留在隔离实例中。两容器已停止，媒体未修改。
未接触 server222、未执行全套七项检查；非空建议/审核/清理、标签生成取消矩阵、
质量阈值校准、人物、容量和发行仍待完成。
