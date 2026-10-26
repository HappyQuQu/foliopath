# CR-2026-035：融合搜索与命中来源

- 用户授权：按认可照片搜索原型完成来源与融合，本地Docker验证后更新进度；全部完成再汇报。
- 目标POST-MVP-5；FR-SEARCH-001（有界融合/真实来源）、002（故障与快照）、003（正式页面/原型一致）。
- Owner internal/search、SQLite adapter、api、search feature；ADR-0031；OpenAPI为wire权威。
- 不含OCR实现、人物新能力、模型/标签换代、源文件修复、server222、原6010部署或推送发行。

## 实施顺序

1. 冻结ADR/新增search/assets合同、排序和故障语义。
2. 后端服务/存储/HTTP回归；后端通过后生成client并接正式界面。
3. 默认全部来源，保留filename/path与画面语义筛选，卡片有真实来源，预览显示“为什么匹配”。
   OCR明确未接入；已载入数量不冒充全量。手机/平板/1920桌面，键盘及刷新/范围保持验收。
4. 稳定Story直接使用正式SearchPage；隔离搜索原型归档，不保留第二份生产实现。
5. 合成本地Docker：真实模型、只读媒体、独立数据，验证融合与降级，不触碰原库人工数据。

## 与原型有意差异

模型相似度只能说明检索来源，不能解释图中具体对象或给出准确率百分比；不把AI标签当成视觉证据。
正式服务异步/可失败/分页，卡片只展示实际来源。视频语义保留既有独立入口；OCR为禁用说明。
融合采用ADR-0031固定排序，全部来源视图不提供假的文件排序；独立文件名模式继续使用兼容assets端点，保留原排序/过滤并由catalog返回真实字面来源。

## 当前状态

本次搜索切片已完成，最终固定镜像本地Docker验收通过（2026-10-25）。需求、流程、UI、ADR、OpenAPI/生成client、正式Story与实现已同步。
后续经“继续下一步”授权，原6010已于12:35 +08停写全备份后更新同一dac镜像，原库中文/路径/分页/手机验收及保护复核完成，见[单独部署记录](DEPLOY-2026-10-25-fused-search-6010.md)。下述“原6010未部署”是此前独立开发验收阶段的历史边界，不代表当前部署状态。
原照片搜索Story不再单独导出；历史复合原型留在归档，不作为生产证据。正式入口：`features-search-mode-guidance--fused-results` / `--fused-unavailable`。

## 实现与边界

- catalog唯一拥有字面来源规则；查询中的所有词须联合落在文件名或库内相对路径中；支持复制的完整相对路径片段，basename词不重复归因路径。SQL只筛选候选，最终归因不因basename重复报路径。
- search流式遍历全作用域，Top-K堆最多limit+1，组合命中→字面→画面，页间去重/查询绑定/快照变更校验。
- 固定cosine≥0.20为本切片检索准入，不是质量校准；API不显示置信百分比。视频画面保留既有独立模式，OCR禁用。
- 语义暂不可用时all明确降级；取消照常取消，非法向量/身份/数据即使被legacy适配器包进availability错误也保持失败。独立推理超时可降级，用户取消不可降级。
- 查完快照后，HTTP按ID水合资产并核对内部源指纹/名称/相对路径摘要；同ID资产变化返回409，不把旧来源贴给新源。
- 不改schema、模型/身份、任务、标签/人物或原件。融合请求不启动重算。原6010未部署、未重启、未备份切换。

## 验证过程中的失败与处理

