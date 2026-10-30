# DELETE-S3 本地 UI 集成验收

> 后续简化：用户明确接受 ADR-0044，撤掉 Landlock/隔离子进程并授权 222/9090 候选部署。本文保留原阶段的历史验证，不代表当前实现仍要求内核隔离。当前验证见部署记录。

2026-10-30；POST-MVP-DELETE；Owner：Web deletion / browse / design system / API。

**本地 UI 集成 Go；S4 发布和真实可写部署 No-Go。** 前序 [S1 合同](delete-s1-contract-ready.md)、[S2 后端](delete-s2-backend-ready.md) 允许本切片；需求 FR-DEL-001～010、NFR-DEL-001～004、FR-BRW-016/017，目标发布号仍由 S4 指定。

## 交付

- 浏览、搜索、收藏和标签页面共享右侧选择模式；点击“选择”才显示复选框，左侧目录树不勾选。仅选已载入项，最多 200 个顶层对象；更换范围清空选择，归档成员和不可用来源有禁用原因。
- 目录标题一个操作菜单，桌面右键和 Shift+F10 指向被操作目录；顶部指向当前目录，根和包内目录受保护。全部展开/收起采用虚拟行及最多两个目录请求，迟到结果不会重开已收起树。
- 侧预览和查看器复用单项删除确认；整归档明确命名。预检显示后端去重范围及回收策略，永久操作输入本地化确认词；未知提交结果保留同一请求键重试，不能伪装完成。
- 管理导航新增“文件管理”：回收站、文件任务、实例设置。回收站按库/游标分页，恢复无覆盖，清理及清空明确范围；文件任务展示状态、明细和后端允许的暂停/继续。设置默认禁用、回收开启、仅手动清理，危险变化先确认并预览到期数量。
- 页脚、媒体库添加确认、关于页及 README 的绝对“始终只读”表述已改为默认只读/扫描只读，说明删除需要明确开启；普通浏览、分析与配置移除仍不修改原件。
- 原语沿用 canonical Button/Dialog/Menu/Select/Switch/InlineStatus，新增共享 Checkbox；生成客户端和 hand-written adapter 是唯一 HTTP 入口。稳定生产组件 Story 使用标明合成的合同夹具，不维护第二份 UI。
- 布局使用流式宽度和既有管理侧栏；库移除说明回收站/未完成文件任务（含暂停）的阻止条件，并给出处理导航。

实现与原设计的名称/任务入口差异、控件到合同映射见 [方案](../../architecture/media-deletion-contract.md#s3-落地差异与入口追踪)。

## 实际验证

- `make fmt`、`make generate-check`、`make lint`、`make test`、`make test-integration`：通过。`make compatibility-check OPENAPI_BASELINE=/tmp/foliopath-deletion-openapi-baseline.yaml`：通过，无 breaking changes；新增 wire 字段及错误类型在更新 revision lock 前经过该兼容检查。
- 按固定 Node 24.22.1 / npm 11.19.1 执行 `npm ci`；Web production build、类型、架构、OpenAPI lint、既有 visual-reference manifest 检查通过；OpenAPI 仍有原先两个 health 4XX 提示。全量 Vitest **86 个文件 / 570 项通过**，包含永久确认/断线同键重放、共享勾选的键盘/禁用语义、选择模式不触发预览，以及原触发项/对话框同时卸载后的焦点回退；Storybook 构建通过。manifest 检查只验证既有引用完整性，不冒称新增删除页面已有旧发布视觉基线。
- Playwright 目录操作 **16 组通过**；新增真实后台删除 UI **17 组通过**：完整开启、整目录回收、恢复、永久删除和稳定目录 ID，另有中英 × 明暗 × 390/768/1440/1920px。每组检查确认、回收站、任务、设置的 axe 与横向溢出，并留截图；已检查移动确认、宽屏回收站及设置布局。
- 真实后台使用新编译 Linux arm64、`CGO_ENABLED=0` 的开发二进制、已有运行时和临时 Linux 数据/媒体卷；浏览器没有拦截这些 API。认证和 CSRF 沿真实实现，响应式用例复用已授权测试会话，没有放宽登录限流。合成图片/文本和目录由专用 fixture 准备，未使用真实媒体。
- 关于页的默认只读/显式启用说明另有 **16 组** 稳定生产 Story 检查通过（中英、明暗、四档宽度，axe/溢出/截图），使用合成版本合同，不冒称真实后台验证。静态构建完成后再预览，避免重建时删除资源导致的测试加载失败。三套浏览器输出分别保存在本地 `test-results/playwright-directory-actions`、`playwright-deletion-real`、`playwright-deletion-policy`，互不覆盖。
- 重新编译的 files 原生边界与 SQLite 强杀恢复测试通过，三个 SIGKILL 阶段及完整 HTTP 幂等/恢复/永久删除通过。候选 `compose.media-mutations.yaml` 与基础、simple AI、独立 AI + data-volume + multilingual + OCR 的配置合并差异检查通过：只显式授权媒体变更和同一媒体根可写，要求现存目录，不改 data/AI/服务或增加能力；没有执行实际部署。

## 仍未通过的发布门禁

Vite 构建保留大 bundle 提示；未通过提高阈值隐藏。生产 libvips 冒烟镜像构建两次受 Docker Hub EOF 阻断，临时等价构建又受 Debian 仓库不可达阻断，`make test-e2e` **未通过**。真实页面开发二进制不替代生产镜像、原生 amd64/arm64 配对、支持存储重启/备份恢复或 100k/10k 容量和性能验收。

基础 Compose 仍只读，应用删除默认关闭；可写覆盖仅为候选，不得在 S4 通过前用于真实媒体。测试源、截图、夹具和输出依仓库政策仅本地保留，不增加发行输入 allowlist；未提交生成 Vite 输出、运行数据库或媒体缓存。本轮未发布镜像、修改真实实例或变更其挂载权限。
