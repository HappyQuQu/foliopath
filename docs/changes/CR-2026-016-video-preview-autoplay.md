# CR-2026-016：视频预览自动播放偏好

## 状态

- 状态：Confirmed / Implemented
- 变更等级：C1
- 目标版本：POST-MVP-1 revision 6
- Change Record ID：CR-2026-016
- 提出日期：2026-08-04
- 产品负责人：产品用户（本次明确提出）
- 架构负责人：FolioPath maintainers
- Capability Owner：`web/src/lib/storage/preferences.ts`（偏好语义）、共享
  `MediaPreview` / `useMediaPreviewController`（播放消费）

## 用户问题与价值

用户打开视频预览后希望立即看到内容，不必再次点击原生播放按钮；同时仍需允许不希望自动
播放的用户关闭该行为。

## 范围

- 新增需求：`FR-MED-014`。
- 通用设置新增“自动播放视频预览”，默认开启，保存于既有管理员浏览偏好命名空间。
- 开启时，浏览和搜索共享的非模态视频预览静音自动播放并保留原生控件；静音是浏览器稳定
  允许自动播放的兼容边界，用户可通过控件取消静音。
- 关闭时，不设置自动播放或静音，视频由用户手动启动。
- 不包含：完整查看器自动播放、后台音频、跨媒体连续播放、播放队列、记忆音量或改变故事板
  hover/reduced-motion 策略。
- Scope-budget exception：用户明确接受的独立 Post-MVP/1 小切片；无 API、数据库、媒体处理、
  部署或信任边界变化。

## 架构与合同

- 偏好的默认、读取和写入只由 `web/src/lib/storage/preferences.ts` 拥有。
- 设置页只编辑草稿并在显式保存时提交；恢复操作回到已保存值。
- `useMediaPreviewController` 在页面组合时读取一次偏好，Browse/Search 只传给共享
  `MediaPreview`，不得各自实现播放规则。
- 原内容仍经现有认证、Range 和 admission 合同读取；原媒体保持只读。
- 不需要 ADR、OpenAPI、migration 或生成文件变化。

## 证据

- 偏好单测：缺省为 `true`，显式关闭后保持 `false`。
- 设置页单测：开关默认开启，保存关闭值进入既有偏好对象。
- 共享预览单测：开启时 video 为 autoplay + muted；关闭时两者均不启用。
- Browse/Search 回归、TypeScript、前端架构与生成一致性检查。
