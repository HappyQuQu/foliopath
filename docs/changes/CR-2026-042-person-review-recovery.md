# CR-2026-042 人物详情审核请求恢复

- 授权：CR-2026-041后继续开发；POST-MVP-5既有FR-INT-010/012/016、NFR-INT-006；增量FR-PEOPLE-019～021。Owner：face既有审核/幂等事务、Web PersonPanel及统一AI Query owner。
- 缺口：人物详情每次提交重新生成key，未知响应即关闭确认框；已提交移出/移动会让照片消失，已提交合并会让源人物404，当前早退界面丢失重试和成功回执，无法接统一撤销。
- 后端先行：沿用既有POST face-reviews（split_face/assign_face/merge_people）和请求哈希/同键重放合同。先验证已提交合并/移动/移出后仍可重放，不能以新版本替换原确认。不新增API/数据库/模型身份。
- 正式交互：确认时冻结确切请求、来源/目标和单个key。未知响应保留可见确认或本页恢复提示，关闭可重新打开显式同键重试；未解决时禁止该面板发起新审核/改名。源人物/照片已不可读也不隐藏恢复入口。终态冲突不自动重发；成功后才发布原回执到统一撤销入口，合并再返回目录。
- 生命周期：只保留当前挂载人物详情的一次意图；收起本页恢复提示或离开/刷新不承诺恢复，不取消服务器请求，也不删除人工整理。改名PATCH没有face审核幂等合同，本轮不宣称改名可同键恢复或撤销。
- 原型：稳定生产Person detail及合成fixture直接复用正式组件；需求/流程/UI/测试同步，不新建设计分支。
- 验收：单测、三浏览器及真实模型独立Docker验证先提交后丢失响应，再同键确认、获取回执和guarded undo，含合并源404/移出照片消失/并发冲突/双击/取消/恢复提示。
- 边界：原6010/原媒体/人工整理不写入；不连接server222，10源问题暂留；不部署/推送/重算，不关闭OCR、质量/容量/跨架构/发行。

状态：本切片开发与最终本地Docker验收完成，未部署原6010；不宣称全部AI完成。


## 首轮问题

- 正式应用定向浏览器最初缺少目录使用的thumbnail夹具端点，严格unexpected检查失败；补齐确切合成thumbnail响应，保持断言，不修改生产预览或虚构人物数据。
- 静态Story自动播放首次使用React测试库act包装，生产Story bundle中act不可用，停在初始页面；改用已安装storybook/test工具。最终4项独立Docker静态Story核对通过，没有以仅构建成功冒充交互通过。
- 静态服务器Python能力探查不可用，改用同固定浏览器镜像已有Node只读静态服务；仅内部临时测试网络，无新安装/生产服务。

## 最终验收

本地原生arm64固定候选：`sha256:05868d43cee18ae32011f58bdbe56ac21eb5e20590a0e1cddee365f07b8b7e39`。

| 检查 | 实际结果 |
| --- | --- |
| 后端 | 新增split已提交后空人物/merge已提交后source404的同键重放：回执不变、不重复增版本、guarded undo保持；既有人脸移动/API专项及face/API/SQLite定向race执行通过 |
| 全局 | fmt-check、arch-check、generate-check、lint（contract/go vet）、完整Go、集成及Linux Docker runtime smoke通过；正常缓存，独立大卷TMPDIR/GOFLAGS=-p=1，不称全量race通过 |
| 前端 | 最终完整check通过：76文件497项、audit:high、生成/架构/视觉清单、types及Storybook；最后补充Story测试类型检查通过 |
| 正式应用浏览器 | 同一固定镜像Chromium/Firefox/WebKit共288/288；恢复/撤销/单图/目录/分组/人物详情/侧栏。恢复矩阵390/768/1440/1920，中文浅色/英文深色及axe；源404、照片消失、原body/key、无提交后的版本冲突、本地丢弃/刷新边界与回执撤销通过 |
| 真实模型Docker | 真实提交移出/移动/来源非空合并后丢响应；首次Idempotency-Replayed=false，同原body/key重试true；合并源GET404仍可恢复，确认后真实guarded undo，人物版本每次仅提交＋撤销各增一次 |
| 实际数据范围 | 合并来源1脸/1图、目标0；双方非空的合并重放/undo由本轮执行的后端测试验证，不冒充新增双方非空真实UI样本。最终1归类/1排除/2匿名保留，非空人工标签/源hash在重启前后保持，所属合成主容器database-check通过 |
| 正式原型 | features-intelligence-review--person-detail-recovery直接使用生产组件，通过Story播放合成503/源404，截断所有API写入；独立Docker静态Story Chromium4/4（中英、390/1920）通过，不把合成响应当实际服务器结果 |
| 视觉与收尾 | 人工复核真实模型1920合并恢复及390中文Story，模态/提示/固定上下文无横向溢出；测试专属容器/网络/卷/临时源精确清理。原6010/原件/人工整理/server222未操作，10源问题继续暂留 |

后续Story fixture与导入修正不改变生产bundle；288项及真实模型均在同一05868d43候选执行，Story单独记录构建/交互证据。
[机器证据](../evidence/people/2026-10-25-person-review-recovery-local.json)。复验入口：`node tests/release/person_recovery_smoke.mjs`。
本地日志`.artifacts/person-recovery-*`，截图`test-results/person-recovery-final`、`test-results/person-recovery-story`及`test-results/cluster-review-real`均为忽略产物。
离开/刷新/收起不承诺保存未知请求，改名仍没有face幂等/撤销合同；OCR、不能合并关系UI、质量/容量/跨架构及发行未因本轮关闭。
