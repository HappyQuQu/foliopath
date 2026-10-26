# CR-2026-051 OCR全文索引与融合搜索

- 日期2026-10-26；POST-MVP-5 S2；FR-OCR-005、FR-SEARCH-001～003 / NFR-INT-007/009/010。
- owner：OCR继续拥有结果提交；SQLite拥有派生索引/回填/revision；search拥有融合/证据/分页；app组合显式identity；API只负责HTTP与hydration；Web复用生成client与正式组件。
- [ADR-0037](../adr/0037-ocr-fulltext-fused-search.md)。默认关闭；仅合成Docker，不部署原6010、不连接server222、不修改原件/原任务/人工整理，10源问题保留。
- 本片开发与本地验收完成；[机器证据](../evidence/ocr/2026-10-26-ocr-search-local.json)。

## 实现

1. migration47独立ocr_search_documents与external-content FTS5 trigram，结果更新/删除/级联与FTS保持一致。OCR原fenced结果/checkpoint事务同时提交索引，出错全回滚；Open按100条批次回填已存ready（含空白），不推理或自动启用。
2. 复用catalog NFKC/casefold及ftsAnchor，至少三字词用FTS筛候选，所有词最终instr验证；一二字中文和特殊符号仍字面匹配。全词AND可跨识别行，不将分数当关键词精度。
3. 搜索绑定当前部署OCR identity、当前图片源、已启用非clearing且ready库；只有ready文档参与，视频不编造OCR。关闭保留结果但隐藏文字，明确清理级联删除，旧代次不重命名/复用。当前源失败/未处理不算ready。
4. all统一后端去重，OCR作为字面证据与文件名/路径同档，和语义同时命中更靠前，保持有界heap/keyset。OCR-only不进行文本向量编码；已存文字无需在线AI。索引/配置/源/库可见性变化使快照revision失效，在stream结束和HTTP资产hydration后重检；不继续拼旧游标。
5. GET /api/v1/search/assets增加mode=ocr、实际ocr来源、ocrState/ocrCoverage及最多8个原文片段。每段最多160字符加两端省略号，1-based行号为识别顺序，不代表页面；不是全文/HTML/置信百分比。API无模型指纹/宿主路径/原异常，公开搜索沿用认证/准入。
6. 正式SearchPage、URL/query/domain adapter、生成schema和OCRResults Story同步；使用既有Badge/InlineStatus/RadioOption，预览展示真实识别片段，文本节点防脚本。紧凑可换行状态保持宽屏全宽与移动预览焦点。Story/浏览器使用明确合成数据，不是质量证据。

## 验证

- Linux arm64最终单测镜像sha256:eceff499d93f991ee6933f5ffaeffc827bd31ffa0e4a8ef1d48a363a4947a545；4个race binary共59顶层测试，全部通过、无跳过。2CPU/2GiB/no额外swap、network none、rootfs只读和512MiB临时tmpfs；exit0、OOMfalse。包括短中文/全角/重音/特殊符号、混合来源去重、范围/日期/视频排除、source/identity/关闭/清理/离线、游标和hydration fence、索引回滚、46→47回填/幂等与级联FTS完整性。
- 17表非空人工/抑制/人物/语义保护摘要在迁移与OCR写入/清理前后一致：b82c6ffed9e81fada1d2cee372d7ce2a277f5ff0d2121304d9cbe32d6e71197b，数据库完整性通过。摘要只用于本次合成库前后比较，不与历史随机夹具摘要混算。
- 最终native应用probe sha256:87ddc674ca50812c056f7d67b78afaf9f59cf3f2defedaed63ea67eab9a78a0b，AI仍固定sha256:e995f5748c36ae8de9f3dd86633e4a2f60f3f6d6b7eaaba6616bbdf6ee73da21。三独立OS进程，完成2项后计划handoff、同job续到5项，2图中英文/EXIF各4行与fixture字母数字规范化文本匹配，blank ready0，corrupt/empty真实invalid_media。认证HTTP的中文、订单号和LOCAL TEST查询每次2个实际OCR命中，OCR-only/all均覆盖；不在夹具的词0命中；重复missing0、clear后中文0命中，人工标签/revision、只读源hash/完整性保留。
- AI 4GiB/2CPU，client2GiB/2CPU；实际AI peak2,095,431,680 B，memory.events max/oom/oom_kill均0，scratch/测试容器/网络清理。没有以此声明联合4GiB容量、90°或真实质量达标。
- make fmt / fmt-check / arch-check / generate-check / lint / test / test-integration / test-e2e通过；host search/API定向race通过。Node24.22.1/npm11.19.1执行类型检查、34个搜索/URL/query测试、Vite正式构建和Storybook构建通过。构建仍报告既有大chunk提示，未修改依赖/分包策略。
- Playwright Chromium/Firefox/WebKit × zh-CN/light、en/dark × 320/390/768/1440/1920共30项通过，含真实来源徽标、OCR原文转义、预览焦点/移动可见性、无水平溢出/宽屏宽度及axe；人工查看390与1920截图。此矩阵为正式组件+有限合成HTTP合同，非真实模型质量测试。

## 合同审查和首轮发现

- 反向移除本CR增量可逐字节还原CR050 OpenAPI sha256:63b5e0e8592598d198596d65de88cbad7c0a7bf510c47b6f007cb7110a5da75f。oasdiff --fail-on WARN返回1项response enum新增ocr警告（0error），没有静默压制。按ADR0037接受此未发行切片生成SPA/API协调更新，旧cursor audience提升v2；不宣称无条件向后兼容或严格WARN门通过。
- 首轮测试中旧invalid-mode夹具仍将ocr列为非法，HTTP Verify新增方法的double未返回专用失效值；修正测试后全量通过。初次写OpenAPI锁遗漏文件名被contract gate拒绝，改为标准sha256双字段格式后通过。
- 首轮浏览器18项指出覆盖状态挤低平板/桌面结果，未放宽断言；复用token将状态改为可换行紧凑行、缩短说明，最终30项通过。

## 未完成

正式OCR管理UI、90°无EXIF/复杂版面质量、历史元数据保留策略、100k/联合容量、长期与双架构/发行仍独立。原6010备份切换没有在本轮执行；没有提交推送。不得将本片自动理解为旧十张源文件已恢复或所有OCR质量问题完成。
