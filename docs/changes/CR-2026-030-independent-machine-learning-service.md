# CR-2026-030：从模型商店转向独立 AI 服务

2026-09-16 最终 P2：A04～A08、D02 全部功能验收通过；FR-INT-009 / NFR-INT-002、004，POST-MVP-5 S2，Owner face/SQLite/app/API/web。补齐非空人工闭环、精确重绑、清理依赖顺序、旧可见代次和正式 UI；不变更 ADR-0019/0020 边界，不自动开放整组归类。[固定镜像证据及失败复验记录](../evidence/int-001/face-business-p2-2026-09-16.md)。总清单 9/34 关闭，P3/P4/P5/P6 未完成；以下同日条目为历史增量。

2026-09-16 P2/A05：已补齐正式人物/匿名组读写服务的 app 组合，复用既有合同与事务，不扩展范围；[本地验证记录](../evidence/int-001/face-people-composition-2026-09-16.md)。非空操作与完整 UI 仍待验收，P2 未完成。

2026-09-16 P1 复查更正与修复：先前验收遗漏激活本地忙和推理服务端忙，曾撤回完成结论。现已补齐可取消预算等待、身份约束的限长 busy 解析、同操作恢复成功验证；本地最终镜像探针通过后重新关闭 A02/A03。Owner app/remoteml/face，FR-INT-009 / NFR-INT-002、004 / POST-MVP-5 S2，无架构边界变更。[证据](../evidence/int-001/face-busy-repair-2026-09-16.md)。

2026-09-16 P1 收口：A01～A03 已完成；Owner face/app/SQLite/API，FR-INT-009 / NFR-INT-002、004，POST-MVP-5 S2。最终本地 Docker 正式任务和强杀恢复通过；新增专属恢复、可用性事务校验及共享预算接入，未改变部署单元或公共合同。P2/质量/发布未完成。[证据](../evidence/int-001/face-worker-p1-2026-09-16.md)。以下同日条目为较早增量记录。

2026-09-16 A03 增量：FR-INT-009 / NFR-INT-004，POST-MVP-5 S2，Owner face/SQLite。代次绑定处理器和内部读取合同已实现、Docker 回归通过；未接主进程，不宣称正式分析可用。[证据](../evidence/int-001/face-generation-binding-2026-09-16.md)。

2026-09-16 A02：FR-INT-009 / NFR-INT-002、004，POST-MVP-5 S2，Owner app/aimodel/face。主进程可选人脸配置与启动激活接通；本地实际离线/激活/停服/恢复/重启通过，测试 profile 不等于质量结论。[证据](../evidence/int-001/face-startup-2026-09-16.md)。

2026-09-16 用户授权按 [解决方案](../features/intelligent-media-discovery-completion-plan.md) 继续完成功能。FR-INT-009 / NFR-INT-004，POST-MVP-5 S2，Owner face/SQLite/app：A01 完整身份与原子提交已验证，A02 app 适配实现但未接启动。[证据](../evidence/int-001/face-deployment-commit-2026-09-16.md)。

2026-09-16 人脸激活调度：FR-INT-009 / NFR-INT-002、004，POST-MVP-5 S2，Owner aimodel/SQLite。
复用原调度，独立服务人脸运行时及迁移 40 已实现，文件提交失败关闭。[合同与验证范围](../evidence/int-001/face-deployment-worker-2026-09-16.md)；尚未由 app 装配，不代表真实激活/入库。

2026-09-15 人脸图片运行时：FR-INT-009 / NFR-INT-002、004、005，POST-MVP-5 S2，Owner imagevips/remoteml/face。
实现安全解码重编码与既有质量规则复用；Docker 三包竞态及四格式真实零人脸通过。[证据及代次/主进程剩余项](../evidence/int-001/face-runtime-adapter-2026-09-15.md)。

2026-09-15 Go 人脸传输：FR-INT-009 / NFR-INT-002、004、005，POST-MVP-5 S2，Owner remoteml/face/app。
独立身份 + 共享并发 + 有界规范化结果已实现；Docker 13 项竞态与真实离线零人脸调用通过。[证据与尚未组合范围](../evidence/int-001/face-go-transport-2026-09-15.md)。

