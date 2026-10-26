# CR-2026-049：OCR源绑定结果与持久任务基础

- 日期：2026-10-26；目标：POST-MVP-5，内部backend storage子片；FR-OCR-003 / NFR-INT-007/009/010。
- owner：internal/ocr、internal/store/sqlite；ADR-0035。
- 范围：独立identity/epoch、源绑定结果、missing/retry_failed任务事务、lease/fence/recovery、禁用/清理隔离；公开API/worker/搜索/UI/6010启用不在此片。
- 合同：docs/architecture/ocr-persistence-contract.md 与 internal/ocr/persistence.go；迁移00045。
- 原数据保护：仅合成数据库，不操作原件/6010/server222；旧10源问题仍跳过。人工标签/删除抑制/人物/语义表不由OCR写入。
- 状态：内部存储子片开发与本地Linux Docker验收完成；未接runtime/API或部署原6010。

## 实现

- `internal/ocr/persistence.go`拥有内部类型/验证与Repository；`ocr_settings/jobs/results.go`在原SQLite写gate中实现唯一状态机，无额外运行调度。迁移45只增休眠OCR表，不变更ai_model_operations或旧能力身份。
- 每库默认关闭，CAS revision/epoch隔离。完整identity验证输入/输出hash后按指纹存储；换模型保留旧输出但仅当前源/模型可读，同身份重新启用保留进度。
- missing不循环坏源；retry_failed显式重试固定坏源。freeze上界与逐条keyset，32活动上限/库1任务。服务/IO类错误只终止job，结果/checkpoint同事务，回滚不留半条结果。
- claim/lease/epoch/source/checkpoint共同fence；心跳忽略可变进度，取消确认不覆盖已完成计数。收尾审查补齐offline时以有效claim终止source_unavailable（不要求库重新ready），设置变化/取消仍优先。
- clear先禁用隔离、每批≤500删除全部代次该库正文，可重开续清；其他库不变。读取隐藏非当前源/代次/禁用/offline/video结果；资产/库删除走复合外键。
- 同步ADR/AGENTS/需求/数据模型/安全/测试/追踪/进度；无UI/OpenAPI行为变化，明确保持OCR搜索不可用。fitness阻止主应用/API私自消费OCR以及OCR修改旧能力表。

## 验证与首轮记录

- `make fmt fmt-check arch-check generate-check lint`、`make test test-integration`、定向`go test -race`与`make test-e2e`最终执行结果见机器证据。
- 新16项SQLite顶层测试（含7子场景）与2项domain测试；另跑旧自动标签人工保护、Curation、人脸备份恢复、语义embedding回归。17表保护摘要覆盖非空人工中文标签、删除抑制、人物anchor/审核与语义向量，迁移和OCR写入/清理前后一致。
- 初轮Go全量命令沿用了已不存在的TMPDIR，未执行即失败；创建本片独立scratch后完整重跑通过，首轮日志保留。
- 初次Docker用build config digest作本地image ID不可解析，未创建容器；改用image inspect返回的immutable image/index ID。宽范围SQLite全量race矩阵过慢，主动中止而非宣称通过，改为全部新OCR/持久化加相关旧能力race；Go正常全量另已通过。初版镜像与最终修订不混用。
- 无浏览器/原生模型重跑（无UI/producer/预处理变化），无实际SIGKILL运行worker测试（尚未compose）；存储重开+过期lease模拟覆盖持久恢复。不是全OCR/真实图库/联合容量/双架构/发行证明。

## 后续Gate

安全源processor（internal/files + 当前指纹核对）、复用jobs.WorkerPool的runtime/计划handoff、管理API/权限和状态展示尚待实施；持久化任务元数据保留策略及全文索引/融合搜索/UI另行验收。未部署原6010，未连接server222，未提交推送。

## 最终实测（2026-10-26）

- 最终固定测试镜像：`sha256:0ee70929d5783ff32dddf573694d244b49aa69c0e992544423dc0deea8e05246`，Linux arm64 / Go1.26.5，network none / rootfs只读 / 2CPU / 2GiB RAM且无额外swap / 512MiB临时tmpfs；无任何原库/模型/应用data挂载。
- **33个顶层测试全部通过，无失败/跳过**：含全部16个新SQLite场景（另7子场景）、2个新domain以及既有OCR合同/其他相关能力回归，均带race。最终container exit0/OOMKilled=false；memory.events max/oom/oom_kill=0，peak146,604,032 B。这是存储测试资源观察，不是联合AI容量证明。
- 44→45迁移及OCR写入/clear前后17表摘要一致：`5e374136abc07e46b0e2606594fa9b1bd586b4afecb77d53c3f57c0efe46436e`。包含非空中文人工tag/抑制/person anchor/audit/语义vector；容器Linux域完整性与FK检查通过。不能借此宣称真实6010数据已验收。
- `make fmt fmt-check arch-check generate-check lint`、Go正常全量/集成、定向race、最终`make test-e2e`全部执行通过。最终应用smoke镜像`sha256:87c5dbf29666b439c036ec1503b8239b7a1d8b824735d383ae5d0b3d6084d204`，应用启动验收通过，没有启用OCR worker。
- 实际复现：`docker build -f tests/release/Dockerfile.ocr-persistence -t foliopath:ocr-persistence-20261026 .`；按[机器证据](../evidence/ocr/2026-10-26-ocr-persistence-local.json)的immutable镜像与矩阵运行。Dockerfile默认提供更宽SQLite全量race入口；本次最终验证使用明确的定向pattern，不将默认全量算作通过。
- 最终执行入口：`sh -c '/tests/ocr.test -test.v -test.timeout=120s && /tests/sqlite.test -test.v -test.timeout=600s -test.run="TestOCRPersistence|TestSemanticEmbeddingRepository|TestFaceStateSurvivesSQLiteBackupAndRestore|TestAutomaticTagsCommitAtomicProgressManualProtectionAndMissing|TestCuration"'`。旧28文件/producer、模型和输入策略本轮均未修改，不重做其历史验收。
- 三个确切合成容器及临时数据库已移除，镜像和忽略的日志保留；独立Go scratch已清理。未修改原6010、真实媒体/任务/人工整理，没有提交推送。
