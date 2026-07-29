# WCH-S3 Consumer/UI

## 判断

- 日期：2026-07-29
- 目标版本：`POST-MVP-2` revision 2
- 需求：`FR-SCN-014`、`WCH-AC-001`、`WCH-AC-011～012`
- Owner：`web/src/features/browse`、`web/src/features/search`
- 当前判断：**Conditional Go**

产品负责人明确选择“浏览/搜索页提供刷新按钮，进入或切换目录时重新获取”，不使用 catalog
revision 轮询、SSE 或 WebSocket。该消费者 UI 获准在本地 Linux/arm64 组合上实现与验证；
原生 Linux/amd64 仍是 WCH-S2 和最终发布阻塞。

## 已实现合同

- 浏览工具栏提供中英文“刷新当前目录”，搜索命令区提供“刷新搜索结果”。
- 目录路由发生变化时，重新获取目标目录详情、直接子目录、媒体首屏和媒体库详情。
- 显式刷新和目录返回会裁掉已载入的后续 cursor 页，只保留第一页锚点并建立新链。
- URL 中的目录、递归、类型、排序和搜索筛选不变；布局偏好不变。
- 刷新使用共享 Button 与唯一 query-key owner；没有新增定时 revision 请求。

## 已执行证据

```text
npm test -- --run
30 files / 116 tests passed

npm run build
passed（仅保留既有 >500 kB chunk warning）

npm run check:architecture
passed

make fmt
passed

make arch-check
passed

make generate-check
passed

make lint
passed

make test
passed
```

定向回归覆盖：

- 浏览刷新按钮重新请求当前 directory/assets/library；
- 搜索刷新按钮重新请求当前查询；
- 目录导航返回 15 秒 staleTime 内的已缓存目录时仍重新请求；
- 多页 infinite query 刷新前裁掉旧 cursor 后续页。

## 未解除的阻塞

- 原生 Linux/amd64 自动发现与容量矩阵仍未执行，WCH-S2 保持发布 No-Go。
- WCH-S4 仍需真实容器中文/英文、键盘、焦点、滚动锚点与新增文件端到端验证。
