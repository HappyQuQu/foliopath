# CR-2026-020：收藏与手动标签

## 状态

- 状态：Confirmed
- 变更等级：C2
- 目标版本：`POST-MVP-4` / `Post-MVP/4`
- Scope：[POST-MVP-4 revision 1](../releases/POST-MVP-4-scope.md)
- Change Record ID：`CR-2026-020`
- 基线事件：2026-08-10 产品用户确认开始实现
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- Capability Owner：`internal/curation`
- 前端 Consumer：`web/src/features/curation` 与共享媒体集合

## 用户问题与价值

用户需要跨真实目录保留喜欢的图片和视频，并用自己输入的标签进行轻量整理，而不移动、
重命名或修改原媒体。收藏提供一次点击的快速归集；标签提供可复用的人工分类。

## 范围

- 新增 `FR-CUR-001～009`、`NFR-CUR-001～003` 与 `CUR-AC-001～015`。
- 包含：收藏切换与分页列表、全局扁平标签、标签创建/改名/删除、单资产标签替换、按标签分页
  浏览、当前库/全部库范围、图片/视频筛选、最近收藏或修改时间排序。
- 不包含：评分、评论、层级标签、自动/AI 标签、相册、原文件 metadata/sidecar 写入、跨实例
  同步、批量选择和批量打标签。
- `MVP-NG-006` 继续描述 MVP 非目标；本能力不回写 MVP 范围。
- Scope-budget exception：N/A；使用新的 `POST-MVP-4`，不改变已冻结 Post-MVP/1～3。

## 标签选择与新建交互

- 当前实例已有的、尚未关联到当前媒体的标签，必须以独立的可选列表或 chip 直接
  选择；用户不需要重新输入已有标签的名称。
- 文本输入框只用于新建标签，使用“新建标签”动作和明确辅助文案；不把它同时伪装成
  已有标签搜索、自动补全或选择控件。
- 直接选择已有标签复用 `GET /api/v1/tags` 的有界分页词表和既有的单资产标签集合
  原子替换；新建仍先调用现有创建合同，再将返回的 tag ID 加入当前集合。无新 API、
  schema 或 migration。词表不得无界一次载入。

## 架构影响

- `internal/curation` 唯一拥有标签规范化、收藏/标签状态转换、revision 和分页绑定。
- `internal/store/sqlite` 提供只追加 migration 与 repository adapter；写入仍通过实例级写门。
- `internal/api` 只解析 DTO、资源 ID、条件更新与统一错误，不直接执行 SQL。
- 收藏和标签只持久化到 `/app/data`；`/library` 保持只读。
- 资产或媒体库可靠删除时由外键级联删除关联。offline、失败、取消或部分扫描继续保留资产，
  因而也保留整理数据。
- 不改变部署单元、核心技术、信任边界或依赖方向；无需新增 ADR。若实现需要改变扫描删除资格、
  资产身份或跨实例写入，必须先新增 ADR。

## 风险与验证

- 新增 `R-023`：大规模标签关联或频繁收藏写入导致分页漂移、查询放大或丢失更新。
- 缓解：复合索引、稳定 keyset、全局 curation revision、单事务状态替换、每资产最多 20 个标签、
  每次列表最多 200 项。
- 验证：migration upgrade/FK、Unicode 名称、重复标签、幂等收藏、revision/cursor 篡改、离线保留、
  可靠删除级联、100k 资产与有界 tag join、HTTP/CSRF、生成 client、已有标签直接选择、
  新建专用输入的可辨识文案、键盘/移动端与虚拟化。
