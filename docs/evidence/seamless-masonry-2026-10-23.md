# 无缝瀑布流验证（2026-10-23）

关联 [CR-2026-033](../changes/CR-2026-033-seamless-masonry.md)、`FR-BRW-014`。

环境：macOS arm64、Node 22.22.2 / npm 10.9.7（仓库锁定版本）、本机 Docker Linux 引擎。

## 已执行并通过

- `npm ci --prefix web`、`npm run build --prefix web`。
- `npm run check --prefix web`：高危依赖审计、生成一致性、前端架构、视觉引用、类型、
  47 个文件 / 172 个测试，以及 Storybook 生产构建。
- `make fmt`、`make arch-check`、`make generate-check`、`make lint`、`make test`、
  `make test-integration`、`make test-e2e` 全部成功。容器烟测验证真实二进制启动、认证、
  持久化、只读媒体哨兵及正常停止。
- `FOLIOPATH_WEB_E2E_URL=http://127.0.0.1:4274 npm run test:e2e --prefix web --
  --project=chromium --project=firefox --project=webkit --grep 'seamless masonry'`：
  48/48 通过（2 locales × 2 themes × 4 widths × 3 engines）。
  每项检查横纵间距（容差 0.08 CSS px）、无圆角/文字栏、卡片与图片同高、有界 DOM、
  焦点、无横向溢出、滚动、刷新记忆、切回网格。全部使用合成 catalog 与专用图片 fixture。
- 最后的小数高度兼容性与设置描述调整后，生产构建及受影响组件/设置的 24 项测试再次通过。

## 过程与边界

首次 Go / 容器检查因新 worktree 未生成 embed 资源失败；生产构建生成资源后 Go 检查通过。
容器烟测还暴露根 Docker ignore 排除前端产物的问题，专用
`tests/e2e/Dockerfile.dockerignore` 修复后重跑成功。更新依赖后重新启动 Vite，完整浏览器矩阵
重跑通过；不把这些已解决的首次失败记为通过。

初次依赖审计存在 11 high / 1 critical。兼容更新锁定的 Router、Vite、Storybook、
js-yaml、brace-expansion 及受影响传递依赖后，`audit:high` 无适用 high/critical，未增加
豁免。npm 仍报告 2 项 moderate（Vitest / mocker 开发测试服务）；没有绕过 peer 或 audit
失败。生产/Storybook 构建有现存的大 bundle 提醒，构建成功。本轮不宣称新增 100k 容量性能证据，
也不把 macOS 测试等同于原生 Linux 双架构验证；正式双架构构建由既有发布工作流执行。

## 四档截图

以下是生产浏览页面、明确合成的媒体 fixture，不是用户媒体；原始 Playwright 截图供复核：

- [390px](seamless-masonry-2026-10-23/390.png)
- [768px](seamless-masonry-2026-10-23/768.png)
- [1280px](seamless-masonry-2026-10-23/1280.png)
- [1920px](seamless-masonry-2026-10-23/1920.png)
