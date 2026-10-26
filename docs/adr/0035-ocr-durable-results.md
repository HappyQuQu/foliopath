# ADR-0035：独立 OCR 持久结果与有界任务事务

- 状态：接受（2026-10-26），仅内部存储子片；POST-MVP-5 / CR-2026-049。
- 需求：FR-OCR-003，NFR-INT-007/009/010；owner：internal/ocr（合同）与 SQLite adapter（事务）。

## 背景

ADR-0032～0034已验证输入和推理，尚无主应用任务或索引。不能把一次HTTP成功当作当前源的索引，也不能借用语义代次或标签checkpoint。此片先建立可执行的持久合同；不增加公开API、worker composition、自动启动或搜索。

## 决定

- OCR identity完整JSON按指纹不可变存储，输入/输出hash须匹配当前Go合同。每库默认关闭，显式带revision配置；独立epoch在模型切换、禁用/启用、清理时递增，旧模型输出保留但不作为当前输出；同身份重新启用保留已完成结果，epoch仅隔离写入，不强制重算。
- 结果键为library + generation fingerprint + asset；绑定catalog原有v1源指纹，保存有界规范化行/框/分数及准备后尺寸。ready允许0行；failed只允许固定invalid_media/input_limit，绝不保存异常、路径或服务错误文本。
- missing只处理当前identity/源缺失，已记录坏源不循环重试；retry_failed是单独显式模式，既有成功不重算。每任务冻结upper asset ID，逐条keyset读取，最多32个活动任务/每库1个；新源或任务窗口后新增资产留待下一次missing。
- SQLite为状态机唯一owner。claim带单调fence/lease（最多3次崩溃尝试），提交必须同时核对库ready、enabled/epoch/generation、有效lease、checkpoint与下一候选、当前源指纹，在同一事务保存结果并推进checkpoint/计数。服务故障只可固定码终止任务，不伪装图片失败。取消立即阻止提交；终态不复活。恢复仅处理过期lease，每事务最多32项。
- 关闭、换代或清理立即取消/隔离旧claim；旧运行者仍占活动库槽到确认退出/lease过期，避免本地重复执行。计划停机handoff与runtime worker集成仍待下一片，不以普通context取消退还attempt。
- 清理先带revision关闭库、递增epoch、标记clearing，再逐事务最多500条删除所有代次该库结果。清理期间不可重新启用，旧claim不可写回；最后批次完成后才解除clearing。仅清OCR派生文本，不改人工标签、抑制、人物、语义。库/资产删除通过复合外键级联，不离线清空索引。
- 当前读取消disabled/clearing/offline/非当前源或代次的内容；原始OCR文本不进入日志/错误/公共接口。数据库备份可能含文本，逻辑清理不承诺清除旧WAL或备份。

## 替代与影响

不复用semantic/tag表或通用ai operation kind，以免无公开合同时偷偷接产品；不逐库建worker/队列。暂不构建全文索引；搜索和API/权限/任务展示需后续合同及测试。新增迁移不会启用功能或触发任务。runtime后续必须复用jobs.WorkerPool、现有HTTP/native准入与internal/files，不在此片制造另一套运行调度。

## 验证

需要合成SQLite迁移/重开、换源/换代、交叉库、失效claim/lease、崩溃恢复上限、取消/清理竞态、空结果、坏源不循环、原有整理保留及Linux Docker/race/完整性。实际结果见CR-2026-049；未通过前不宣称完成。

后续：ADR-0036 / CR-2026-050已接受并验证显式opt-in runtime/管理组合；本ADR保留原存储子片的历史边界，搜索/UI仍无授权。
