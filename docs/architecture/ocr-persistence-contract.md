# OCR内部持久化合同（CR-2026-049 / ADR-0035）

这是内部Go repository合同，没有公开HTTP接口或后台自动调度。identity来自独立预配配置，非endpoint TOFU。Configure需要库revision，默认disabled/revision1/epoch1；配置不创建任务。有效identity的输入/输出hash必须与当前Go合同匹配。

任务仅missing与retry_failed，显式请求键（8～128 ASCII opaque字符）绑定库/epoch/identity/mode；同键同请求返回原任务（含终态），不同请求冲突。每库一个活动任务；不静默coalesce。候选按assetID升序，冻结上界，只接受当前next candidate；失败、ready(含空数组)分别计数，均不循环。源变动导致旧提交冲突且不推进；重新取候选按最新源处理。源指纹沿用catalog size+mtime v1而非内容哈希，不宣称检测同size/mtime的替换。

claim/recovery/commit/cancel/finish所有状态转换只由SQLite原子实现。lease仅running可刷新且不得复活过期者；queued取消立即terminal，running变cancelling，提交立即禁用。过期running小于3次可requeue；第三次failed/interrupted，cancelling过期cancelled。任务completed表示已记录尝试，不表示每张有文本或每张成功。finish succeeded必须不存在剩余candidate；固定服务错误终止不制造asset failed；库变offline时仍可用有效claim将任务终止为source_unavailable，保留已有结果，取消/换代优先。心跳只核对所有权、不绑定变化的checkpoint；成功提交/完成需核对进度，取消确认不覆盖已提交计数。

结果读取只返回enabled、非clearing、ready库、当前epoch所选identity以及当前源匹配结果。历史代次保留直到显式clear/library删除；失败不可携带lines/dimensions，ready必须有效32～2048尺寸及规范化行。准备图尺寸用于框解释，未映射原件像素。

清理用expected revision开启隔离（disabled+clearing+epoch++），重复旧revision冲突；Drain需对应新epoch，最多500条结果/事务、可重启续清。无剩余即clearing=false/revision++，仍disabled。对端运行者的lease不被清理复活，后续ack或过期回收终止。逻辑删除不擦除已有备份/WAL。现无公开读取/搜索权限入口；接入必须复用认证管理员/媒体权限边界，禁止日志正文。


## CR-2026-050 / ADR-0036运行扩展

现已有可选后台与管理API组合，api/openapi.yaml权威；未配置ocr.identity则不注册。源读取与复核经ContentService/files，采用原生OCR输入与共享client；任务池/heartbeat/计划handoff归原jobs.WorkerPool。busy以持久next_attempt延后5秒并退还当前claim attempt，模型/身份/输出故障固定终止，无标签式临时恢复重发。扫描过渡以单资产60秒预算等待，保留同claim/结果、不重复分析，heartbeat在scanning保持lease但仍核对epoch/取消。offline/error固定终止且不污染坏源。明确preprocessor坏源才写asset failure，producer拒绝准备PNG是job错误。

计划handoff仅在信号cause且processor/heartbeat已退出的边界调用，精确owner/fence/lease与最新存储checkpoint审计；取消优先，崩溃不退attempt。管理不暴露正文/原路径；清理靠clearing状态有界轮询，配置和开关不自动创建missing。源已删除/换源终止后需扫描再显式missing，不偷偷清catalog。未来全文索引/融合搜索、UI与历史任务保留仍待后续Gate。
