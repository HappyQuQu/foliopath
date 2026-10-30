# 媒体库选择记忆与扫描期间删除

2026-10-30；FR-BRW-001/004、FR-DEL-003/005；POST-MVP / DELETE-S3；Owner：Web browse / deletion。沿用 CR-2026-058、CR-2026-060 和 ADR-0044；既有认证、媒体写边界、扫描串行及 schema 49 保持。

用户报告选择媒体库后刷新回到第一个，并提供删除确认框被“有其他任务正在处理”禁用的截图。

- 当前导航的媒体库由现有浏览器偏好 owner 记住。明确的 URL 始终优先；访问首页或重新登录后优先回到上次选择。未在列表首屏的库使用既有详情 API 解析，不取全量列表；离线库保持选择，只有已移除的库恢复原默认规则，暂时服务错误显示重试。
- 预检去掉扫描状态的重复阻拦。既有提交事务登记变更范围并请求扫描取消；后台仍等待扫描/校准退出后执行删除，新扫描准入仍被范围挡住。重叠删除、库移除、离线、根目录等原约束保持，不引入新锁或调度器。
- 被阻拦的预检在原确认框提供“刷新”，重新读取当前可执行状态；不要求关闭重开。没有增加确认步骤。

接口沿用现有 library detail/list 和 deletion preflight/operations，无 OpenAPI 或迁移变化。服务器只读诊断时三库均 ready、无活动扫描/删除/移除，两个历史失败项为 source_missing 且无活动范围；截图不能证明当时具体任务，故不把历史任务改成成功或再次提交。

本地回归包括真实 SQLite 扫描→预检→提交→取消请求→扫描退出前不可 claim→退出后删除收尾，重叠删除仍被阻拦；浏览器使用明确合成 API 夹具，通过真实选择器切到第二库，刷新、访问首页、会话过期后登录均回到第二库。测试与输出本地保留。

## 验证与 222 交付

Node 24.22.1 / npm 11.19.1：类型、前端架构、生成检查、Vite 与 Storybook 构建通过；Vitest 86 文件 / 581 项通过。Go SQLite、scanner、api 相关测试，make arch-check、contract-check 与差异空白检查通过。Linux amd64 容器中真实临时库的 11 个顶层删除测试（含三个 SIGKILL 阶段子例）通过；第二库刷新/首页/过期登录的 Playwright 回归通过。浏览器和存储回归只使用合成数据。

222 原生 Go 1.26.5 / libvips 编译镜像 `foliopath:library-refresh-20261030`，版本 `v1.9.2-delete-preview.20261030-direct.1`，镜像 ID `sha256:236be06e1413723641fd2f63deb65ff7f54dbf93be641d7715336c10ba17df88`，源码包 SHA-256 `bd0c94dddd349214c72700672bfc3863b87889feb28d995eec790905d86b1d1b`。沿用既有原生依赖，无正式标签发布。

独立合成容器 healthcheck/database-check 通过后，只重建主服务。最初只读根/tmpfs 冒烟因 storage_io 启动失败，改用与既有合成验证相同的普通本地临时 bind 后通过；未操作生产库规避问题。部署脚本的最终 Compose 比较最初缺工作目录失败，已从保留的升级前基线单独完成比较，没有重复部署或覆盖备份。

9090 新版本、healthy、database-check 和 8 项静态文件/SPA 路由逐字节比较通过。有效 Compose 只变更主服务 image；数据/媒体挂载与端口不变，AI 的 ID、镜像、启动时间、重启次数、挂载完全一致。schema 49 无变更，不复制在线 SQLite；同 schema 的 `foliopath:no-trash-20261030` 和升级前 Compose 保留可回退，无需用旧数据库覆盖后续用户工作。

服务器证据与配置备份在 `/root/foliopath/previews/delete-20261030/library-refresh`。测试容器、合成媒体/数据目录和本地 Vite 已清理；未修改 PVE、原件、原账号或删除策略。