2026-09-15 人脸私有接口：FR-INT-009 / NFR-INT-002、004、005，POST-MVP-5 S2 接入子片，Owner inference/face/app。
先行固定合同、可选缓存构建、独立身份和结果规范化已实现；Docker 13 项及真实零人脸通过。[合同、证据与尚未接入范围](../evidence/int-001/face-service-contract-2026-09-15.md)。

2026-09-15 跨库审核清理隔离：FR-INT-004 / FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/SQLite/curation。
混合账本失效、B 库审核/独立账本/正式标签保留通过；[证据与正式重放待验收](../evidence/int-001/tag-review-cross-library-2026-09-15.md)。

2026-09-15 新词条编码取消恢复：FR-INT-004 / FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner app/semantic/remoteml。
实际服务暂停下取消及重试通过，不推导为 ONNX 内核中断；[证据](../evidence/int-001/tag-text-request-cancellation-2026-09-15.md)。

2026-09-15 标签非空提交交错：FR-INT-004 / FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/SQLite。
提交前拒写、提交后保留、停机恢复四场景测试；[证据及真实推理待验收范围](../evidence/int-001/tag-job-processing-cancellation-2026-09-15.md)。

2026-09-15 标签生成实际等待取消：FR-INT-004 / FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner app/semantic/resourcecontrol。
新镜像 1 候选任务取消及重试通过，原 curation 保留；[证据及剩余矩阵](../evidence/int-001/tag-job-runtime-cancellation-2026-09-15.md)。

2026-09-15 标签生成取消：FR-INT-004 / FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/SQLite。
已接受取消有界收尾并优先于后续终态尝试，停机不误取消；[证据与剩余范围](../evidence/int-001/tag-job-cancellation-2026-09-15.md)。

2026-09-15 非空标签审核闭环：FR-INT-004 / FR-INT-016，POST-MVP-5，Owner app/semantic/curation。
真实模型及正式 API 接受、重放、审核清理保留正式标签通过；[证据](../evidence/int-001/tag-review-chain-2026-09-15.md)，阈值未改，质量与取消矩阵待完成。

2026-09-15 标签生成组合：FR-INT-004 / FR-INT-016，POST-MVP-5，Owner app/semantic。
同代次文本编码、既有图片向量、单 worker/共享预算接入；候选阈值 0.65（cosine 0.30）仅过滤待审核建议，尚未校准。
[合同与首次实际任务证据](../evidence/int-001/tag-generation-wiring-2026-09-15.md)，非空审核闭环、取消和质量矩阵仍待完成。

2026-09-15 上游缺口校正：FR-INT-004 / FR-INT-016，POST-MVP-5，Owner app/semantic。
标签生成任务尚未组合、正式 POST 404；优先补生成接入后再验收非空审核清理。[核查证据](../evidence/int-001/tag-generation-gap-2026-09-15.md)。

2026-09-15 标签审核清理实际镜像：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/SQLite/resourcecontrol/app。
新镜像取消及重试通过；[证据及非空实际验收边界](../evidence/int-001/tag-clear-runtime-2026-09-15.md)。

2026-09-15 标签审核清理批次取消：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/SQLite。
非空批次后已接受取消优先收尾，正式标签保留；[证据及剩余范围](../evidence/int-001/tag-clear-batch-2026-09-15.md)。

2026-09-15 标签审核清理准入取消：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/SQLite。
取消有界收尾、无用户请求的停机保持恢复；[证据与待完成矩阵](../evidence/int-001/tag-clear-admission-2026-09-15.md)。

2026-09-15 取消修复运行镜像回归：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/scanner/resourcecontrol/app。
包含图片/清理终态修复的新镜像四类取消恢复通过；[证据与剩余范围](../evidence/int-001/cancellation-runtime-refresh-2026-09-15.md)。

2026-09-15 清理批次取消：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/SQLite。
修复部分及最终删除批次后取消被覆盖/冲突，保留真实删除进度；[证据与剩余范围](../evidence/int-001/clear-batch-cancellation-2026-09-15.md)。

2026-09-15 图片处理器终态组合：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/SQLite。
分页提交后与编码失败后取消验证通过，保留已提交向量；[证据及清理待修项](../evidence/int-001/image-processor-terminal-2026-09-15.md)。

