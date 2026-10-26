# ADR-0036：OCR安全源处理、任务运行与显式管理

- 接受：2026-10-26；POST-MVP-5 / CR-2026-050；FR-OCR-001～004、NFR-INT-007/009/010。
- owner：ocr拥有处理/worker适配/管理用例；app连接ContentService、imagevips、共享remoteml；SQLite独占状态事务；api只处理HTTP。

## 决定

ADR-0035存储已验证。接受在既有可选AI部署JSON下新增独立ocr.identity，校验完整指纹与当前输入/输出hash。不配置则没有OCR路由/worker；配置不启用任何库、不自动提交missing。库开关和任务必须由已认证管理员明确操作，管理API无OCR正文/路径、无浏览器模型身份参数。此片仍无搜索/UI或原6010部署。

处理器只通过media.ContentService（最终internal/files openat2）打开已登记图片，核对库/源和格式，预处理并关闭源；推理后再次安全打开/核对。在首次失败结果提交前也重新核对源，IO/close/原生/未知服务错误不能记坏源。只将明确invalid_media/input_limit记源失败；准备输入合同/尺寸及独立模型均验证，服务只接准备PNG。

OCR使用jobs.WorkerPool一个processor、既有resourcecontrol.LimitBackground、同一HTTP/native准入。busy仅表示未进入执行的背压，可带持久next_attempt延后5秒并退还本次claim的attempt；不套用标签的临时断连恢复窗口。模型/身份/输出/超时错误终止本operation，已提交进度保留，无自动重发。新请求missing才补未完成，retry_failed需明确确认。

ADR-0027的计划停机规则扩展到OCR：只由原WorkerPool在processor和heartbeat退出后调用handoff，原SQLite事务重检精确claim/fence/有效lease，记录OCR独立有界审计，退还当前attempt并保留最新checkpoint。取消优先；普通context取消/崩溃不退attempt，终态不复活。不扩展旧offline维护工具或旧analysis_job_handoffs约束。

清理由已持久clearing状态驱动有界轮询（每批最多500/轮最多32库、keyset公平），不增加推理队列。库删除/清理竞态失败关闭；默认不读取正文到HTTP。管理状态在单SQLite快照计数当前源/模型的ready/failed/pending/stale/empty，不把历史模型或任务completed说成成功文本覆盖。

## 合同与验证

api/openapi.yaml先定义6组库作用域管理路径（settings、jobs创建/读取/取消、clear），所有修改CSRF+ETag，提交用幂等键；不加入通用AI operation kind。task status只暴露固定错误码和尝试/已记录/失败计数。新迁移46只添加next_attempt和OCR handoff审计；默认无激活/任务。

合成单元/SQLite/管理HTTP权限与错误测试，原生Linux真实模型完整安全源→预处理→任务→持久结果、取消/计划停机/重开/明确坏源及保护摘要/完整性。实际证据记入CR-2026-050；未通过不可宣称全OCR、联合容量/质量或6010可用。

## 启动扫描竞态

启动时旧queued工作可能先于startup scan被claim。接受OCR单资产最多60秒的临时scanning/pending等待，复用原heartbeat核对lease/取消，每秒重新检查，不持有事务；同一预算覆盖读candidate和写结果，已经得到的有界结果不重复推理。offline/error立即固定source_unavailable终止；设置取消/换代仍立即fence。此等待由OCR worker一个owner实现，不改变扫描或人脸策略。
