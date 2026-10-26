# CR-2026-044 人物合并冲突预检

- 授权与范围：CR-2026-043后用户要求继续开发；POST-MVP-5 FR-INT-011/012/016、NFR-INT-006，新增FR-PEOPLE-025～027。本轮补充既有合并流程的只读原因说明，不新增强制合并、关系删除或全局历史。
- Owner：face合并预检用例、SQLite原合并约束查询、API/生成adapter与PersonPanel确认框。先固定合同并测试后端，再接正式组件和Story。
- 合同：GET /api/v1/people/{personId}/merge-check?targetPersonId=...，单快照返回双人物/版本、canMerge及cannotLinkCount/duplicateAssignmentCount/sourceLimitExceeded。最多3个可读照片关系示例（仅opaque库/媒体ID）；离线或无当前bound照片时不暴露示例，但关系仍阻止合并。没有示例不代表没有冲突，不伪造脸号。
- 一致性：预检与POST合并复用同一约束查询；POST仍在写事务重新检查，预检不是写入授权、租约或保证。新确认只有预检已知可合并且版本与已选快照相同才可提交；状态变化需重新选择/核对，不静默采用新版本。已有未知响应merge意图必须继续同键恢复，不能被源404/预检变化破坏。
- UI：目标选择后显示检查中、不可用、快照已变、禁止关系/重复归属/批次上限；可读示例打开照片核对，最多3组，区分总关系数和示例数。不提供清除约束或覆盖按钮。正式Story渲染生产组件及标注合成合同。
- 数据与部署：无迁移/新模型身份/新部署单元或事务owner。仅本地独立合成Docker；不连接server222、不部署或写原6010/原件/人工整理，10源问题保留，不推送。
- 验证：读前/后不能有任何写入；反向人物、跨图/跨库关系、离线隐藏示例但保留阻断、后续人物变化/关系变化仍由POST拒绝、缺失/非法目标、批次上限、旧请求恢复和既有undo回归。真实模型合成Docker验证正式GET及UI合并阻断/解除/重试和源hash/人工标签/重启/数据库。

状态：本切片开发及最终本地Docker验收完成；原6010未部署本候选，不代表全部AI或发行完成。

## 验证进展与首轮修正

- 新API初始编译引用了不存在的opaque ID helper名，改为复用既有libraryID/assetID映射后通过。新增非法ID测试误将长度合法但不存在的opaque字符串当400，修正为实际长度非法输入；没有扩大路径规则或访问文件系统。
- 组件首轮新增失败仅为复用ErrorState的重试按钮实际叫“重新尝试”，修正测试文案后21项PersonPanel测试通过；原15项包括合并成功响应丢失/源404同键恢复及guarded undo上下文。
- 全量合同首轮准确指出新GET未进入精确operation白名单、摘要文件遗漏既有双列格式；按本变更记录补充单个端点并恢复`<sha256>  openapi.yaml`格式，保留精确合同检查，重新执行全量。
- 后端定向及race通过：face、SQLite、API、app组合。跨库5个人脸关系限制为3个不同照片示例；原允许预检后增加关系，人物版本不变的旧POST仍409；读取不写审计或归属。
- 前端完整check通过：76文件510项，包括21项PersonPanel，audit:high、生成/架构/visual inventory/types和Storybook。OpenAPI lint有效，两个既有health缺4xx warning保留。
- 正式静态Story合成Docker12项通过：新增合并阻止4项、原请求恢复4项和同图关系4项；中英深浅、390/1920，未伪造API成功。
- 固定arm64候选c2969f15真实模型5阶段通过；新预检未登录401、真实禁止关系/照片示例与确认框禁用、解除后允许，以及原重放/恢复/undo、非空人工标签/源hash/其他脸、重启和原测试容器内database-check均通过。

## 最终验收

- 最终本地arm64候选`sha256:c2969f15e506e45d5d0732bca7ca3b72a2aca8b096f41e767e32eaa6e5d86501`（`foliopath:local-merge-check-20261025`）的完整三浏览器363/363通过：新增预检39项，加前轮工作区兼容324项。四档宽度、中英深浅、axe、取消焦点、读取失败只重试读取、版本不采用、离线无示例仍阻断与各原因说明均通过，没有新失败。
- `make fmt-check arch-check generate-check lint test test-integration test-e2e`最终通过；全量使用专用扩展盘TMPDIR及GOFLAGS=-p=1，不放宽生产预算。Go/race与前端完整76文件510项/21项PersonPanel通过。OpenAPI有效，两个既有health警告保留。generic runtime smoke使用独立bb525685，不冒充最终SPA镜像。
- 同固定候选真实模型5阶段及静态正式Story12项通过，见上文。目检1920正式确认框与390正式Story截图，无横向溢出；测试容器/数据卷/网络已清理。
- [机器证据](../evidence/people/2026-10-25-person-merge-check-local.json)。未部署或修改原6010/原件/人工整理/任务，未连接server222，未提交推送。10源问题继续暂留；OCR、全局/跨图关系管理与历史、质量/容量/跨架构和发行仍独立保留。
