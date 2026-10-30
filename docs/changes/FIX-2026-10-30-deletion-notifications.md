# 删除结果进入消息中心

2026-10-30；用户要求“删除文件结果在右上角小铃铛上通知”。FR-DEL-005/007、CR-2026-058 / POST-MVP DELETE-S3；Owner：deletion / Web notifications。沿用 ADR-0044、CR-2026-060 的永久删除和一次确认，不增加删除流程或部署单元。

复用现有消息中心和文件任务合同：全局读取最近一页（最多 25 个）任务，新永久删除显示等待/处理状态及成功、部分完成、失败、暂停、取消或需要处理的结果；带目标名称、所属库、完成/失败项数、时间。角标包含这些消息，关闭面板也按既有两秒任务轮询更新；点击进入文件任务查看详情。提交后原删除状态失效规则使该页立即刷新。

现有 operations 列表原来只返回头部，现沿同一服务的 GetOperation 读取每项的第一个目标及原有后续游标；最多一页，没有遍历目录、无新 API 字段或数据库迁移。先使用真实 SQLite/Linux 回归验证列表目标与结束结果，再接生产通知组件；接口范围和鉴权不变。

终态消息在点击或“清除结果通知”后，用既有 browser preferences owner 保存 operation ID + revision 的本地已读标记，最多 200 个，与扫描完成标记共存；活动任务不能清除。同任务继续后产生新修订的结束结果可再次提示。清除只影响消息，不停止任务、不删除历史记录。只展示本页新永久删除，不恢复旧回收产品入口；完整历史仍在文件任务分页。

不增加 WebSocket、通知服务或持久数据库表；既有查询和状态 owner 共用，活动状态由 deletion 公共函数统一判断。稳定 Storybook 直接渲染 NotificationCenter，使用明确合成任务夹具，不另造页面。原件、认证、扫描串行、媒体边界和删除开关保持。

## 验证与部署

固定 Node/npm 的类型、前端架构、生成检查、Vite 和 Storybook 构建通过；完整 Vitest 86 文件 / 582 项通过。消息中心回归覆盖关闭时轮询完成、任务链接、点击/清除记忆、保留活动项、重载后不重复，以及同任务新修订再次通知。Go SQLite、API 及 make fmt-check、arch-check、contract-check 通过。

Linux amd64 候选容器内 11 个顶层删除测试及三种 SIGKILL 阶段子例通过，真实服务额外断言操作列表带目标名及结束结果。独立合成图库浏览器验收通过：实际删除 album，核对目录和后台结果，再由铃铛结果进入文件任务、刷新确认已读记忆；other/keep.txt 保留。未用生产媒体执行删除验收。

生产 NotificationCenter 稳定 Story 在中英、明暗、390/768/1440/1920px 的 16 组中，状态、数量、目标、链接、清除、Escape/焦点、横纵向面板边界和 axe 均通过；已检查移动/宽屏截图。最初外部 axe 与 Storybook 自动检查并发导致三个校验运行失败，改为该 Story 手动模式、由外部矩阵统一执行 axe 后全通过；Story 测试壳补 alignItems 起始对齐，排除测试壳将锚点撑满高的显示干扰，无生产样式修改。测试及截图仍本地保留。

原生 Go 1.26.5/libvips 镜像 `foliopath:delete-notice-20261030`，版本 `v1.9.2-delete-preview.20261030-direct.3`，配置 ID `sha256:6ea26be1f6babfcbd68f6f894fdaee7f25d56e19b3092f3e3933fce54958648f`；源码包 SHA-256 `b338dae7764c078d02f8132d950ddd9bbc262ecbc42233f52a2d526dbfdd0e6d`。同一候选先通过合成后台 healthcheck/database-check、真实浏览器和 8 项静态文件/SPA 路由逐字节校验，再切换 222 主服务。

9090 version、healthy、database-check、8 项嵌入页面字节校验通过。有效 Compose 只变更主服务镜像，端口、数据/媒体挂载和 AI 的 ID/镜像/启动时间/重启次数/挂载完全一致；schema 49 保持，不复制在线 SQLite，不改 PVE、删除策略或原账号。升级前 Compose、`foliopath:no-ellipsis-20261030` 镜像和证据在 `/root/foliopath/previews/delete-20261030/delete-notice`，无需恢复旧数据库即可同 schema 回退。正式标签未发布。

最终收尾将通知 DOM key 固定为 operation ID，已读键仍绑定 revision，避免进度更新重新创建链接而丢失键盘焦点。新增真实查询更新前后焦点断言通过，相关 2 项消息中心回归、类型和 Vite 再次通过；后端源码与上述原生/真实删除验收候选逐字节一致。

最终 9090 采用 `foliopath:notice-final-20261030` / `v1.9.2-delete-preview.20261030-direct.5`，镜像 ID `sha256:cdda13558dd7cf6ed5ba569e2a8d5ff66421684767710c06860f6a625109f88e`，源码 SHA-256 `a4c2df5e39dc8b154c601fd74c80e4d4baa8c7cc75c81c57951f26d2d75803be`。最终配置/基线在 `/root/foliopath/previews/delete-20261030/notice-final`；只切换主 image，挂载/端口/AI 再次比较一致。测试容器、合成媒体/数据、SSH 转发及本地预览均已清理。
