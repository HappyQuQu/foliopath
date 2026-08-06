# FolioPath 变更记录

本目录保存两类只追加记录：

- `CR-*`：用户可见范围、版本归属或高风险切片的 Change Record。
- `FIX-*`：已批准切片内的例行修复，链接既有 Gate、不变量和回归证据。

记录完成或被替代后仍保留；实现应以当前产品文档、OpenAPI、migration、ADR 和 Gate 为准。

## Change Records

### MVP

- [CR-2026-001：非模态媒体预览](CR-2026-001-non-modal-media-preview.md)
- [CR-2026-002：经认证的局域网 HTTP](CR-2026-002-authenticated-lan-http.md)
- [CR-2026-003：统一品牌标识](CR-2026-003-brand-identity.md)
- [CR-2026-006：管理中心独立页面](CR-2026-006-management-center-pages.md)
- [CR-2026-009：生产前端原型一致性](CR-2026-009-frontend-prototype-fidelity.md)
- [CR-2026-010：AVI 与文件大小排序](CR-2026-010-avi-and-size-sort.md)
- [CR-2026-011：目录媒体计数](CR-2026-011-directory-media-counts.md)
- [CR-2026-012：NAS 资源模式](CR-2026-012-nas-resource-profiles.md)
- [CR-2026-013：root runtime 与零初始化 bind 数据目录](CR-2026-013-root-runtime-bind-data.md)
- [CR-2026-016：视频预览自动播放偏好](CR-2026-016-video-preview-autoplay.md)
- [CR-2026-017：默认排序与媒体布局](CR-2026-017-default-media-presentation.md)
- [CR-2026-018：直接设置资源并发数](CR-2026-018-explicit-resource-limits.md)
- [CR-2026-019：视频预览默认静音偏好](CR-2026-019-video-preview-default-mute.md)

### Post-MVP 与未来提案

- [CR-2026-004：视频故事板悬停预览](CR-2026-004-video-storyboard-preview.md)
- [CR-2026-005：媒体库自动发现](CR-2026-005-automatic-library-discovery.md)
- [CR-2026-007：运维与维护原型](CR-2026-007-operations-maintenance-prototype.md)
- [CR-2026-008：后台任务中心](CR-2026-008-task-center.md)
- [CR-2026-014：扫描后派生媒体进度](CR-2026-014-derived-media-progress.md)
- [CR-2026-015：任务可恢复、日志中心、版本更新与消息中心](CR-2026-015-operations-observability-and-updates.md)

## Routine fixes

### 2026-08-06

- [自适应故事板处理预算与四帧降级](FIX-2026-08-06-adaptive-storyboard-budget.md)
- [大视频探测与失败诊断纠偏](FIX-2026-08-06-large-video-probe-diagnostics.md)

### 2026-08-03

- [名称排序先按来源文件夹分组](FIX-2026-08-03-folder-first-name-sort.md)

### 2026-08-04

- [自动版本与用户友好更新日志](FIX-2026-08-04-automated-friendly-releases.md)
- [关于页直接展示更新内容](FIX-2026-08-04-inline-release-notes.md)
- [打开预览时保留媒体滚动锚点](FIX-2026-08-04-preview-open-scroll-anchor.md)
- [预览详情长路径换行](FIX-2026-08-04-preview-detail-wrapping.md)
- [浏览器拥有视频播放能力判断](FIX-2026-08-04-browser-owned-video-playback.md)

### 2026-07-29

- [管理员密码最低长度](FIX-2026-07-29-admin-password-minimum.md)
- [浏览媒体类型过滤](FIX-2026-07-29-browse-media-filter.md)
- [应用导航精简](FIX-2026-07-29-navigation-simplification.md)：已被 redesign 应用壳恢复取代
- [分页重试恢复过期游标](FIX-2026-07-29-pagination-retry.md)
- [redesign 应用壳恢复](FIX-2026-07-29-redesign-shell-restoration.md)
- [根目录递归保持目录当前项](FIX-2026-07-29-root-recursive-selection.md)

### 2026-07-30

- [浏览页当前目录筛选](FIX-2026-07-30-browse-directory-filters.md)
- [全局 Header 与管理导航统一](FIX-2026-07-30-global-header-management-navigation.md)
- [原型品牌对齐](FIX-2026-07-30-prototype-brand-alignment.md)
- [浏览原型滚动壳修复](FIX-2026-07-30-prototype-scroll-shell.md)

### 2026-07-31

- [AVI 解封装与失败任务重试](FIX-2026-07-31-avi-demuxer.md)
- [双语 README](FIX-2026-07-31-bilingual-readme.md)
- [Compose 无 `.env` 快速部署](FIX-2026-07-31-compose-without-env.md)
- [Docker Hub 双架构自动发布](FIX-2026-07-31-dockerhub-publish.md)
- [关闭自动 CI，改为本地验证](FIX-2026-07-31-local-verification.md)
- [Header 快速语言切换](FIX-2026-07-31-header-locale-toggle.md)
- [Header 快速主题切换](FIX-2026-07-31-header-theme-toggle.md)
- [媒体库列表自适应宽度](FIX-2026-07-31-library-list-fluid-width.md)
- [管理页按钮前景色](FIX-2026-07-31-management-button-contrast.md)
- [当前目录筛选继承递归范围](FIX-2026-07-31-recursive-directory-filter.md)
- [移除账户页首版限制文案](FIX-2026-07-31-remove-account-release-copy.md)
