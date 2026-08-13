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

1. `make test-e2e` 未运行产品 smoke：本机 Docker daemon 不可连接。
2. `make web-check` 在第一步 dependency audit 失败：`js-yaml`、`brace-expansion`、`nanoid`、
   `undici` 的当前 advisory 达到 high；本切片未改变依赖或 lockfile。其余 web checks 已单独执行
   并通过。
3. 尚未取得目标浏览器/真实触摸设备上的 390～1440px 人工视觉与交互签署。

以上阻断解除前不得把 POST-MVP-4 标记为 Release Ready；代码可继续接受评审和本地验证。
