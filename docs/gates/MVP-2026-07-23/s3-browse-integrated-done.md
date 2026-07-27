# Stage 3 浏览与非模态预览 Integrated Done

## 结论

**Go — Stage 3 浏览、缩略图集合与非模态预览纵向切片 Integrated Done。**

真实目录树、cursor 浏览、ready 缩略图、虚拟 grid/masonry、完整浏览状态矩阵和共享
非模态图片/原生视频预览已经连接冻结后端契约，并完成真实成功链、故障状态、容量、
响应式、主题和可访问性证据。该结论允许前端进入 Stage 4 搜索与完整查看器，不表示
完整 FolioPath 或发布版本已经完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-BRW-001～009`、`FR-MED-001～006`、`FR-UI-001～004`、
  `NFR-SAFE-001`、`NFR-SEC-001～002`、`NFR-PRIV-001`、`NFR-ACC-001`、
  `NFR-PERF-001～002`
- 前序 Gate：[S3 浏览/缩略图 Backend Ready](s3-browse-thumbnail-backend-ready.md)、
  [S4 原媒体内容 Backend Ready](s4-media-content-backend-ready.md)
- 权威契约：`api/openapi.yaml`
- 浏览 feature owner：`web/src/features/browse`
- API adapter owner：`web/src/lib/api/catalog.ts`
- 集合 owner：`web/src/components/patterns/MediaCollection`
- 预览 owner：`web/src/components/patterns/MediaPreview`
- URL、Query key、偏好和状态语义分别保持唯一 owner，不复制到页面组件。

## 验收判断

| 判断项 | 证据 | 结论 |
| --- | --- | --- |
| 真实纵向成功链 | 一次性 SQLite、只读两层合成 `/library` 和真实 Go 进程执行 setup → 建库/扫描 → 目录浏览 → ready WebP → 原内容预览 | 通过 |
| 导航与 URL | 固定桌面目录侧栏、移动抽屉、面包屑、direct/recursive、来源目录链接、sort 与浏览器前进/后退/刷新恢复 | 通过 |
| 集合与分页 | 50 项 keyset cursor、DOM 顺序、grid/masonry 记忆、TanStack Virtual；分页错误保留项目并停止自动重试，显式重试成功 | 通过 |
| 状态矩阵 | 同一生产页面覆盖 skeleton、pending→failed、首屏错误/重试、分页错误/重试、empty、offline preserved index | 通过 |
| 非模态预览 | 桌面右侧停靠、窄屏内容流、图片/原生视频、基本信息、前后项、调宽、父列表继续可用 | 通过 |
| 固定交互 | 未固定单击跟随；固定后单击只选择、双击切换；取消固定跟随；Escape/关闭恢复虚拟项焦点 | 通过 |
| 资源与容量 | 100k 主档有界 DOM、顺序 cursor、4 页 pending 刷新上限、单活动媒体和 12 帧焦点恢复预算 | 通过 |
| 安全与隐私 | 浏览/预览只使用 opaque ID、library-relative 路径和同源内容 URL；不显示 host path，不写原媒体 | 通过 |
| 响应式与主题 | 390/1024/1280px 浏览链无页面横向溢出；稳定浅色/深色主题均通过 | 通过 |
| 键盘与可访问性 | DOM/语义顺序、pressed 状态、状态文字、移动抽屉焦点、Escape 和关闭焦点恢复；Chromium axe serious/critical 为零 | 通过 |
| 设计一致性 | `web/design-qa.md` 的 S3-101～106 原型同状态对照最终结果均为 passed；S3-107 不新增产品视觉 | 通过 |
| CI 固化 | `Authentication, library, and browse E2E` 使用锁定 Chromium、一次性真实后端及受控契约 fixture 执行 `make test-web-e2e` | 通过 |

## 自动与设计证据

- `web/tests/e2e/auth.spec.ts`
- `tests/e2e/web_auth.sh`
- `web/src/components/patterns/MediaCollection/*.test.tsx`
- `web/src/components/patterns/MediaPreview/*.test.tsx`
- `web/src/features/browse/**/*.test.tsx`
- [S3-101 目录导航](s3-frontend-directory-navigation.md)
- [S3-102 浏览范围](s3-frontend-browse-scope.md)
- [S3-103 媒体集合](s3-frontend-media-collection.md)
- [S3-104 浏览状态](s3-frontend-browse-states.md)
- [S3-105 媒体预览](s3-frontend-media-preview.md)
- [S3-106 固定预览](s3-frontend-pinned-preview.md)
- [S3-107 容量预算](s3-frontend-capacity.md)
- `web/design-qa.md` 与 `web/qa/s3-*`
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

- Stage 4 仍负责搜索 UI、可直达完整查看器、fit/zoom/pan/1:1/fullscreen，以及
  Range/codec/离线/删除状态的产品整合。
- 当前真实浏览器成功链使用 JPEG 图片。原生 video DOM/资源释放已有组件证据，但真实
  MP4/MOV/MKV 播放与 Range 浏览器矩阵仍由 S4-007～009 和 Stage 5 固定。
- 100k 前端证据使用组件工作台合成主档；后端 100k/10k 已有独立容量 Gate。受限
  四核/4 GiB 下的完整 100k HTTP UI 遍历、最终 FPS/RSS 和代表性存储仍属于 Stage 5。
- Firefox/Safari、最终 Chrome 版本、可信代理、非回环网络和发布级视觉回归仍属 Stage 5。

## 交接

- 后端：浏览、缩略图和原媒体内容 Backend Ready。
- 前端：Stage 3 浏览与非模态预览 Integrated Done。
- 允许的下一步：`S4-004` 搜索输入、filter、结果列表、URL 状态和无结果恢复。
- 评审日期：2026-07-28。