1. 旧前端测试默认假定无mode即文件名模式，已改为显式filename兼容用例并新增真正all用例，保留语义与视频回归；新日期截止时刻写入URL，保证翻页窗口稳定。
2. 初次全量Go回归的模型下载夹具在宿主临时目录返回insufficient_space；改用扩展盘TMPDIR重跑，全量与集成通过。未降低空间保护或清理用户数据。
3. 浏览器合成夹具初次缺少media-processing和preview content端点，补齐只读合同；macOS WebKit默认Tab跳过按钮，原键盘断言未过，使用固定Linux浏览器环境完整验证，未删断言。
4. 首轮真实Docker的“恢复”只等待ping健康，首次推理仍不可用，断言正确失败。改为等待真实融合响应ready（最长240秒、有间隔），验证功能性恢复而非把降级页算恢复。
5. 截图复核发现窄屏预览在集合下方且受max-height/overflow裁切，原toBeVisible不足以证明在屏幕中可用。共享MediaPreview改为窄屏首次打开定位/聚焦、完整内容纵向可达，切换媒体不重复抢焦点；集合打开预览时暂停自动追加，显式加载与关闭后自动加载仍可用。补组件回归与浏览器toBeInViewport，未靠放宽断言遮盖。
6. OpenAPI单独lint发现nullable coverage缺少显式object type；补充类型并确定性重生成client/hash后通过（两条健康检查4xx建议为既有warning）。
7. 兼容性复查补上复制完整相对路径跨父目录/basename分隔的匹配；basename单词仍不重复报路径，路径依据展示完整库内相对路径。新增Go/HTTP模型联验断言，避免默认融合漏掉旧字面搜索可找到的路径。
8. 额外宽范围`go test -race ./internal/search ./internal/catalog ./internal/api ./internal/store/sqlite ./internal/app`的SQLite整包累计达到默认10分钟超时，栈在既有GuardedUndo测试的建库迁移中；没有DATA RACE报告，不能称整包race通过。新切片定向race已通过，整包普通测试通过；单独该旧用例race复核4.246秒通过。此项不关闭全量race/长期容量门禁。

## 最终验收证据

最终本地原生arm64候选：
`sha256:dac00067b3af8857517e5a9045543a9a3aa23dab75d220c0b51282e4046336ae`；
真实AI `sha256:52f4a40c9f0169e87783b138200e24c8e96492fccf244b9d329ec1e05e45f626`。
仅本地增量候选，不是双架构发行或供应链批准。Node24.22.1/npm11.19.1，Go1.26.5。



| 验证 | 实际结果 |
| --- | --- |
| 必需Go表面 | fmt/fmt-check、arch-check、generate-check、lint、全量test、integration、runtime E2E全部执行通过；最终命令使用Go1.26.5和扩展盘TMPDIR |
| 定向race | search/app/api/sqlite的融合、故障、游标、源水合与复制路径用例通过；额外整包race超时保留，不称全量race通过 |
| 前端完整check | Node24.22.1/npm11.19.1；audit:high、生成、架构、视觉清单、types、71文件446项、Storybook全部通过。最后一次audit遇registry TLS断连，重跑成功；未跳过审计 |
| OpenAPI lint | 通过；健康检查端点无4xx的2条既有warning保留 |
| 最终Docker三浏览器 | 固定Linux Playwright1.61.1；搜索融合30项＋既有模式30项＋功能侧栏30项，共90/90；搜索320/390/768/1440/1920、中英/深浅、axe、键盘/抽屉/关闭焦点、真实视口内依据可读全部通过 |
| 最终真实模型Docker | fp-multilingual-2ce26ed7；640维真实模型与独立合成媒体，来源组合/复制路径/去重/分页/失效/范围、AI断连后真实恢复、非空人工标签、源hash、重启及database-check全部通过 |
| 可复核模型输出 | blue→blue.png为filename/path/semantic三来源（cosine0.24226004）；中文“蓝紫色背景”→blue.png为semantic（0.38611156）。仅合成样片，不是汽车识别或准确率证明 |
| 资源观察 | 最终AI峰值3384590336 B，memory.events max/oom/oom_kill=0；较早1dd8566e验收峰值曾触4GiB/max1531，无OOM。两项都保留，不关闭默认预算/长期容量门禁；原6010仍不限内存 |
| 人工截图复核 | 中文320手机、1920桌面确认来源徽标及“为什么匹配”正文可见；合成渐变图不是模型质量样本。手机预览不再裁切 |
| 原6010边界 | 最终只读inspect：主ff2ee7e4、AI52f4a40c均healthy/restart0、mem0，StartedAt仍为10-24部署时间；本轮未部署、未重启、未修改数据/任务，10项源问题继续暂留 |

机器可读摘要：[本地验收证据](../evidence/search/2026-10-25-fused-local.json)。
完整模型/身份/源保护与内存边界均记录，不将本轮完成量算作OCR、人物新工作区、质量/容量、跨架构或发行通过。
独立合成测试容器/网络和本轮可丢弃scratch在收尾移除，保留合成数据和本地候选；未清理既有实例或用户数据。
