# POST-MVP-5 原生容量回归收敛

- 日期：2026-09-02
- 状态：**Implemented and locally verified / native rerun pending**
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
fingerprint 改变时才清理旧 thumbnail/storyboard 状态；已有资产变更、只见代次变化与新资产的业务语义不变。
这消除了首代 10 万资产的 20 万次无效索引删除，同时保留 `secure_delete=ON` 和真实隐私删除证据要求。

## 验证

- `go test ./internal/catalog ./internal/store/sqlite ./internal/scanner`：通过；
- `make spike-capacity`（darwin/arm64，4 CPU，10k 目录/100k 资产）：通过，零预算违规；
  - scan `65,515ms <= 120,000ms`；
  - search keyset P95 `79,632us <= 250,000us`；
  - first/second page P95 `66,663us / 13,247us`；
  - peak Go heap `43,953,240 bytes`，DB+WAL `157,372,416 bytes`。

本机结果只证明修复候选满足本地门槛。`INT-403` 仍须同一 source SHA 的真实原生 amd64/arm64 rerun；
最终模型、质量、供应链与 owner 批准仍不得由本记录推断。
