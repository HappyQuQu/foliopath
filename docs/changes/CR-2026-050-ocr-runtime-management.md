# CR-2026-050：OCR后台处理与管理接口

- 日期2026-10-26；POST-MVP-5；FR-OCR-001～004 / NFR-INT-007/009/010；ADR-0036。
- 范围：安全源处理器、共享任务池与计划handoff、部署显式opt-in、库管理API及有界状态/清理；无搜索/UI/原6010部署。
- 合同：api/openapi.yaml；内部OCR持久合同与迁移46。
- 默认：无ocr配置不注册；有配置仍各库关闭，不自动创建任务。保留人工标签、抑制、人物、语义和原任务；仅合成Docker，禁止server222/原件写入。
- 本片开发与最终本地Docker验收完成；只交付候选backend，原6010未更新。

## 实现与边界

- 独立安全源Analyzer经ContentService/files开图并复核库/源/格式，imagevips使用原OCR输入合同；文件在推理前关闭，固定坏源也要复查来源。Runtime adapter把准备PNG的producer拒绝/输出预算归为程序/任务错误，不伪造原件失败。
- Worker由现有jobs.WorkerPool和resourcecontrol包装，独立业务处理不复制线程池或native门。每个资产的临时scan等待≤60秒，读candidate/写结果共用等待预算，保留已取得结果；heartbeat在scanning继续核对租约与取消。
- busy仅持久延后5秒并退还本次claim收费；普通服务/身份/输出错误固定终止。新missing保留旧成果，不循环坏源。取消在terminal/deferral竞态也会确认，计划handoff只在原owner退出processor/heartbeat后精确审计、保留最新checkpoint并退当前attempt；崩溃仍最多3次。
- 迁移46仅OCR next_attempt与有界handoff审计。没有OCR配置则不注册路由或worker；有配置也默认各库关闭，不自动提交。6个管理operation只给元数据，无正文/原路径/浏览器模型参数；GET/写入沿用现有session/CSRF/强ETag/中央限流，幂等绑定库epoch/模型/mode，retry和clear明确确认。
- 修复清理后的读取竞态：settings、coverage和latestJob由同一个SQLite只读快照返回；清理已提交后不会因后续分批drain的revision变化被误报409。单模型ready/failed/pending/stale/empty计数不混旧源/模型，recorded不等于成功图片数量。
- 旧OpenAPI字节可逆还原为原锁定SHA256（见机器记录），仅增加OCR路径/schema/tag和additive marker，再更新锁和精确operation清单；生成client重建，无产品UI改动。

## 最终验证

- 单测固定Linux arm64镜像：`sha256:083c9b29a45efc2fb248424b5ed83ae3bf0fc20dbee00bce2fbd84acf7299362`。52个顶层测试/race全部通过、无失败/跳过；含16既有OCR持久化、8个运行SQL场景和source/API/app/相关旧能力。2CPU/2GiB且无额外swap/512MiB tmpfs、network none/rootfs只读；exit0、OOMfalse、memory.events max/oom/oom_kill=0，peak359,854,080 B。
- 17表保护摘要迁移/写入/清理前后一致：`2bcd50d4d4b7d858a8cf6f30019241a372c8ad5bd5b9f68c85bd1c627b8f9ffe`，覆盖非空人工中文tag/抑制、人物anchor/audit和语义向量；完整性/FK检查通过。
- 真实模型最终probe：`sha256:fbdc6da5672b9d2308197304a3ce25e21bc3df8b09b0559b83efde542536bfb0`，AI未修改，仍`sha256:e995f5748c36ae8de9f3dd86633e4a2f60f3f6d6b7eaaba6616bbdf6ee73da21`。3个独立OS应用进程：记录2项后计划停机，原同job续补到5项；人工标签/revision、源hash/完整性保留。bilingual和EXIF两图各4行与固定fixture文字匹配，blank为ready0，corrupt/empty为真实invalid_media；重复missing为0项，确认clear后disabled/pending5。
- 管理HTTP由完整app+真实认证执行；未经配置则404、认证/CSRF/If-Match/确认词和跨库固定错误另有unit/race。真实probe仅单/library只读合成挂载、内部网络无端口、DB在同Linux tmpfs；没有macOS打开Docker DB或运行中复制。
- AI测试预算4GiB/2CPU（client2GiB/2CPU），实际peak2,185,785,344 B，max/oom/oom_kill=0；没有改6010资源配置，不证明联合4GiB容量。两份OCR权重测试后仍匹配固定SHA，模型cache只读，快照/测试容器/网络清理。
- `make fmt`、`make fmt-check arch-check generate-check lint`、`make test test-integration`、四包定向`go test -race`、`make test-e2e`通过。Node24.22.1/npm11.19.1执行`npm ci --prefix web`与`npm --prefix web run check:types`通过；安装器存在既有audit/install-script提示，未放开脚本或改依赖。无浏览器视觉测试（UI未改）。

## 首轮失败与修复

1. OpenAPI锁与operation清单按预期拒绝6个新增管理操作；反向还原验证旧合同字节不变后，按本CR添加兼容记录/更新锁及生成client，检查通过。
2. API/app单测Docker最初缺embed dist，构建失败；仅单测recipe添加明确占位fixture，原生完整app probe继续使用实际web构建，不混为UI验收。
3. 首次native测试在同Go进程重启libvips被原process-final保护拒绝。未放松保护；改为3个独立OS进程共享测试tmpfs目录，正确模拟应用重启。生产代码无此放宽。
4. 第二轮native已通过handoff/同任务补齐/真实文本持久化，但clear成功后分批drain使多次状态读取的revision变化，误报409。改为SQLite单快照状态，新增并发清理回归，最终重跑通过。
5. 最终额外核对持久化文字（按fixture测试的字母数字/大小写比较规则）和EXIF结果，不能以4行计数冒充真实文字验证。前几轮镜像/失败日志保留，不算最终通过。

## 复现和未完成

- 单测：`tests/release/Dockerfile.ocr-persistence`构建，运行4个测试binary的OCR/相关旧能力矩阵（机器记录保留命令/名单）。默认更宽SQLite入口不是本次全量race声明。
- 原生：`docker build --target ocr-worker-probe ...`，然后`python3 spikes/ocr-service/run_worker_probe.py --probe <immutable image> --output <fresh absolute artifact directory>`。仅可用合成测试媒体/专用只读cache，不指向真实库/DB。
- 真实模型测试通过的是标记计划停机cause与新OS进程重启；未执行本片完整OS SIGTERM/SIGKILL故障矩阵、真实图库质量/90°/复杂版面、联合容量/长期/amd64/发行。存储过期lease/三次上限与通用signal转译由各自测试覆盖，不能混说所有断电验收完成。
- OCR全文索引/融合搜索/命中来源和产品UI、历史元数据保留策略、原6010备份切换仍待后续Gate；本轮没有部署原6010、连接server222或提交推送。旧10源问题继续暂留。
