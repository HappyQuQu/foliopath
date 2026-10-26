# CR-2026-041 人脸整理的受保护撤销入口

- 授权：用户在CR-2026-040后要求继续下一步开发；既有POST-MVP-5 / FR-INT-012/016、NFR-INT-003/006；增量FR-PEOPLE-016～018。Owner：face既有undo事务、Web AI审核适配层及AIReviewPage。
- 缺口：正式撤销API和guarded快照已有后端回归，人物页面未接入；误归类/排除/移出/合并只能用新动作手动纠正。
- 合同：只使用既有POST /api/v1/ai/face-reviews/{reviewId}/undo，携带原成功回执reviewId/revision形成If-Match及独立幂等键；不刷新成新修订、不修改undo事务/数据库/API模型或模型身份。
- 单一owner：AIReviewPage只保留本页最近一次收到的可撤销成功回执及操作说明，单图/分组/人物详情共用。开始下一次人脸审核即清除旧入口；创建空人物和改名不冒充face undo。页面内导航可保留，离开/刷新不承诺恢复，不提供历史列表或全局最近动作推断。
- 交互：核对原操作、照片/人脸或合并来源/目标后确认撤销；取消不写入。成功不允许撤销撤销，冲突/已修改/不可撤销明确结束且不自动重发，未知响应显式同键重试；清除提示不清除服务端审核历史。
- 验证：先验证现有后端undo/重放/冲突，再接生成client adapter与统一Query失效；稳定Story直接使用正式组件和合成回执。单测/三浏览器/真实模型独立Docker验证后更新进度。
- 边界：本地合成数据，原6010/人工整理/原媒体只读不写入，不连接server222，10源问题暂留；无新部署/持久化/架构边界，不关闭OCR/质量/容量/跨架构/发行。

状态：本切片开发与最终本地Docker验收完成，未部署原6010；不宣称全部AI完成。


## 首轮发现与纠正

- 首次后端定向模式没有命中API测试；补跑确切TestFaceMutationRoutesDecodeTypedReviewAndStrongUndoPrecondition并在三包race及全量中覆盖，不把no tests to run算API验收。
- 新回执可选文字字段初次不满足exactOptionalPropertyTypes，英文字典插入未命中带类型声明的定义；纠正类型/双语字典，完整类型检查通过。Story初次遗漏两个由render提供的必需参数默认值，补齐合成默认值，不放宽生产props。
- 三浏览器严格端点检查发现排除后还会GET /people/：来源observer在await期间已因归属变化切到空ID，旧refetch回调因此命中空人物。移除单图页对变化来源/目标observer的强制refetch，由canonical mutation统一失效人物查询；审核/撤销失败同样刷新但不自动重发。保留严格端点断言，未伪造空人物响应来掩盖。
- 发现既有undo实际409错误码face_constraint_conflict未出现在该路由x-error-codes；同步正式合同/生成client/hash，不改变服务端行为或抹去历史错误。

## 保留边界

回执只表示当前挂载人物工作区最近收到的确认结果，不是服务器全局历史或全局最后一条审核。开始另一次人脸审核时清除旧入口；创建/改名和标签没有借此获得撤销合同。未知响应的同键意图在当前提示存续期保留；关闭确认框可重新打开，dismiss/离开/刷新不承诺恢复或取消服务器请求。

服务器已有Undo事务负责检查相关人物/anchor/exclusion的后置版本，UI不自行执行反向assign或篡改版本来绕过冲突。成功或冲突刷新状态并重置旧面板选择；不新增清理、数据库迁移、模型重算或原6010部署。


## 最终验收

最终本地原生arm64候选：`sha256:43e1684404a8c6cf0c8bd8af4adf7f60268b3b56751d9769ec27742f141805bc`。

| 检查 | 实际结果 |
| --- | --- |
| 后端 | 既有guarded undo（assignment/exclusion/split/merge）、同键重放和API强If-Match专项执行通过；face/API/SQLite三包相同范围定向race通过 |
| 全局 | fmt-check、arch-check、generate-check、lint（contract/go vet）、完整Go、集成及Linux Docker runtime smoke通过；正常缓存，独立大卷TMPDIR/GOFLAGS=-p=1，不称全量race通过 |
| 合同 | 既有路由补齐实际409错误码，生成client/hash一致；OpenAPI有效，仅2条既有health缺4xx警告，无新端点/迁移/模型身份 |
| 前端 | 最终完整check通过，76文件490项；含生成/架构/视觉清单、audit:high、types及Storybook，新增回执/适配层/Query失败刷新验证 |
| 最终镜像三浏览器 | Chromium/Firefox/WebKit共252/252：撤销、单图、目录、分组、人物详情、侧栏；390/768/1440/1920、中英深浅/axe/焦点，原回执版本、同键重试、冲突、页面生命周期和空人物ID回归通过 |
| 最终镜像真实模型Docker | 合成2图4脸；单图排除撤销、真实已提交响应丢失后的同键重放、另一会话新归类阻止旧撤销、人物移出撤销、空人物合并撤销及分组排除撤销通过 |
| 数据保护 | 最终1归类/1排除/2匿名保留，人物与待处理/全部组语义保留；非空人工标签与源hash重启前后核对，所属合成主容器Linux锁域内database-check通过 |
| 视觉/收尾 | 人工复核最终390真实模型分组及1920深色单图撤销截图，提示/按钮自然换行，无横向溢出；专属容器/网络/数据卷/临时源精确清理；原6010/server222/原媒体/人工整理未操作，10源问题暂留 |

真实模型合并撤销使用独立空人物，不把它冒充新增非空合并UI测试；非空合并撤销的既有后端回归本轮已执行。
[机器证据](../evidence/people/2026-10-25-face-review-undo-local.json)。复验入口：`node tests/release/face_review_undo_smoke.mjs`。
本地日志为`.artifacts/face-undo-*`，截图为`test-results/face-undo-final`及`test-results/cluster-review-real`，均为忽略产物。