2026-09-15 图片终态取消优先：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/SQLite。
已接受取消优先于后续成功/失败收尾，保留领取版本校验和进度；[证据及待验收范围](../evidence/int-001/image-terminal-cancellation-2026-09-15.md)。

2026-09-15 扫描/清理实际混合验收：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner scanner/semantic/resourcecontrol/app。
新镜像完成取消及恢复测试，后台预算保持原设定；[证据及剩余范围](../evidence/int-001/scan-clear-admission-runtime-2026-09-15.md)。

2026-09-15 扫描准入取消修复：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner scanner。
入口取消/超时复用已有有界 abort，资源控制仅通知；[证据及未覆盖边界](../evidence/int-001/scan-admission-cancellation-2026-09-15.md)。

2026-09-15 清理准入取消修复：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/SQLite。
取消收尾要求持久化取消状态，停机保留恢复；[证据](../evidence/int-001/clear-admission-cancellation-2026-09-15.md)。

2026-09-15 图片等待取消实际验收：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic/resourcecontrol/app。
双向混合任务脚本及新镜像验证，保留既有预算；[证据](../evidence/int-001/image-admission-runtime-2026-09-15.md)。

2026-09-15 图片准入取消修复：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner semantic。
图片能力复用已有通知与原 CAS，停机保留恢复；[证据](../evidence/int-001/image-admission-cancellation-2026-09-15.md)。

2026-09-15 真实混合任务验收：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner resourcecontrol/semantic/app。
新镜像通过图片运行中视频等待取消及恢复，原设置复原；[证据](../evidence/int-001/mixed-admission-runtime-2026-09-15.md)。

2026-09-15 视频准入取消收尾：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner resourcecontrol/semantic。
资源包装器通知能力层，视频复用原终态 CAS 与有界清理；[证据及未完成范围](../evidence/int-001/video-admission-cancellation-2026-09-15.md)。

2026-09-15 后台资源准入修复：FR-INT-016 / NFR-INT-007，POST-MVP-5，Owner resourcecontrol。
拒绝已取消请求取得空闲名额，保持共享预算/释放策略；[回归与剩余项](../evidence/int-001/background-admission-2026-09-15.md)。

2026-09-15 旧任务记录实际验收：FR-INT-016，POST-MVP-5，Owner 前端 ai。
新运行镜像的真实旧记录移除、刷新及中英提示通过；[证据](../evidence/int-001/missing-task-records-runtime-2026-09-15.md)。

2026-09-15 旧任务记录修复：FR-INT-016，POST-MVP-5，Owner 前端 ai。
消费既有 404 合同，仅移除浏览器记录，不删除/取消服务端任务；[回归证据](../evidence/int-001/missing-task-records-2026-09-15.md)。

2026-09-15 浏览器多库选择验收：FR-INT-004/016，POST-MVP-5，Owner 前端 ai。
差异进度、切换刷新和补齐目标通过；旧任务残留待修，见[记录](../evidence/int-001/multilib-browser-2026-09-15.md)。

2026-09-15 多库视频隔离验收：FR-INT-004/016，POST-MVP-5，Owner semantic/SQLite/app。
正式 HTTP/实际推理验证单 worker 多库排队及取消重试隔离；[证据](../evidence/int-001/multilib-cancellation-2026-09-15.md)。无生产调度变更。

2026-09-15 视频取消实际镜像验收：FR-INT-004/016，POST-MVP-5，Owner semantic/SQLite/前端 ai。
本地 Docker 两种运行中取消及取消后补齐通过；[证据及剩余范围](../evidence/int-001/video-cancellation-runtime-2026-09-15.md)。

2026-09-15 视频取消修复：FR-INT-004/016，POST-MVP-5 接入验证阶段，Owner semantic/SQLite。
沿用现有任务合同和终态事务，修复取消滞留及被失败覆盖；不改 API 或媒体边界。
[确定性交错及跨库隔离回归](../evidence/int-001/video-cancellation-2026-09-15.md)。

2026-09-15 视频操作子片：FR-INT-004/016，POST-MVP-5，Owner 前端 ai。
消费已接受并集成验证的视频 jobs POST 合同，只补 missing/all 操作入口、重建确认和任务追踪。
复用既有 CSRF、幂等键、任务记录/取消/错误提示，不新增任务调度或部署单元；原始媒体保持只读。

