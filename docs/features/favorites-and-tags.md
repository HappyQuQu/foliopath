# FTR-CUR-001：收藏与手动标签

## 文档状态

- Feature ID：`FTR-CUR-001`
- 状态：[CUR-S4 Integrated Slice Current](../gates/POST-MVP-4/cur-s4-integrated-slice-current.md) Conditional Go
- 目标版本：`POST-MVP-4`
- 交付切片：`CUR`
- Change Record：[CR-2026-020](../changes/CR-2026-020-favorites-and-tags.md)
- Scope：[POST-MVP-4 revision 1](../releases/POST-MVP-4-scope.md)
- Capability Owner：`internal/curation`
- Store Adapter：`internal/store/sqlite`
- HTTP Adapter：`internal/api`
- Web Consumer：`web/src/features/curation`

## 用户结果

用户可以从卡片、预览或查看器收藏媒体，在统一媒体工作台中跨目录找回它；也可以给媒体添加
自己命名的标签，并从标签入口浏览对应媒体。目录、收藏和标签只切换左侧视角与中间集合，点击
媒体继续在右侧显示详情与整理操作。所有操作只改变 FolioPath 数据，不改变原媒体。
已有标签在整理区可直接选择；文本输入仅用于新建标签，不要求用户重复输入已有名称。

## 需求

| ID | 需求 |
| --- | --- |
| `FR-CUR-001` | 用户必须能对任一已索引资产幂等添加或取消收藏，并在卡片、预览和查看器看到一致状态。 |
| `FR-CUR-002` | 收藏页必须支持全部媒体库或单个媒体库、图片/视频筛选、最近收藏和修改时间排序，以及稳定游标分页。 |
| `FR-CUR-003` | 用户必须能创建实例级扁平标签；规范化后名称唯一，支持简体中文、英文和内部空格。 |
| `FR-CUR-004` | 用户必须能改名和删除标签；删除标签只删除应用数据及关联，不改变媒体。 |
| `FR-CUR-005` | 用户必须能用一次原子操作替换单个资产的标签集合，每资产最多 20 个标签。 |
| `FR-CUR-006` | 标签资产页必须支持全部媒体库或单库范围、媒体类型、修改时间排序和稳定游标分页。 |
| `FR-CUR-007` | 快速访问必须位于目录树之前，收藏与标签在视觉、路由和可访问名称上不得伪装成真实目录。 |
| `FR-CUR-008` | 收藏与标签必须沿用目录浏览的单一媒体工作台；媒体库切换器与快速访问保持可见。收藏视角隐藏目录树并在收藏项旁显示总数；标签视角隐藏目录树并在原位展示带数量与数量级字号的标签墙；单击媒体在右侧打开共享非模态预览并可编辑收藏和标签，只有显式操作才进入完整查看器。 |
| `FR-CUR-009` | 单资产标签编辑必须把已有标签作为独立可选项直接添加；文本输入只用于新建标签，不兼任已有标签搜索或选择控件。 |
| `NFR-CUR-001` | 收藏、标签、关联、查询和 cursor 必须在 100k 资产目标档保持有界，不得无界返回或渲染。 |
| `NFR-CUR-002` | 整理数据只写 `/app/data`；原媒体和 `/library` 保持只读，API 不接受或返回宿主机路径。 |
| `NFR-CUR-003` | 任一变更递增 curation revision；旧 cursor 不得与新状态拼接，冲突或篡改必须失败关闭。 |

## 合同摘要

- `GET /api/v1/favorites`
- `PUT /api/v1/assets/{assetId}/favorite`
- `GET /api/v1/tags`
- `POST /api/v1/tags`
- `PATCH /api/v1/tags/{tagId}`
- `DELETE /api/v1/tags/{tagId}`
- `GET /api/v1/tags/{tagId}/assets`
- `GET /api/v1/assets/{assetId}/curation`
- `PUT /api/v1/assets/{assetId}/tags`

所有写操作需要认证与现有 CSRF/同源保护。列表 `limit` 默认 50、最大 200；cursor 绑定规范
scope、kind、sort/order 与 curation revision。标签名先 NFC、去除首尾 Unicode 空白并压缩内部
连续 Unicode 空白，再用 Unicode case-fold 形成唯一键；显示名 1～32 个 Unicode code points，
拒绝控制字符。创建同名标签返回既有标签，改名冲突返回 `tag_name_conflict`。

## 数据与删除语义

- `curation_state(singleton_key, revision)`：全局单调 revision。
- `asset_favorites(asset_id, library_id, created_at_ms)`：一资产最多一个收藏记录。
- `tags(id, name, normalized_name, created_at_ms, updated_at_ms)`：实例级词表。
- `asset_tags(asset_id, library_id, tag_id, created_at_ms)`：复合唯一关联。
- 资产外键使用 `ON DELETE CASCADE`。可靠扫描或媒体库删除移除资产时同步删除关联；失败、取消、
  offline 或部分不可读扫描不清理可靠资产，因此整理数据保留。
- 无引用标签不会自动删除；用户可显式删除。

## 验收

- `CUR-AC-001` 收藏写入幂等，首次时间不因重复收藏改变。
- `CUR-AC-002` 取消收藏幂等且不会删除媒体或派生文件。
- `CUR-AC-003` 收藏列表按稳定 tuple 分页，状态改变使旧 cursor 返回 `invalid_cursor`。
- `CUR-AC-004` Unicode 等价或大小写等价标签不能重复。
- `CUR-AC-005` 标签改名冲突安全失败，删除只级联关联。
- `CUR-AC-006` 标签集合替换是单事务全成或全不成，并执行 20 项上限。
- `CUR-AC-007` 不存在或跨资源资产/标签使用统一 not-found，不泄露归属。
- `CUR-AC-008` offline 保留收藏与标签并显示来源不可用；可靠删除级联关联。
- `CUR-AC-009` 所有页面保持 cursor 分页和虚拟化，不一次加载完整集合。
- `CUR-AC-010` 写 API 受会话与 CSRF 保护，错误不含 SQL、绝对路径或堆栈。
- `CUR-AC-011` 简中/英文、键盘、焦点、触控目标和 390～1440px 布局通过。
- `CUR-AC-012` 原媒体 hash、mtime、名称和路径在完整纵向链前后不变。
- `CUR-AC-013` 从目录切换到收藏或标签不离开媒体工作台，也不替换媒体库切换器和快速访问；两者均隐藏目录树，收藏只补充总数，标签在原区域展示标签墙。网格选择、右侧预览、固定、前后项、完整查看器和焦点恢复遵循目录浏览的同一交互规则。
- `CUR-AC-014` 标签墙显示每个标签的媒体数量，并映射到 5 个稳定字号等级；数量越多字号越大，数量徽标仍须可读可访问。
- `CUR-AC-015` 当前媒体未使用的已有标签以可辨识、可键盘操作的列表或 chip
  直接选择，不需重新输入名称；“新建标签”输入和动作与已有标签区域有明确可见、
  语义和焦点区分。已有词表保持有界分页，不一次载入全部标签。
