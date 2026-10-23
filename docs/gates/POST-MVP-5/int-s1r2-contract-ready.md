# INT-S1R2：C+D+E Contract Ready

- 日期：2026-08-29
- 目标版本：[POST-MVP-5 revision 2](../../releases/POST-MVP-5-scope-r2.md)
- 范围：C 受控标签建议 + D 视频代表帧语义搜索 + E 匿名人脸聚类/人物库
- 当前判断：**Go / Contract Ready**

## 当前授权

Revision 2 已冻结 C/D/E 的产品范围和 32 工程周 scope-budget exception，但没有跳过 backend-first
交付顺序。本 Gate 只授权 `INT-114～120` 文档、OpenAPI/data 合同设计和必要隔离 spike；在转为 Go 前：

- 不创建 C/D/E production migration、handler、worker、app composition 或消费者 UI；
- 不创建 `internal/face` production package；
- 不把现有 synthetic face state-machine、10 图 pilot 或故事板能力解释为真实质量/隐私证据；
- 不接通空模型、mock、隐藏 route 或用户自行提供未审核权重的入口。

## Contract Ready 清单

| 任务 | 权威合同 | 当前状态 |
| --- | --- | --- |
| `INT-114` | 标签词表、suggestion/review、人工标签 owner、失效/清除 | Accepted |
| `INT-115` | 视频故事板输入、聚合/命中、部分失败、质量 Gate | Accepted |
| `INT-116` | face/person/manual constraint 状态机、默认关闭、禁止身份推断 | Accepted |
| `INT-117` | SQLite/data、事务、cascade、retention、backup/restore | Accepted |
| `INT-118` | OpenAPI、opaque ID、cursor/ETag/idempotency、错误与批量上限 | Accepted；lint/contract/generate-check 通过 |
| `INT-119` | security/privacy/deployment/testing、合法数据、双架构、容量、供应链 | Accepted as Gate contract；外部 evidence 仍由 S2 持有 |
| `INT-120` | traceability、owner、fallback、fitness check 与 Gate 复审 | Accepted |

## 外部准入不会在 S1 伪造完成

合同可以在没有真实数据时冻结，但 S2 Backend Evidence 必须继续持有：

- C 的受控词表与代表性 suggestion precision/人工复核证据；
- D 的至少 100 个合法代表性视频及 4/10 帧聚合质量、双架构联合负载；
- E 的隐私 owner 签署、合法真实 face ground truth、core precision ≥99.5%、可商业分发模型/runtime；
- 全范围 native Linux/amd64 + arm64、4 CPU/4 GiB/100k、SBOM/VEX/notices/provenance。

## 2026-08-29 复审结论

`INT-114～119` 权威源已接受；OpenAPI lint、contract check、client generation/check 与 architecture check
通过；traceability、R-024～030 owner/fallback 及 production composition fitness 已同步。因此本 Gate 为
Go，授权 S2B production backend-first 工作。

Gate 转换不完成任何 S2B/S2C 任务，也不授权 S3 UI。S2C 仍受
[人脸隐私与模型准入](int-s2c-privacy-ready.md)独立 No-Go 阻断；在该 Gate 通过前不得创建 production
`internal/face`、migration、handler、worker 或 route composition。
