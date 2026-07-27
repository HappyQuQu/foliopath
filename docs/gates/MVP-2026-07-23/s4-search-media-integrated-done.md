# Stage 4 搜索与完整查看器 Integrated Done

## 结论

**Go — Stage 4 搜索、共享非模态预览与完整媒体查看器纵向切片 Integrated Done。**

统一搜索、搜索结果预览、可直达完整查看器、GIF/原生视频策略、真实 Range 和媒体降级
状态已经连接冻结后端契约，并完成真实成功链、受控故障、桌面/移动输入、主题、响应式与
可访问性证据。该结论允许进入 Stage 5 发布加固，不表示 FolioPath 已形成发布候选。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 4 / `S4-009`
- 需求：`FR-SRH-001～004`、`FR-MED-002～008`、`FR-UI-001～007`、
  `NFR-SAFE-001`、`NFR-SEC-001～002`、`NFR-PRIV-001`、`NFR-ACC-001`、
  `NFR-PERF-001～002`
- 前序 Gate：[搜索 Backend Ready](s4-search-backend-ready.md)、
  [原媒体内容 Backend Ready](s4-media-content-backend-ready.md)、
  [媒体交互矩阵](s4-frontend-media-matrix.md)
- 权威契约：`api/openapi.yaml`
- 搜索 feature owner：`web/src/features/search`
- 完整查看器 feature owner：`web/src/features/media`
- 共享预览、查看器与媒体状态 owner：`web/src/components/patterns` 与
  `web/src/lib/media/availability.ts`
- URL、Query key、生成 client adapter、焦点恢复与预览状态机继续保持唯一 owner。

## 验收判断

| 判断项 | 证据 | 结论 |
| --- | --- | --- |
| 真实纵向成功链 | 一次性 SQLite、只读两层合成 `/library` 与真实 Go 进程执行 setup → 建库/扫描 → 搜索 → 非模态原图预览 → 完整查看器 → 返回搜索 | 通过 |
| 搜索与 URL | 当前库/当前目录/全部库范围、类型/日期、排序、无结果、历史前进后退与规范 URL；页面不保存 cursor | 通过 |
| 非模态预览 | 搜索命令和结果保持可操作；固定后筛选移出结果仍保留唯一媒体，清除筛选后关闭恢复真实卡片焦点 | 通过 |
| 完整查看器 | 搜索预览显式进入 opaque-ID 查看器；关闭按钮初始聚焦，`I`/Escape 可用；退出准确恢复原查询 URL 与触发卡片焦点 | 通过 |
| 媒体交付 | 图片/GIF 使用认证原内容，视频使用原生 controls/playsInline/poster；真实合成 MP4 观察到浏览器 Range 和 `206 Content-Range` | 通过 |
| 降级状态 | offline、missing/unreadable、probe failed/unsupported、unsupported codec、deleted 与运行时失败使用唯一策略；保留关闭和有界导航 | 通过 |
| 安全与隐私 | 只使用 opaque ID、library-relative 路径和同源内容 URL；不暴露 host path，不自动修改、移动或删除原媒体 | 通过 |
| 响应式与主题 | 搜索/预览/查看器覆盖 390、1024、1280px 与浅/深主题；无页面级横向溢出，移动信息面板不遮挡恢复操作 | 通过 |
| 键盘、触摸与可访问性 | 工具条焦点下快捷键、原生视频/表单冲突保护、Pixel 5 触摸恢复、语义状态与焦点还原；axe serious/critical 为零 | 通过 |
| 设计一致性 | 原型搜索结果/空状态/完整查看器与生产同状态审查记录均为 passed；S4-009 不新增视觉模式 | 通过 |
| CI 固化 | `make test-web-e2e` 使用锁定 Chromium、一次性真实后端、受控媒体矩阵执行完整纵向链 | 通过 |

## 自动与设计证据

- `web/tests/e2e/auth.spec.ts`：真实建库/扫描后的搜索 → 预览 → 查看器 → 精确返回与焦点恢复，
  以及固定预览筛选保留、scope 历史和 axe/overflow。
- `web/tests/e2e/media-matrix.spec.ts`：真实 MP4 Range/206、键盘/触摸、codec/offline/
  deleted 组合。
- `web/src/features/search/**/*.test.tsx`、`web/src/features/media/**/*.test.tsx`
- `web/src/components/patterns/MediaPreview/**/*.test.tsx`
- `web/src/components/patterns/MediaViewer/**/*.test.tsx`
- [S4-004 搜索界面](s4-frontend-search.md)
- [S4-005 搜索预览](s4-frontend-search-preview.md)
- [S4-006 完整查看器](s4-frontend-media-viewer.md)
- [S4-007 媒体策略](s4-frontend-media-strategy.md)
- [S4-008 交互矩阵](s4-frontend-media-matrix.md)
- `web/design-qa.md` 与 `web/qa/s4-*`
- `.github/workflows/ci.yml`

本地实际执行并通过：

```text
npm --prefix web run check
npm --prefix web run build
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
make test-web-e2e
```

## 保留限制

- Firefox、Safari/WebKit 的最终版本、物理移动设备媒体栈、代表性低性能 NAS 客户端
  FPS/RSS 与发布级视觉回归仍由 Stage 5 固定。
- 发布镜像双架构构建、可信代理、非回环网络、只读 volume 运行期卸载、备份/恢复/升级和
  发布签署仍是 Release Gate，不因前端 Integrated Done 自动通过。
- MVP 不转码、不提供显式下载或完整 EXIF，也不承诺浏览器无法播放的 codec。

## 交接

- 后端：搜索、资产详情、缩略图和原内容 Backend Ready。
- 前端：Stage 4 搜索与完整查看器 Integrated Done。
- 允许的下一步：Stage 5 发布加固与 Release Candidate 验收。
- 评审日期：2026-07-28。
