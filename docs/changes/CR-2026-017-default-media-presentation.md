# CR-2026-017：默认排序与媒体布局配置

## 状态

- 状态：Confirmed / Implemented
- 变更等级：C1
- 目标版本：POST-MVP-1 revision 7
- Change Record ID：CR-2026-017
- 提出日期：2026-08-04
- 产品负责人：产品用户（本次明确提出）
- 架构负责人：FolioPath maintainers
- Capability Owner：`web/src/lib/storage/preferences.ts`（偏好）、Browse/Search URL codec
  （可导航排序状态）、共享 `MediaCollection`（布局）

## 用户问题与价值

管理员希望常用的排序与网格/瀑布流布局成为新打开媒体视图的默认值，而不是每次进入页面后
重复切换。

## 范围与语义

- 新增 `FR-BRW-013`。
- 通用设置保存一个默认排序：按当前视图默认，或文件名、修改时间、文件大小的正序/倒序。
- 既有默认布局配置继续支持网格与瀑布流，并明确同时作用于新打开的浏览和搜索。
- 按视图默认保留现有语义：直接目录按名称正序，递归/目录筛选/搜索按修改时间倒序。
- URL 中显式 `sort/order` 始终优先。偏好改变无排序 URL 的初始状态后，页面把结果规范化为
  显式 URL，复制到另一设备仍得到相同顺序。
- 浏览页现场切换布局继续即时记忆；排序现场切换继续写入 URL，不静默改写全局偏好。
- 不包含：每库偏好、每目录偏好、列数/卡片尺寸、服务端设置、API 或数据库变化。
- Scope-budget exception：用户明确接受的独立 Post-MVP/1 小切片；无后端、媒体处理、部署或
  信任边界扩张。

## 证据

- 偏好测试覆盖 contextual 默认、排序写入与保留其他设置。
- Browse/Search URL codec 测试覆盖偏好默认、显式 URL 优先和规范化 URL。
- 通用设置测试覆盖默认排序与瀑布流一起保存。
- Browse/Search 页面、完整前端类型/架构/生成检查保持通过。
