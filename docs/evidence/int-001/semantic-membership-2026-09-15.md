# 图片新增/移除覆盖率子片（2026-09-15）

CR-2026-030 / POST-MVP-5 / FR-INT-005/015/016；Owner SQLite。
沿用扫描事务和既有语义进度表，不新增 API、模型状态机或迁移。

## 实现与范围

- 新索引图片/动画在扫描 upsert 事务中增加已有语义进度的 eligible；重复 upsert 不重复增加。
- 已启用且 ready 的库出现新候选时变为 building，避免仍声称全部处理完成。
- 完整扫描确认消失的图片，在删除资产及级联向量前按代次汇总移除数量；
  同步扣减 eligible、有效 completed 和已失效向量对应的 stale，计数不减为负数。
- 失败/不完整扫描不执行移除汇总；安全提交的新候选保留，旧索引也保留。
- 仅修改 SQLite 派生状态，不删除任何原始媒体文件。

验证入口：本地 remote-ml-regression 镜像内 SQLite race，匹配
`TestScanInvalidatesSemantic|TestFailedScanPreserves|TestSemanticSearch|TestSemanticEmbedding`，
3 分钟超时；新测试覆盖新增、重复 upsert、失败扫描、成功移除过期项、移除已完成项、
重复空扫描。补充案例后最终复跑全部 PASS / exit 0。
`make arch-check`、`git diff --check` 通过；两次回归镜像构建成功。

## 未覆盖

本次为仓储回归，未重新运行真实 AI/浏览器全链路或完整 SQLite 全包。
此前没有进度记录的库、历史失败项精确归属、指纹回退和并发全量回填仍需专项验收。
视频搜索也依赖保存的进度计数，尚未补齐视频源变化/新增/删除的对应覆盖率更新；
不能将图片子片标为视频已完成。未访问 server222 或真实图库。
