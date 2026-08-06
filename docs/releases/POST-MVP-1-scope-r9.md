# POST-MVP-1 Scope Manifest — Revision 9

Revision 9 完整继承 [revision 8](POST-MVP-1-scope-r8.md)及其全部安全、非目标和验收约束，
并通过 [CR-2026-019](../changes/CR-2026-019-video-preview-default-mute.md)追加
`FR-MED-015`：

- 通用设置允许独立配置视频预览是否默认静音；
- 默认保持静音，以延续稳定的浏览器自动播放体验；
- autoplay 与 muted 两个偏好互不改写；
- 带声音自动播放是否获准仍由浏览器策略决定，失败时保留原生手动播放；
- 只影响浏览与搜索共享的非模态视频预览，不改变服务器或媒体内容。

- 版本：`POST-MVP-1`
- Scope revision：`9`
- 状态：`Scope Frozen`
- 冻结日期：2026-08-06
- 产品负责人：产品用户
- Scope-budget exception：用户明确接受的既有前端偏好扩展；不扩大媒体或部署边界
