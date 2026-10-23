# FIX-2026-09-01：coser 私有人脸复核材料准备

- 目标版本：`POST-MVP-5` revision 2 / S2C
- 关联 Gate：`INT-S2C Privacy Ready`（保持 **Release No-Go**）
- 影响任务：`INT-241`、`INT-250`、`INT-251`
- 影响不变量：原媒体只读；候选聚类不得冒充身份 ground truth；未经正式质量 Gate 不允许整组归人

## 目的

操作者授权 `/Volumes/r0data/picture/coser` 仅用于本机功能验证，并确认目录中包含多个人。既有 smoke
只输出聚合计数，不能为逐脸身份复核提供材料。本修复增加隔离工具
`spikes/int001-ai/face_ground_truth_prepare.py`，把原图只读检测、五点对齐、AuraFace embedding、候选聚类
和人工复核清单串成可重复流程。

## 安全边界

- 原媒体树只读打开，不移动、改名或写入；输出必须位于媒体树外且初始为空。
- 模型按 catalog 的 size 与 SHA-256 复核，拒绝 symlink；媒体扫描不跟随 symlink。
- 私有输出包含 112×112 派生缩略图、embedding、相对路径和 `review.csv`，只保存在
  `/tmp/foliopath-coser-review/output`，不进入 Git、普通 CI artifact 或公开对象存储。
- `candidate_cluster` 只是模型提示；清单默认 `review_status=pending`、`identity_id` 为空，工具输出固定
  `identity_ground_truth=false` 和 `review_complete=false`。
- AuraFace 有限输出先用 float64 稳定归一化，再以显式 `einsum` 计算 cosine，避开本机 BLAS 对有限单位
  向量产生的浮点异常；极值和 300×512 分块回归覆盖该边界。

## 本地结果

- 九个顶层来源共发现 6,202 张支持格式图片；均衡上限选择 496 张，496 张全部解码。
- 首轮 496 张全部解码，308 张产生 309 个有限候选。原单链 0.6 阈值曾得到 83 簇和八个表面稳定大簇；
  扩大到每来源 300 张后，1,547 张全部解码，974 张产生 986 个候选，单链却把八个来源串成一个
  834-member 巨簇，证明首轮结论受到 bridge face 和样本规模偏差影响，已撤回“八个稳定身份”的表述。
- production 与准备工具改用 smallest-ID anchor coherence 后，首轮重算为 129 簇、3 个 `≥20` 候选簇；
  扩大样本重算为 316 簇、9 个 `≥20` 候选簇，834-member 巨簇消失。九个大簇仍为 pending review
  candidate，不能解释为九个已确认身份。
- 继续把每来源上限提高到 500 后，实际选择并解码 2,347 张，1,496 张图产生 1,515 个唯一、有限的
  512 维候选。anchor-coherent 阈值扫描在 `0.60/0.65/0.68/0.70/0.72/0.75/0.78/0.80` 时分别得到
  `17/8/2/2/1/1/0/0` 个 `≥20` 候选簇。对 `0.70` 的两个大簇逐脸拼图复核后，一个较一致，另一个至少
  含可疑异人；它们均保持 pending，不能自动提升为身份标签。该结果量化了当前目录的上限：提高阈值会
  快速失去每身份 20 张覆盖，降低阈值又引入错误合并，不能靠调参制造正式 `50×20`。
- `review.csv` 现在固定 LF 行尾，并在 summary 中报告 `candidate_clusters_at_least_20` 与
  `largest_candidate_cluster`，便于本地复核工具读取；这些计数仍明确不是身份 ground truth。
- 该目录明显不能单独证明正式 Gate 的 `50 identity × 20 image`，也没有五类 bias slice 标注。结果用于
  候选展示、人工逐脸确认和错误合并功能验证，不能产生正式 `face-quality-score` 通过报告。
- production SQLite 读取投影继续强制 `groupAssignmentAllowed=false`；本结果不启用自动整组创建或并入人物。

## 回归

```text
/tmp/foliopath-coser-review/venv312/bin/python -m unittest \
  face_ground_truth_prepare_test.py face_functional_smoke_test.py \
  face_arcface_functional_smoke_test.py

Ran 14 tests ... OK
```

本记录关闭“从授权目录生成可人工复核材料”的工具缺口，不关闭 50×20 ground truth、偏差质量、模型合规、
native 双架构或最终发布签署。
