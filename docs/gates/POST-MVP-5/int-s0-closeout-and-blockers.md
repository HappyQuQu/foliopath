# INT-S0 收口与阻塞清单

- 日期：2026-08-27
- 状态：**本地技术探索收口 / INT-S0 对 revision 1 A+B 为 Go**
- 适用范围：`FTR-INT-001`、`INT-001～023`
- Gate：[INT-S0 Architecture Ready](int-s0-architecture-ready.md)
- 证据：[INT-001 evidence](../../evidence/int-001/README.md)

## 结论

现有隔离 spike 已足够回答“本地 AI 方案是否存在可行技术路径”：**存在，但只能保留
SigLIP 1 float16-internal + SQLite exact 的资源优先候选，图片语义搜索最适合作为首个切片；
标签、视频和人脸必须各自通过独立质量/隐私/许可 Gate，不能一起承诺。**

从本记录起冻结 S0 本地测试面。不再通过增加合成数据规模、查询排列、重复轮次或相似故障注入来
推动 Gate。已有测试保留为审计证据，不删除，也不等同生产验收。

产品用户在 2026-08-27 对推荐方案回复“继续”，以下五项决定据此全部接受。外部条件归对应
Backend/Release Gate，不再阻塞范围冻结，也不能在发布时跳过。

## 产品用户决定（已接受）

| ID | 必须决定 | 已接受默认 | 历史上不决定的结果 |
| --- | --- | --- | --- |
| `DEC-INT-001` | revision 1 纳入哪些切片 | 先冻结 A 模型基础 + B 图片语义搜索；C 标签建议、D 视频、E 人脸保持可删除后续切片 | scope 不能冻结，S1 不开始 |
| `DEC-INT-002` | 人物库作用域 | 人物为实例级；face observation、匿名组和资产关系仍按库隔离；允许用户显式跨库合并到同一人物 | E 不进入冻结 scope |
| `DEC-INT-003` | AI 开关与清除 | semantic/face 按库独立开关；关闭即停任务和查询；默认仅清除可重建派生数据，人物名与人工关系必须二次确认后才能清除 | 设置、删除和隐私合同无法冻结 |
| `DEC-INT-004` | 模型获取承诺 | `/models:ro` 离线目录为必选基线；项目签名在线源仅在有运营 owner 后启用；不承诺国内镜像 | 只提供离线安装，不显示无后端的下载入口 |
| `DEC-INT-005` | 预算与停损 | A+B 最多 16 单人工程周；C/D/E 各自评审，POST-MVP-5 总上限 32 周；任一切片超预算或 Gate 失败即删除该切片 | 继续保持提案，不排期 |

以上决定已写入 [POST-MVP-5 scope revision 1](../../releases/POST-MVP-5-scope.md)。

## 外部阻塞，不再用合成测试代替

| ID | 条件 | 责任角色 | 最迟 Gate | 缺失时的处理 |
| --- | --- | --- | --- | --- |
| `EXT-INT-001` | 合法、代表性的图片质量集 | 产品/QA/ML | Slice B Backend Evidence Ready | 不发布图片语义搜索 |
| `EXT-INT-002` | 合法视频集和代表帧检索标注 | 产品/QA/ML | Slice D Backend Evidence Ready | 删除或延期 D |
| `EXT-INT-003` | 经书面授权和隐私评审的人脸 ground truth | privacy/legal/QA | Slice E 开工前 | 删除 E；不允许用公开图片许可替代生物特征授权 |
| `EXT-INT-004` | 原生 Linux/amd64 runner | release/infrastructure | 对应切片 Backend/Release Gate | 不声明双架构完成，不发布对应切片 |
| `EXT-INT-005` | runtime/权重的再分发、SBOM、漏洞与 notices 签署 | security/compliance/release | Slice A Release Gate | 不分发该模型；SFace 继续 hold |
| `EXT-INT-006` | 在线模型源、签名 key/checkpoint、轮换/撤回和可用性 owner | release/operations/security | 在线下载能力开工前 | 只保留 `/models:ro` |

## 移出 S0 的验证

下列事项仍然必须完成，但不再作为“继续做 S0 合成测试”的理由：

- 真实 embedding Recall、1,000 图和 100 视频质量：进入 B/D 的 Backend Evidence Gate。
- 人脸 detector/ROC/聚类 precision、清除和隐私演练：进入 E 的开工准入和 Backend Evidence Gate。
- 生产 adapter、migration、API、事务、日志脱敏：进入 S1/S2；S1 Contract Ready 前不得提前实现。
- 最终双架构镜像、完整进程 100k 容量、SBOM/provenance/notices：进入对应 Release Gate。
- 既有 catalog 搜索 query-plan 修复：使用独立 maintenance Change Record，不属于 AI Gate。

## 停损规则

1. `DEC-INT-001～005` 已确认；S1 期间不新增 AI spike/benchmark，除非合同或候选变化会直接改变决定。
2. 在真实授权数据或原生 amd64 runner 未到位时，只登记阻塞，不用更大的合成集替代。
3. 同一风险已有一次通过证据和一次失败关闭证据后，除非候选、合同或环境发生变化，不重复测试。
4. 新测试必须能直接改变一个明确决策；只“增加信心”但不改变选择的测试不进入清单。
5. 当前工作树中的 benchmark-only SQL 不进入生产 import graph；是否修复现有搜索由独立 Gate 决定。

## 下一阶段

进入 A+B 的 S1 合同设计：更新产品/流程/架构/数据/安全/部署/测试影响，冻结 OpenAPI 与事务合同。
仍不得开始生产后端或 UI，也不再滚动扩张 S0 证据目录。
