# FIX-2026-09-02：人脸功能证据目录分组口径纠偏

- 目标版本：`POST-MVP-5` revision 2 / S2C
- 关联 Gate：`INT-S2C Privacy Ready`（保持 **Release No-Go**）
- 影响任务：`INT-241`、`INT-250`、`INT-251`
- 影响不变量：没有 face-level ground truth 时不得声称身份 recall、false-positive rate 或质量 Gate

## 问题

经操作者授权的本地功能样本包含多个人。目录只提供采样分组，不保证一个目录对应一个身份，也不保证
不同目录对应不同身份。旧版功能 harness 虽已明确 `identity_ground_truth=false`，但输出字段仍使用
`same_pair_recall`、`different_pair_false_positive_rate` 和 `false_positive_pairs`，容易把纯功能统计误读为
身份验证质量。

## 修复

- 报告 schema 提升到 v2，分组评分 owner 重命名为 `score_directory_group_pairs`。
- 指标改为 `within_group_accept_rate`、`cross_group_accept_rate` 和
  `cross_group_accepted_pairs`；pair 计数同步改为 `within_group_pairs` / `cross_group_pairs`。
- 报告固定 `group_semantics=directory-group-only-not-identity-ground-truth`，继续禁止持久化路径、人物名、
  crop 和 embedding。
- 用同一只读授权根、固定 YuNet/AuraFace SHA 和相同有界采样重新运行 135 张功能样本；79 个候选全部
  产生有限 512 维 embedding。结果只证明候选流水线与阈值统计可运行，不满足 50×20 ground truth、
  bias slice、99.5% anonymous-core precision 或发布准入。

## 回归证据

- 标准库单测验证 schema 不再出现 `same_pairs`、`different_pairs`、`same_pair_recall` 或
  `different_pair_false_positive_rate`。
- 平衡采样、pair 上限、symlink/超限输入和“不持久化向量”合同继续覆盖。
- AuraFace v2 聚合记录不包含文件路径、目录名、crop、embedding 或身份标签。
