# POST-MVP-5 原生容量回归收敛

- 日期：2026-09-02
- 状态：**Implemented / paired native baseline verified**
- 类型：已批准 S4 容量切片内的例行修复
- Requirement：`NFR-INT-002`、`NFR-INT-003`、`NFR-INT-005`、`INT-403`、`INT-407`
- 目标版本与阶段：`POST-MVP-5` revision 2，S4 Release evidence
- Owner：catalog / scanner / QA
- 关联 Gate：[INT-S4](../gates/POST-MVP-5/int-s4-current.md)

## 问题与修复

原生双架构 baseline run `33609061702` 正确失败关闭，暴露两个容量预算违规：

- 首代 10 万媒体扫描在 amd64/arm64 分别为 238.019s/213.346s，超过 120s；
- 两页 production search keyset P95 分别为 255.112ms/398.499ms，超过 250ms。

搜索排序与 hydration 查询本身均在预算内；超时来自每个游标页重复执行相同的全量分类计数。资产游标
payload 升级为 v2，把第一页算出的 `all/images/videos` 放入已认证 payload。该 payload 继续绑定规范化
query fingerprint 与 library generation/global catalog revision；篡改、旧版本、查询变化或代次变化均拒绝，
后续页因此可安全复用计数而不重复扫描 FTS 结果。

扫描路径在首次发现资产时原先仍执行两个注定为空的 stale-derived `DELETE`。现在只有资产此前存在且 source
fingerprint 改变时才清理旧 thumbnail/storyboard 状态；资产与父目录状态按最多 200 项有界预取，资产本体按
最多 60 项批量 UPSERT，每个 scanner/reconcile 事务末尾再用一条 bounded UPSERT 接纳该批 grid media jobs。
预取仍记录父目录是否属于当前 generation，逐 entry 按 walker 顺序复核，不能借批处理接受尚未出现的父目录。
已有资产变更、只见代次变化与新资产的业务语义不变。这样消除了首代 10 万资产的 20 万次无效索引删除及
逐资产 SQLite 往返，同时保留单事务、有界 batch、`secure_delete=ON` 和真实隐私删除证据要求。

## 验证

- `go test ./internal/catalog ./internal/store/sqlite ./internal/scanner`：通过；
- `make spike-capacity`（darwin/arm64，4 CPU，10k 目录/100k 资产）：通过，零预算违规；
  - scan `29,735ms <= 120,000ms`；
  - search keyset P95 `83,230us <= 250,000us`；
  - first/second page P95 `67,106us / 15,985us`；
  - peak Go heap `42,324,120 bytes`，DB+WAL `157,470,720 bytes`。

中间 source `0eed9a9` 的原生 run
[`33611605670`](https://github.com/HappyQuQu/foliopath/actions/runs/33611605670) 证明游标优化已使
amd64/arm64 keyset P95 降至 `120,714us / 235,581us`，但扫描仍为 `190,135ms / 193,474ms`，因此
两个架构和 paired verifier 均正确失败关闭。上述进一步批处理优化必须在新 source SHA 上重跑，不能用
中间失败 artifact 充当通过证据。

后续 source `26977c7` 的原生 run
[`33613627441`](https://github.com/HappyQuQu/foliopath/actions/runs/33613627441) 在只完成状态预取和 job
批量接纳时，扫描进一步降至 amd64/arm64 `167,428ms / 156,717ms`，仍超过冻结预算，paired verifier
继续失败关闭。当前资产本体批量 UPSERT 是基于该实测继续收敛。

最终 source `5af4da0…` 的原生 run
[`33616238888`](https://github.com/HappyQuQu/foliopath/actions/runs/33616238888) 在 amd64/arm64 分别得到
扫描 `51,738ms / 45,441ms`、keyset P95 `182,299us / 247,994us`，两端均无预算违规；native jobs 与
paired verifier 全部通过，发布 jobs 均跳过。完整解释见
[S4 原生配对 baseline 证据](../evidence/int-001/int-s4-native-baseline-linux-amd64-arm64-2026-09-02.md)。

该结果关闭同 source SHA 的原生 baseline 缺口，但 paired summary 明确 `finalModelEvidence=false`；
最终模型、质量、final image、供应链与 owner 批准仍不得由本记录推断。
