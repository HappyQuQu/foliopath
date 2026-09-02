# INT-S2B：标签建议与视频语义 Backend Evidence Ready

- 日期：2026-08-29
- 目标版本：[POST-MVP-5 revision 2](../../releases/POST-MVP-5-scope-r2.md)
- 范围：C 受控标签建议 + D 完整故事板视频语义搜索
- 当前判断：**Backend Ready / Release No-Go**
- Contract input：[INT-S1R2 Contract Ready](int-s1r2-contract-ready.md)（Go）

## 已授权边界

允许按 `INT-221～228` backend-first 实现 capability、append-only migration、SQLite adapter、HTTP handler、
production composition 与集成测试。仍不授权 S3 消费者 UI，也不允许为了模拟可用而接入未审核模型、自由
文本标签、第二套 FFmpeg 抽帧或部分故事板搜索。

## Backend Ready 必需证据

- C：受控词表、Top-5、generation/vocabulary/source 失效、100 项 review、curation 原子接受、review clear；
- D：只消费完整 10/4 帧 plan、max/best-frame 稳定 cursor、partial/degraded/fallback、取消与缓存重建；
- governed quality scorer/输入 validator、逐项 tag precision/recall 与 video Top-20 重算、阈值失败回退；
- 本地任务/查询/取消/restart/offline/clear 故障矩阵及原媒体 sentinel 不变。

最终合法 tag 数据签署、至少 100 个合法视频的 Top-20 success ≥80%、native linux/amd64+arm64、4 CPU/
4 GiB/100k 最终联合负载及 model/runtime SBOM/VEX/notices/provenance 由 S4 Release Gate 持有。任一项
缺失时 release 继续 No-Go：C 失败则删除 suggestion、D 失败则删除 video semantic。

## 2026-09-01 Backend Ready 签署

[CR-2026-022](../../changes/CR-2026-022-s2-backend-release-gate-separation.md) 将最终发行输入恢复到 S4。
`INT-221～228` 的 production repository、durable worker、HTTP、ranking/scorer、failure semantics 与
fail-closed composition 已完成并通过本地回归，因此 S2B 签署 **Backend Ready**。reviewed catalog 为空，
最终 governed tag/video 质量、native paired final image 和供应链未通过，故 **Release No-Go**。
同一 closeout 已成功执行完整仓库验证与 100k 强制容量；命令和结果集中记录于
[S2 最终完成审计](int-s2-final-blocker-audit-2026-09-01.md)。

## 2026-08-30 本地实现复审

[本地实现证据](int-s2b-local-implementation-evidence-2026-08-30.md)确认 `INT-221～226` 的 C/D
repository、队列、worker、HTTP 与失败矩阵已落地，并补齐零建议覆盖率、review clear、video production
job composition。该结果不包含 `INT-227` 合法质量、native amd64、最终联合负载或签署供应链，因此本 Gate
继续 **No-Go / Implementation Authorized**，S3 UI 仍不授权。

## 2026-08-31 非人脸复审

按产品用户“当前先跳过人脸测试”的执行顺序，本轮只推进 A～D 与共同发布输入，不把 S2C 暂缓解释为
删除范围。两库 10k/100k 的 benchmark-only order-first 查询候选已扩展并通过 11×2 页面、
0/1/10/100/>100 cardinality 等价矩阵；完整常规验证也再次通过。但 production 强制容量基线此前仍以
`searchKeysetP95Us=296212` 超预算失败，见
[搜索容量回归复审](../MVP-2026-07-23/s4-search-capacity-regression-2026-08-31.md)。
同档后续又补齐 22 个完整 hydrated 页面以及 FTS rebuild/integrity-check/数据库连接重开后的等价复核，
均通过；这只关闭候选的本地 hydration/recovery 子项，production 查询仍未获修改授权。
再加入中文两字、组合字符、sharp-s、标点、多词 AND 和带引号 exact fallback 后，17 个首页、11 个
第二页、28 个完整 hydrated 页面继续等价；完整重扫期间 10 次 hydrated 对照和扫描发布后复核也通过，
最新最慢 candidate ID 页 67,471 µs。真实选择性、旧 cursor 变更复跑、production owner 和 native
双架构仍未关闭。

Debian trixie `libc6` 仍无新 revision，合法 tag/video 质量集、native amd64、最终模型/runtime 供应链
签署也未到位。因此 `INT-227` 不勾选；`INT-228` 只按“复审已完成、判断为 No-Go”勾选，本 Gate 继续
**No-Go / Implementation Authorized**。

同日新增的手动
[`Intelligent media native evidence`](../../../.github/workflows/intelligent-media-native.yml) 入口只建立了
可审计的原生 amd64/arm64 执行路径；尚无当前 source SHA 的实际双架构成功 artifact，也未包含合法 tag/
video 质量签署，不能解除 `INT-227` 或把 Gate 转为 Go。任一容量或仓库检查失败都会保持 job/Gate 红灯。

paired verifier 已进一步拒绝单架构、跨 commit/run/attempt、QEMU、错误 runner identity 和任一步骤失败，
但尚未取得实际 run，也不验证合法 tag/video 质量集；因此仍不改变 `INT-227` 或 Gate 判断。

`make verify-intelligent-media-quality` 已把未来 `INT-227` 结果绑定到 governed manifest/model hash、三方批准
引用、逐 tag counts 和实际 Top-20 result IDs，并对 100-video 内容覆盖及冻结 80% 门槛失败关闭。当前没有
真实输入或签署，工具自身不算质量证据，本 Gate 继续 No-Go。

`make verify-intelligent-media-supply-chain` 还为最终 catalog/model、native 双架构 complete SBOM、签名
provenance、notices、漏洞/VEX 与 security/compliance/release/inference 批准建立了真实文件哈希校验入口。
当前没有最终产物或签署，故同样不改变 `INT-227` 或本 Gate 判断。

最终复审还须由 `make verify-intelligent-media-s2-evidence` 把质量、strict native 与供应链 summary 绑定到
同一 source commit、model package 和逐架构 final image digest。当前没有三份真实 summary，聚合入口的
单元测试不构成外部证据，本 Gate 继续 No-Go。
