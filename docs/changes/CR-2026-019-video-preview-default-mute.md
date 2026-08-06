# CR-2026-019：视频预览默认静音偏好

## 状态

- 状态：Confirmed / Implemented
- 变更等级：C1
- 目标版本：POST-MVP-1 revision 9
- Change Record ID：CR-2026-019
- 提出日期：2026-08-06
- 产品负责人：产品用户（本次明确提出）
- 架构负责人：FolioPath maintainers
- Capability Owner：`web/src/lib/storage/preferences.ts`（偏好语义）、共享
  `MediaPreview` / `useMediaPreviewController`（播放消费）

## 用户问题与价值

视频预览的静音状态此前与自动播放绑定，用户无法独立决定新打开的预览默认是否有声音。
自动播放和声音是两个不同偏好，应能分别配置。

## 范围

- 新增需求：`FR-MED-015`。
- 通用设置新增“视频预览默认静音”，默认开启，并保存于既有当前浏览器偏好命名空间。
- 自动播放与默认静音独立：关闭自动播放不改变静音；关闭静音也不关闭自动播放请求。
- 自动播放开启且默认静音关闭时，浏览器可能按自身策略阻止带声音自动播放；FolioPath 保留
  原生 controls，用户可手动开始，不伪造播放成功。
- 只影响浏览和搜索共享的非模态视频预览；不改变完整查看器、故事板悬停动画、原视频、
  服务器设置、音量记忆或跨设备同步。
- Scope-budget exception：用户明确接受的独立 Post-MVP/1 小切片；无 API、数据库、媒体处理、
  部署或信任边界变化。

## 架构与合同

- 偏好的默认、读取和写入只由 `web/src/lib/storage/preferences.ts` 拥有。
- 设置页只编辑草稿并在显式保存时提交；恢复操作回到已保存值。
- `useMediaPreviewController` 在页面组合时分别读取 autoplay 与 muted；Browse/Search 只把
  两个值传给共享 `MediaPreview`，不得自行实现声音策略。
- `MediaPreview` 的 `autoPlay` 和 `muted` 属性分别由对应偏好控制。
- 不需要 ADR、OpenAPI、migration 或生成文件变化。

## 证据

- 偏好单测固定默认静音与显式带声音值。
- 设置页单测固定两个独立开关一起保存。
- controller 与共享预览单测固定 autoplay/muted 可独立表达。
- Browse/Search、TypeScript、前端架构与生成一致性回归。
