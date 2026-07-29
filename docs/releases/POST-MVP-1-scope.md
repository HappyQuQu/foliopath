# POST-MVP-1 Scope Manifest — Revision 1

## 冻结记录

- 版本：`POST-MVP-1`
- 产品显示标识：`Post-MVP/1`
- Scope revision：`1`
- 状态：`Scope Frozen`
- 冻结日期：2026-07-29
- 基线事件：[CR-2026-004](../changes/CR-2026-004-video-storyboard-preview.md)
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- 前置版本：`MVP-2026-07-23`；本 manifest 不改变其 scope 或 Release Candidate 结论
- Scope-budget exception：N/A；这是独立后续版本，不向冻结 MVP 插入能力

## 版本目标

让用户在大型视频集合中，无需打开播放器即可通过桌面卡片悬停快速了解视频跨时间段内容，
同时保持扫描、poster、浏览、搜索、原视频播放、原件只读和目标设备资源边界可靠。

## 冻结功能需求

- `FR-MED-009`：为时长至少 2 秒且成功探测的受支持视频生成最多 10 帧均匀采样的
  storyboard；它是可重建派生资源，不是真实关键帧、镜头识别或视频转码。
- `FR-MED-010`：storyboard 在 poster 可用后以较低优先级异步处理；pending、failed、
  cancelled、offline 或缓存淘汰不影响 poster、索引和原视频播放。
- `FR-MED-011`：storyboard 绑定 asset identity、source fingerprint、variant 和 transform
  version，使用临时文件与原子发布，并受统一缓存配额、LRU 和安全磁盘余量约束。
- `FR-UI-008`：只在桌面 fine pointer hover 意图成立时播放；移出、隐藏或虚拟回收恢复
  poster；触摸、键盘焦点和 reduced-motion 不自动播放。

详细行为、采样合同、质量属性和所有权由
[FTR-VID-001](../features/video-storyboard-preview.md)固定。若 Contract Ready 根据真实 spike
需要改变帧阈值、cell 尺寸或内部质量参数，只要不改变上述用户结果和验收 ID，可通过同一
feature 的契约记录收敛；改变用户结果或目标版本必须新增 scope revision。

## 冻结非目标

- `POST-MVP-1-NG-001`：场景检测、镜头识别、AI 摘要、人脸或对象识别。
- `POST-MVP-1-NG-002`：视频转码、预览视频、音频提取或自适应流媒体。
- `POST-MVP-1-NG-003`：移动端长按、滑动、自动播放或新的手势。
- `POST-MVP-1-NG-004`：用户可配置帧数、画质、间隔或独立缓存配额。
- `POST-MVP-1-NG-005`：新部署单元、独立 worker、外部队列、Redis 或外部媒体服务。
- `POST-MVP-1-NG-006`：修改、移动、重命名或删除原视频。

## 冻结验收

- `VSP-AC-001～008` 全部适用，定义见
  [feature spec](../features/video-storyboard-preview.md#验收标准)。
- `VSP-S0 Architecture Ready`、`VSP-S1 Contract Ready`、`VSP-S2 Backend Evidence
  Ready`、`VSP-S3 Consumer/UI Ready` 与 `VSP-S4 Integrated Slice Done` 必须依次为 Go。
- 原生 linux/amd64、linux/arm64 使用同一合成 fixture；目标浏览器覆盖
  Chromium、Firefox、WebKit。
- 所有适用的架构、生成、契约、unit/race/integration/E2E、容量、可访问性和文档检查必须
  实际执行；未执行不能以计划替代。

## 继承约束

本版本继承 `MVP-2026-07-23` 的全部产品与安全不变量，尤其是：

- 原媒体只读；
- `/library` 单一锚定媒体根与 mount-crossing 失败关闭；
- filesystem 为媒体存在与层级真相；
- SQLite/任务/缓存是可恢复派生状态；
- 认证业务 API、CSRF 和错误脱敏；
- 单容器模块化单体、现有 capability 依赖方向；
- 有界后台任务、短事务、原子派生发布和统一缓存余量。

本 manifest 冻结范围，不代表 feature 已实现、Gate 已通过或版本已发布。
