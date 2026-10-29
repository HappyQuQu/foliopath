# CR-2026-055：ZIP 图片浏览首版

- 日期：2026-10-29。用户先讨论方案，随后明确要求“切分支 做一版看看”。
- 分支：`codex/zip-image-browsing`；目标：POST-MVP / ZIP-S1；先完成本地可运行试用，再依用户后续授权合并发布。不修改冻结 MVP。
- Owner：media（来源/指纹）、files（包解析/临时展开）、scanner/SQLite（扫描/持久化）、catalog/API（浏览合同）、Web browse（生产目录组件）。
- 合同：[ADR-0040](../adr/0040-read-only-zip-images.md)、[来源合同](../architecture/zip-source-contract.md)、`api/openapi.yaml`。

## 验收映射

| ID | 能力及验收 |
| --- | --- |
| FR-ZIP-001 | 扫描 ZIP 为只读虚拟目录，保留内部层级及真实图片数量；原 ZIP 字节不变 |
| FR-ZIP-002 | ZIP 封面/标记、目录进入返回、自然名称顺序、既有缩略图/预览/连续查看 |
| FR-ZIP-003 | 源更新失效、同逻辑成员保留人工整理、失败包保留旧成员并解释恢复动作 |
| FR-ZIP-004 | 加密/损坏/不支持/超过额度明确提示；视频/嵌套包跳过；AI 不准入包内成员 |
| NFR-ZIP-001 | 入口路径与数量/输出额度、取消清理、只读内核边界、现有任务有界并发和派生存储 |

首版使用请求级单成员临时文件，关闭即删；不实现跨请求原图 LRU 缓存。统一资产带来的字面搜索、收藏和手动标签沿用现有合同，不另造功能入口。所有验证使用合成媒体及独立数据目录。

## Gate 与证据

- S0：来源/存储/API 合同先行；2026-10-29 已记录。
- Backend Ready：2026-10-29 通过真实 files.ScanWalker→scanner→SQLite 来源集成（`go test ./internal/store/sqlite -run TestZIP -count=1`），及组合应用 HTTP 认证/目录/自然排序/原图/Range/ETag/改源拒绝（`CGO_ENABLED=0 go test ./internal/app -run '^TestComposedZIPBrowseAndRange$' -count=1`）。files/media/thumbnail 全包回归通过；随后授权接入正式 UI。
- UI/浏览器：正式 BrowsePage 接入后，真实 Linux arm64 应用已通过中文暗色/英文亮色、390/768/1440/1920 四档根目录和包内页面验证；生产组件 ZIP Story 共四个。最终镜像复验另列下方。
- 工程检查与可运行预览：实施后逐项记录，不将未执行的测试记为通过。

- OpenAPI：`make compatibility-check OPENAPI_BASELINE=/tmp/foliopath-zip-openapi-baseline.yaml` 通过，无 breaking changes；更新审阅后摘要并从源重新生成客户端。`npm run lint:openapi` 通过，保留 health 接口既有两项 4XX 文档警告。
- 工具：Node 24.22.1 / npm 11.19.1；`npm ci --ignore-scripts` 通过，依赖审计报告两项既有 moderate。

## 已执行验证

- `make fmt`、`CGO_ENABLED=0 make arch-check contract-check generate-check`、`CGO_ENABLED=0 make lint`、`CGO_ENABLED=0 make test test-integration` 已通过。迁移48导致旧本地测试固定版本47的断言失败，更新预期并确认原字段数据保护后重跑通过；目录HTTP旧JSON断言同步新增null字段后通过。
- `go test -count=1 ./internal/media ./internal/files ./internal/thumbnail` 及同包 `-race` 通过；SQLite 全包通过，ZIP专项包含来源更新保留ID/收藏、坏包保留、真实删除、四类AI排除及自动发现fallback。
- 独立代码复查修复两项：ZIP条目按路径组件排序，保证跳过系统目录时子树连续；封面只认当前 thumbnail transform。两项均有本地真实实现回归。
- `make test-e2e` 通过：Linux容器内真实进程、认证边界、原件只读、重启和无AI服务下核心就绪。
- 前端 `npm run build`、`npm run check` 通过；check包括高危审计、生成/架构/视觉清单、类型、79文件538测试及Storybook构建。Vite保留既有大chunk提示。
- 使用生产Dockerfile构建Linux arm64试用镜像，在独立命名卷与只读合成媒体上启动。HTTP真实扫描9张图片、中文隐式目录/空目录、跳过视频与嵌套包、封面、查看器自然顺序、20张四宽度/双语言主题截图、严重/关键无障碍检查、错误提示、原件SHA256不变通过，浏览器pageErrors为空。容器内 `database-check` 通过。

## 试用与已知边界

- 预览：`http://127.0.0.1:6020/libraries/lib_1/browse`，容器 `foliopath-zip-preview-20261029`；数据为独立Linux卷 `foliopath-zip-preview-data-20261029`，媒体为 `/tmp/foliopath-zip-preview/library` 合成样本。原6010/6011实例未改动。
- 本地截图、日志、测试账户/session只保留于 `/tmp/foliopath-zip-preview`；不入Git、不记录凭据。测试源按既有规则仅本地保留。
- 首次生成封面期间显示ZIP图标；生成完成后刷新目录即可显示封面。没有新建无法区分“等待生成/全部失败”的持续轮询。
- 本地试用阶段没有长期容量或原生amd64发行验收，不作为正式发行证据。回退迁移后不能让旧镜像把虚拟成员当普通磁盘资产；真实实例升级/回退仍必须保留并恢复完整的升级前备份。

## 最终分支试用结果

最终源码执行 `make fmt` 与 `CGO_ENABLED=0 make arch-check generate-check lint test test-integration` 全部通过；组合ZIP HTTP测试另以 `go test -race ./internal/app -run '^TestComposedZIPBrowseAndRange$' -count=1` 通过。最终生产Dockerfile镜像重建后只替换6020试用容器，同session/同数据的完整20截图浏览验收再次通过，容器healthy且同内核database-check通过。镜像配置ID为 `sha256:8b8fba656d95cf2ccc17918e355e934fce8a1e76d63f6cd6ddb387f50a313695`，这是本地arm64试用证据，不是双原生发行标记。

试用交付时，预览已在本聊天浏览器打开并登录，停留在京都图集ZIP内部；代码尚未提交、推送或发布，既有6010/6011实例不变。

## 正式发布授权与门禁

2026-10-29，用户验看试用后明确要求“更新readme 提交推送合并发布”。据此将 ZIP-S1 纳入后续正式版本，沿用 Release Please 和 main/AI × amd64/arm64 原生构建验证及摘要推广流程。README 中英版与 Docker Hub 概览同步说明格式、限制与完整备份回退要求。

本地 ZIP 专项与浏览器验收仍为上述证据；现有原生发行烟测覆盖镜像、全新迁移、同版本备份恢复和运行时边界，不冒称新增 ZIP 专项或历史 47→48 数据升级双原生验收。测试文件仍只在本地，25 项发行白名单不扩展。正式发布须等四个原生镜像验证和两个镜像推广任务全部成功，结果另记录。
