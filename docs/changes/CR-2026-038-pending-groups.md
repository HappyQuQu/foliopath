# CR-2026-038 待处理分组目录

- 范围：POST-MVP-5 人物人工整理既有 FR-INT-010/011，细分 FR-PEOPLE-007～009；承接 CR-2026-037。负责人：face 用例、SQLite 读取适配器、API 和人物工作区。
- 问题：持久化聚类成员不会随着人工归类/排除即时删除，原目录不能据此声称所有成员仍待处理。
- 合同：列表新增 `review=all|pending`（省略保持 all 兼容）；按所选库及 kind 返回全范围 totalGroups/pendingGroups/totalFaces/pendingFaces，计数在分页和 review 过滤之前。每组返回 pendingMemberCount；pending 仅保留至少一张未归类且未排除人脸的组。memberCount 继续表示全部人脸数。
- 一致性：SQLite 以同一个只读事务流式读取活动 build 和正式人脸归属/排除投影；只保留一组及 limit+1 条页面结果，预览最多4图。同一事务形成摘要、计数和页面；摘要绑定 build、设置/库修订及每个人脸的修订和当前整理状态。游标绑定库、kind、review 与摘要，整理/换代后返回 face_cursor_stale；不用浏览器首屏推断全库状态，不增加迁移或更改写入事务。
- UI：默认待处理，提供全部分组；显示后端全范围计数及每组待处理/总人脸，筛选切换重置 groupsCursor；打开/返回保留筛选。稳定 Story 使用正式组件与明确合成合同。
- 安全：不重算模型、不修改分组历史/人工整理/原件，不操作 server222 或原6010部署；10源文件继续暂留。
- 验收：用例/API/SQLite 回归覆盖部分处理、完全处理、返回未处理、范围/分页/游标失效；本地 Docker 三浏览器及真实模型合成库验证，结束后记录实际结果。质量/容量/发行门禁不因目录修复关闭。

状态：2026-10-25 本切片开发和本地Docker验收完成；未部署原6010，不代表全部AI或发行验收完成。

## 兼容和成本

review省略保留all；新增字段是附加只读结果。v2游标不接收旧v1/重启前密钥签发游标，返回invalid_cursor；仍能从第一页重新读取。相同进程内整理变更返回face_cursor_stale，禁止越过变化继续分页。每页流式读取所选类型成员，内存受限但时间随成员数增长，本轮不代表十万媒体容量通过。

## 开发中发现与修正

- 初次SQLite夹具错误地构造了一个只有1脸的core组，违反既有最少2脸合同；增加第五个合成观测，未放宽生产验证。
- 初次合同检查提示OpenAPI摘要锁尚未更新；完成兼容审查后由SHA-256重新生成锁，未修改历史门禁。
- 真实Docker重启时单图投影先就绪、分组仍返回face_not_ready，夹具新增分组就绪等待，未将错误改成空列表。
- 重启前游标原来被API归为invalid_request，页面重试不能返回第一页；修复分组invalid_cursor映射，复用正式错误反馈处理此码。重启后签名密钥变化与同进程整理快照变化分别验证400 invalid_cursor/409 face_cursor_stale，不用同一个预期混淆二者。
- 全量Go测试在宿主默认临时目录触发旧模型下载集成测试insufficient_space：可用约31GiB，小于460GiB卷10%保护水位。改为独立本地大卷临时目录重跑，不降低保护水位、不删除用户数据。

- 后续完整Go回归遇到既有视频定时夹具超1秒预算；未修改视频处理代码或阈值，降低包构建并发后原套件通过。
- 一次npm审计TLS中断；新增游标单测最初误用Playwright的exact选项，Testing Library类型拒绝。修正测试并完整重跑，未跳过审计或类型检查。
- 补充浏览器分页夹具显式提供nextCursor，不用“总数大于本页但游标为空”的不一致夹具证明计数；添加0待处理时仍可访问历史分组的回归和正式No pending groups Story。

## 最终验收

固定本地原生arm64镜像：`sha256:0984ed15404201ddcc23727a1b2cf9b3558f408f21ebe72fa93a40990290fbbc`。
正式原型为`features-intelligence-review--unnamed-groups-sidebar`、`--all-groups`、`--no-pending-groups`，均是正式AIReviewPage与明确合成合同；没有独立生产外观副本。

| 检查 | 实际结果 |
| --- | --- |
| 后端 | 新用例/SQLite/API定向及三包定向race（count=1）通过；部分/完全整理、移出后重现、空待处理、core/edge、离线、全范围计数、过滤绑定与整理游标失效均覆盖 |
| 全局检查 | fmt-check、arch-check、generate-check、lint（contract/go vet）通过；OpenAPI lint有效（2条既有health端点4xx响应警告）；完整Go、集成及容器runtime smoke通过，部分普通Go包使用正常缓存；未声称全量race通过 |
| 前端 | 完整check通过：audit:high、生成、架构/视觉清单、types、73文件466项和Storybook |
| 同镜像三浏览器 | 分组/人物/侧栏主回归141项通过；目录补充27项通过，其中24项重跑矩阵、3项新增空状态，合计144个不同浏览器用例，不能称168个不同用例。390/768/1440/1920、中英深浅、axe、筛选清游标和类型/筛选返回保持通过 |
| 真实模型合成Docker | 2组4脸起始待处理；明确归类1脸/排除1脸后，待处理只剩1组2脸，全部仍2组、原组pendingMemberCount=0；单进程旧游标409，重启旧游标400且可重新分页 |
| 保护与重启 | 原2份合成媒体hash/数量、人工标签及其余2匿名脸保留；等模型/单图/目录分别就绪后验证重启持久化，原测试容器database-check通过；0整组质量门禁绕过 |
| 视觉 | 1920英文深色及390中文浅色目录截图人工复核，页面流式宽度、筛选换行无横溢出；不代表完整发行视觉矩阵 |
| 收尾 | 测试容器、网络、数据卷和合成源临时目录已精确清理；未部署或写入原6010、未连接server222。10源文件继续暂留 |

[机器证据](../evidence/people/2026-10-25-pending-groups-local.json)。实际探针入口：`node tests/release/cluster_review_smoke.mjs`（默认当前候选，可用FOLIOPATH_PERSON_APP_IMAGE覆盖）；原始日志在`.artifacts/pending-groups-*`，截图在`test-results/pending-groups-*`，均为本地忽略产物。
下一范围仍包括原库安全部署、人物目录进一步体验、OCR及质量/容量/跨架构/发行；本轮不擅自推进这些门禁。