2026-09-15 图片未记录边界：同属 FR-INT-005/015/016 / POST-MVP-5，Owner semantic/API/前端 ai。
GET/PATCH 设置响应增加向后兼容的 coverageRecorded，区分未记录与已记录空库；旧服务可省略，新服务总是返回。
未记录时 complete=false，UI 显示未知；不改数值字段类型或写入协议，不影响搜索返回的覆盖率合同。

2026-09-15 图片离线进度修复：FR-INT-005/015/016，POST-MVP-5，Owner SQLite/semantic。
既有 GET 设置继续返回历史索引进度，模型不可用不再清零；任务所用 activeGenerationId 的可用性门禁保持不变。
不新增持久化或功能开关，不放开离线新任务；以本地 Docker 仓储和真实离线恢复探针验证。

2026-09-15 视频进度子片：FR-INT-006/007/016，POST-MVP-5，Owner semantic/SQLite/API/前端 ai。
为既有视频任务提供只读库状态合同 `GET /libraries/{libraryId}/ai/video-semantic`（OpenAPI 权威）。
不调用推理、不新建任务，不把缺失进度当作零候选完成；离线/关闭状态保留已记录汇总。
覆盖率绑定活动代次，但不是模型健康诊断。先完成后端及本地 Docker 回归，再接生成客户端与设置页；
不增加服务或表，不改变历史任务、媒体、发行 Gate 或人物能力。

2026-09-15 共享激活 worker 与 app 适配：POST-MVP-5 / FR-INT-017/020、NFR-INT-007/009/010。
aimodel 内部服务准入复用原幂等/队列/失败处理；单一 worker 可在组合期绑定服务 runtime。
app 适配重新核对共享 HTTP 身份，提交交给 semantic 的空间事务；不打开模型文件。
按代次合同实现，尚未在 run.go 创建实例，未开放部署环境变量或浏览器入口。

2026-09-15 服务激活事务：POST-MVP-5 / FR-INT-017/020、NFR-INT-007/009/010，
Owner semantic 合同 / SQLite 既有激活事务。允许按代次合同扩展原事务并原子绑定新服务空间；
不增加第二个激活 worker/状态机，不开放生产配置或浏览器入口。失败保留旧活动代次与向量。

2026-09-15 服务模型记录子片：POST-MVP-5 / FR-INT-017/020、NFR-INT-007/009/010，
Owner aimodel/SQLite；代次合同接受 service 类型和内部 RegisterDeployment，migration 39
保留旧模型与外键，普通安装/激活不允许 service。OpenAPI 的模型响应 storageMode 扩展
service，但安装请求仍只接受 managed/direct；生成客户端同步。主应用调用/激活未接入。

2026-09-15 视频流子片：POST-MVP-5 / FR-INT-019/020、NFR-INT-007/009/010，Owner semantic。
按代次合同复用现有视频完整计划与流接口，保留旧构造器兼容；仓储、任务状态、API 不变。
禁止部分帧提交，验证失败/退役/取消和流关闭。主应用服务配置/激活仍待后续合同与实现。

2026-09-15 用户选择 ADR-0020 方案 A，接受开发专用薄适配镜像。新增
[服务合同](../architecture/independent-ml-service-contract.md)，inference 拥有快照/身份/协议，
app 后续消费独立预期身份；允许候选镜像与 Go 校验客户端实现，不签署生产或发行完成。

2026-09-15 允许隔离的真实推理/持久化组合 probe：复用既有流式任务、BoundEncoder、
imagevips 与 HTTP 适配，在本地 Docker 用合成图片和临时 SQLite 验证提交、数据库重开、
文本检索。Owner sqlite 测试适配；POST-MVP-5 接入验证阶段，FR-INT-017/019/020、
NFR-INT-007/009/010。仅测试构建标签启用；使用明确的合成身份，不认证部署或生产激活。

2026-09-15 轻量修复记录：POST-MVP-5 独立服务接入前兼容 Gate，FR-INT-017/019/020、
NFR-INT-007/009/010；Owner semantic（唯一策略）/app（两处调用）。旧工厂原先只判断
active/768 维，现补充拒绝非空空间绑定，防止新代次落入旧运行时。
无 API、数据库或部署配置变更；[合同](../architecture/independent-ml-generation-contract.md)与
[Docker 回归证据](../evidence/int-001/independent-ml-legacy-boundary-2026-09-15.md)。

