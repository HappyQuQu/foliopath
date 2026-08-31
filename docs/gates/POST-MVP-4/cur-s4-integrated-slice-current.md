# Gate POST-MVP-4 / CUR-S4 / Integrated Slice Current

- 日期：2026-08-10
- 结论：**Conditional Go — 代码与本地纵切完成，发布证据仍阻断**
- 前序：[CUR-S3 Consumer UI Ready](cur-s3-consumer-ui-ready.md)
- Feature：[FTR-CUR-001](../../features/favorites-and-tags.md)

## 已完成纵切

- OpenAPI、migration、domain、SQLite adapter、HTTP adapter、composition、生成 TypeScript client、
  domain adapter、TanStack Query 和生产路由已经形成一条真实链。
- composition integration 覆盖 setup/auth、401、CSRF、收藏/标签持久化、分页读取与原媒体
  hash/mtime 不变。
- 普通 catalog projection 一次返回 `favorite`，没有为可见卡片引入 N+1 curation 请求。
- `make fmt`、`make arch-check`、`make generate-check`、`make lint`、`make test`、
  `make test-integration` 和 `make openapi-lint` 均通过；OpenAPI 只有两个既有 health 4xx warning。

## 发布阻断

2026-08-28 后续维护已解除两个本地阻断：`make test-e2e` 在真实 Docker 应用启动/重启与只读媒体
哨兵下通过；依赖补丁升级后 `make web-check` 的 high-severity audit、生成/架构/视觉引用/类型、
47 个测试文件/167 个测试和 Storybook 构建全部通过。修复记录见
[Web 依赖高危 advisory 修复](../../changes/FIX-2026-08-28-web-dependency-advisories.md)。

同日 `make test-web-e2e` 的真实后端 Chromium / Pixel 5 纵向链也已恢复并通过（7 passed、4 skipped），
包括媒体 hash/path 前后不变；测试镜像不再依赖额外 `alpine/socat` 拉取，当前管理中心、配置 tab 与
媒体库原位状态交互断言已同步。详见
[真实浏览器 E2E harness 与当前界面合同同步](../../changes/FIX-2026-08-28-browser-e2e-harness.md)。

当前剩余发布阻断：尚未取得目标浏览器/真实触摸设备上的 390～1440px 人工视觉与交互签署。

以上阻断解除前不得把 POST-MVP-4 标记为 Release Ready；代码可继续接受评审和本地验证。
