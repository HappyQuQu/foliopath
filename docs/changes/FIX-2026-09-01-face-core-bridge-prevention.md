# FIX-2026-09-01：人脸 core bridge 传递误合并防护

- 目标版本：`POST-MVP-5` revision 2 / S2C
- 关联 Gate：`INT-S2C Privacy Ready`（保持 **Release No-Go**）
- 影响任务：`INT-243`、`INT-250`
- 影响不变量：匿名 core 不得通过 pairwise bridge 把互不相似的身份传递合并；cannot-link 永远优先

## 问题

`ClusterFaces` 原先把所有达到 `CoreSimilarity` 的 pair edge 做并查集连通分量。该单链语义允许
`A≈B`、`B≈C` 在 `A` 与 `C` 低于 core 阈值时仍形成同一个 core。小型合成容量夹具只有完全相同 pair，
未覆盖真实妆容、角度和多人图片产生的 bridge face。

操作者授权的 `coser` 扩大只读运行首次给出可复现实例：1,547 张全部解码，974 张产生 986 个有限候选；
0.6 单链阈值把八个不同顶层来源经少量 bridge 串成 834-member 巨簇。目录不是身份 ground truth，不能用它
计算正式 FPR，但跨八个明确不同来源的传递合并足以证明候选分组不安全。

## 修复

- core 改为确定性 smallest-ID anchor-coherent component。合并后每个迁入成员必须与新的稳定 anchor 达到
  `CoreSimilarity`；bridge face 可按 `EdgeSimilarity` 附着，但不能成为传播 core 合并的桥。
- component root/anchor 继续由排序后的最小 opaque face ID 唯一决定，输入顺序不影响结果。
- cannot-link 的 component-wide 检查保持不变并继续优先于模型相似度。
- exact 与 LSH 大集合路径共享同一 anchor 检查；没有新增部署单元、持久化字段或 API 行为。
- 私有复核准备工具采用相同语义。扩大样本重算后巨簇消失，得到 316 个候选簇，其中 9 个达到 20 张；
  这些仍是 pending review candidate，不是九个已确认身份，也不关闭 50×20 Gate。

## 回归与容量

- 三向量 fixture 固定 `A·B=0.8`、`B·C=0.8`、`A·C=0.28`：A/B 为 core，C 只能为 edge；正反输入顺序
  均通过，定向 Go 回归连续 20 次通过。
- `go test -race ./internal/face -run 'TestClusterFaces' -count=1` 通过。
- `FOLIOPATH_RUN_CAPACITY_TEST=1 GOMAXPROCS=4 go test ./internal/face -run TestClusterFaces100KCapacity -count=1 -v`
  继续通过 100,000×512 paired-core 与 singleton 两个极端：7.247 秒、Go memory sys 421,151,080 bytes，
  指纹仍为 `ed978ca7f471ba742a38f680cebb5f83481b8f622a70b005b906e654f2b706d4`。
- 私有工具及既有 smoke 共 13 个 Python 回归通过。

该修复关闭 production core 的 bridge 传播缺口，不证明真实身份 precision，也不启用
`groupAssignmentAllowed`。