共同准入子片按代次合同实现 semantic BoundEncoder，只组合现有窄接口，未更换 app 或旧 worker。
文本 active / 图片 building-ready-active，推理前后复核指纹/维度/状态。证据见
[共同代次校验](../evidence/int-001/independent-ml-bound-encoder-2026-09-14.md)。

2026-09-14 接续开发：按代次兼容性合同的持久化子片新增 migration 38，只添加一次性空间绑定，
不自动绑定历史数据或激活模型。semantic 持有接口，SQLite 使用既有 write transaction；
空间指纹随 GenerationRuntime 读取。证据：`semantic_space_test.go`，相关存储/semantic/app 包测试。

用户于 2026-09-14 确认 Immich 式双服务模式，并授权复制 server222 已下载模型做本地测试。
范围 POST-MVP-5，FR-INT-017/019/020、NFR-INT-007/009/010；Owner 与合同要求见
[ADR-0019](../adr/0019-independent-machine-learning-service.md)。
冻结 MVP 不扩展；HF 上传继续暂停，不更改线上服务或原媒体。

交付顺序：只读盘点/复制/hash → 独立 AI 容器合成推理 → 内部 API/模型代次合同 →
主应用 adapter 与集成 → 简化 UI 与 Compose → 发行验证。每步单独记录证据。
已有 768 维 SigLIP / YuNet-AuraFace 合同不能被 CLIP / buffalo_l 静默替代。

后端传输子片采用[内部 HTTP 合同](../architecture/independent-ml-transport-contract.md)。
允许独立 Go adapter 与隔离集成测试；不授权直接接入旧代次或以 HTTP 成功替代模型身份准入。

代次兼容性子片接受[向量空间身份合同](../architecture/independent-ml-generation-contract.md)：
semantic 拥有纯描述验证、稳定指纹和代次匹配，合成测试覆盖每项身份变化及旧代次拒绝。
不读取模型文件，不证明服务身份，不自动激活或迁移数据。部署事实核验与生产接入继续待完成。

用户选定 UI 方案一后，允许 `spikes/int001-ui-settings` 的可丢弃交互原型（不连接后端），
复用正式组件与 tokens；生产 UI 切换仍等后端 Gate。范围/验证见该目录 README/design-qa。

2026-09-15 覆盖率修复（FR-INT-006/007/015/016，POST-MVP-5，Owner SQLite）：
视频移除后只有成功扫描可核对剩余有效计划；未知归属的失败/降级汇总退回待处理，
不猜测、不改写历史任务。见 video-removal-2026-09-15 证据和 OpenAPI 行为说明。

2026-09-15 启动组合子片：POST-MVP-5 / FR-INT-017/020 / NFR-INT-007/009/010，
Owner app；合同为 independent-ml-generation-contract.md，复用已验证服务模型、激活
worker/事务及 BoundEncoder。准许以部署 JSON 组合并在本地候选主进程验证离线启动、
自动激活和恢复；不改变发行 Gate 或承诺人脸/完整 UI 已完成。

2026-09-14 用户进一步要求“根据原型调整项目”：接受一个限定的现有合同 UI 重排子片。
Owner：前端 ai；引用既有 S2A/S2C/S3 状态、设置、任务与模型合同，仅调整信息架构与布局。
主入口采用方案一的能力/按库进度/部署详情；旧模型维护退至明确标注的兼容入口，旧 URL 保留。
不引入新服务 DTO、Compose 状态探测、模拟数据或模型代次。独立服务生产 Gate 仍未通过。
这是对已通过后端合同的消费者重排，不是提前宣告新后端或新服务 UI 可用。

安全图片入口子片采用[编码图片合同](../architecture/independent-ml-image-contract.md)，
允许 semantic 窄接口与 imagevips 流适配独立实现/合成测试。native 验证通过之前不标记完成，
不把此入口挂到旧 768 维代次；主应用与持久化仍待后续 Gate。
回归证据：`AISettingsPage.test.tsx`（ETag/CSRF、库隔离、错误关闭、旧维护安全确认）。
