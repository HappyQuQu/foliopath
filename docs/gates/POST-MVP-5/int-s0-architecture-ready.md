# INT-S0：本地智能媒体发现 Architecture Ready

## 当前判断

**No-Go（2026-08-18）**。只允许评审、隔离 spike、fixture/benchmark 和文档收敛；不得修改生产
OpenAPI、migration、后端、前端菜单或发布镜像。

## 进入依据

- Feature：[FTR-INT-001](../../features/intelligent-media-discovery.md)
- Change Record：[CR-2026-021](../../changes/CR-2026-021-intelligent-media-discovery.md)
- 技术方案：[智能媒体发现技术架构](../../architecture/intelligent-media-discovery.md)
- Proposed ADR：[ADR-0013](../../adr/0013-local-ai-runtime-and-derived-vector-index.md)
- Spike：[INT-001](../../spikes/int-001-ai-feasibility.md)

## Go 条件

- [ ] 产品负责人接受范围、非目标、用户流程、隐私告知和降级路径。
- [ ] 确定 target version、scope budget、owners；创建冻结 scope manifest。
- [ ] Workstream A 在原生 amd64/arm64 通过运行时、模型、资源、质量与许可门槛。
- [ ] Workstream B 冻结 vector 存储/索引方案并通过 100k、损坏恢复和容量门槛。
- [ ] Workstream C 证明核心聚类 precision；不通过时正式删除或降级人脸范围。
- [ ] ADR-0013 接受，明确 runtime/model/vector 引擎，不保留结构性 TBD。
- [ ] 模型获取策略已冻结：签名发行清单、真实下载源/镜像 owner、`/models:ro`、托管复制与直接读取
  的信任边界、失败语义和部署成本均有证据；无真实镜像时不承诺国内下载源。
- [ ] 数据分类、备份/恢复、清除、模型升级和人工关系迁移语义已评审。
- [ ] `R-024～R-030` 有 owner、Gate、fallback；严重风险没有无主项。
- [ ] S1 需要更新的 PRD、user flow、UI、data、API、security、deployment、testing 文档已列明。
- [ ] 任务估算与依赖可执行，不与当前 MVP/RC 或其他冻结切片争夺未批准 scope budget。

## 当前缺口

所有 Go 条件均未形成执行证据。当前文档只是方案，模型候选、性能数字和准确率都未验证；因此不能
称为“技术选型完成”或“可以开始开发”。

## 获准下一步

按 [任务清单](../../features/intelligent-media-discovery-task-list.md)执行 `INT-001～023`，完成后更新本
Gate。任何 production source、OpenAPI 或 migration 变化都必须等待本 Gate Go 和 S1 评审。
