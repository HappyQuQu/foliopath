# VSP-S3 视频故事板 Consumer/UI Ready

## 结论

**Go — VSP-S3 Consumer/UI Ready。**

`FTR-VID-001` 已通过生成 client、唯一媒体可用性 adapter 和共享 `MediaCollection`
连接真实 storyboard 契约。浏览与搜索现可进入 `VSP-301～304` 纵向集成与版本交付；
本 Gate 不代表完整 feature 已发布。

## 已接受实现

- `api/openapi.yaml` 将首版 ready `frameCount` 收紧为已冻结的 `4 | 10`，生成
  TypeScript client 重新生成且无手写 wire type。
- `web/src/lib/media/availability.ts` 唯一验证 ready layout，并把 thumbnail 或
  storyboard pending 交给现有有界 Query 刷新策略；browse/search 不解释派生状态。
- 共享 `MediaCollection` 唯一拥有 300 ms hover intent、sprite decode、500 ms/frame、
  循环、同页单活动项及 leave/hidden/recycle/unmount 清理。
- sprite 使用服务端 columns/rows/cell 尺寸定位；横/竖 cell 在 grid 和 masonry 中按
  cover 轴裁切，不拉伸源方向。
- touch/粗指针、键盘焦点和 reduced-motion 不启动；decode failure、pending 与 failed
  静默保留 poster，不新增 live region 或“关键帧/AI”文案。

## 组件与交互证据

`MediaCollection.test.tsx` 覆盖：

- 300 ms 前不创建 decode 请求，成功 decode 后才覆盖 poster；
- 4/10 帧服务端布局、500 ms 递进、末帧循环和横/竖 cover；
- 快速切卡只保留一个活动项，leave、页面隐藏、组件卸载均清理；
- decode failure 恢复 poster；
- fine pointer、touch/粗指针、reduced-motion 与键盘焦点分支；
- 100 个 ready 视频快速掠过时仅一个 decode/活动 interval，卸载后 timer 为零；
- 现有 100k 虚拟 DOM、分页、单击/双击、选择和焦点恢复回归保持通过。

组件工作台新增 ready 4/10、pending/failed poster fallback、横/竖、grid/masonry、长文件名
和 100-video capacity stories；全局浅/深主题与 a11y error Gate 继续适用。

## 浏览器与容量证据

2026-07-29 的 focused product E2E 在 Chromium、Firefox、WebKit、Chrome stable、
Chrome forced-colors 和 Pixel 5 touch profile 全部成功：

- 200 ms 内无 storyboard 请求；
- 意图成立后才加载并进入 playing；
- 移出恢复 poster 且不继续请求；
- reduced-motion 与 touch profile 请求数为零。

`make test-storyboard-browser-capacity` 在 1280×5000 视口挂载并快速掠过 100 个 ready
视频：

| 浏览器 | intent 前/后活动数 | leave 后 | FPS | Peak RSS |
| --- | ---: | ---: | ---: | ---: |
| Chromium | 0 / 1 | 0 | 60.001 | 582,811,648 B |
| Firefox | 0 / 1 | 0 | 60.080 | 1,340,080,128 B |
| WebKit | 0 / 1 | 0 | 60.113 | 115,933,184 B |

三端均挂载 100 项，满足单活动项、≥45 FPS 和 ≤1.5 GiB 进程树 RSS 预算。
`make test-browser-capacity` 也重新通过原 100k 集合回归：三端各挂载 60 项，
FPS 57.396～60.002、P95 frame interval 16.7～22 ms，无 long frame。

## 实际执行的验证

本 Gate 前已成功执行：

```text
npm --prefix web run check
npm --prefix web run test:e2e -- --project=chromium --grep "storyboard hover"
npm --prefix web run test:e2e -- --project=firefox --project=webkit \
  --project=mobile-chromium --grep "storyboard hover"
npm --prefix web run test:e2e -- --project=chrome-stable \
  --project=chrome-forced-colors --grep "storyboard hover"
make test-storyboard-browser-capacity
make test-browser-capacity
```

`npm run check` 包含生成漂移、前端架构、TypeScript、111 个 Vitest 测试和 Storybook
production build；全部成功。

## 残余边界与下一授权

- `VSP-301` 必须把真实扫描、poster、storyboard ready 与浏览/搜索 hover、预览和查看器
  贯通；当前浏览器请求时序测试使用冻结 wire fixture，不能替代真实后端纵向链。
- `VSP-302` 仍需在最终目标 linux/amd64 与 linux/arm64 候选上复验完整前后端组合。
- 本 Gate 不改变 MVP Release Candidate，也不授权把 feature 写入稳定版发布说明。
