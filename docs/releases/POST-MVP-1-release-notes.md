# POST-MVP-1 发布说明草案

> **DRAFT / NOT RELEASED**  
> FTR-VID-001 尚未通过 VSP-302 目标平台复验、VSP-303 文档签署和
> VSP-304 Integrated Slice Done。本文件不能作为稳定版本发布声明。

## 候选新增能力

视频卡片现在可以使用可重建的 WebP storyboard 快速预览视频时间轴内容：

- 成功探测且时长至少 2 秒的视频生成 4 或 10 帧均匀采样 sprite；
- poster 先生成，storyboard 使用更低优先级的有界后台任务；
- 桌面 fine pointer 停留 300ms 后按 500ms/帧播放，同一页面最多一个活动动画；
- 移出、页面隐藏、虚拟回收或卸载会停止播放并恢复 poster；
- 触摸、键盘焦点、reduced-motion、pending、failed、offline 或 decode failure
  都保留现有 poster、预览、查看器和原视频播放行为。

这里的“关键帧”是面向快速浏览的均匀时间采样，不是镜头检测、编码 I-frame 提取或 AI 摘要。

## 数据、缓存与升级

- 只向前 migration 11 为 `thumbnails` 与 `media_jobs` 增加 `storyboard` variant 和布局约束；
- storyboard 绑定 asset、source fingerprint、transform version，源变化后不会复用陈旧缓存；
- 派生文件先写临时文件再原子发布，并参与现有统一缓存配额、LRU 和安全磁盘余量；
- 删除或重建 storyboard 不影响索引、poster 或原视频；原媒体仍以只读方式访问；
- 回滚仍需恢复升级前离线数据备份，不能假设旧镜像理解 migration 11。

## 部署影响

没有新增端口、服务、环境变量、volume 或媒体写权限。继续只允许一个只读 `/library`
挂载和一个可写 `/app/data`。生产镜像需要包含已验证的 FFmpeg runtime；不新增转码、
兼容播放副本或独立 worker。

## 已完成证据

- VSP-S2：真实 FFmpeg、迁移、任务/缓存/故障、认证 API、原件不变与 Linux
  100k/10k、10% 视频容量；
- VSP-S3：生成 client、共享 hover/sprite controller、组件/a11y、输入模式、
  Chromium/Firefox/WebKit 与前端容量；
- VSP-301：生产镜像真实登录、扫描、浏览/搜索 hover、非模态预览焦点恢复和
  cache 200→202→200 repair。

## 发布前仍需完成

- VSP-302：同一源码状态的原生 linux/amd64、linux/arm64 候选结构化证据和成对校验；
- VSP-303：按
  [发布文档与追踪收敛](../gates/POST-MVP-1/vsp-303-documentation-convergence.md)
  复核并签署本草案；
- VSP-304：聚合 `VSP-AC-001～008`，记录 `VSP-S4 Integrated Slice Done`。

当前 GitHub runner 在执行测试前受账户计费/支出上限阻断；本机跨架构模拟不能替代原生
平台结果。版本号、source commit、镜像 digest、升级来源和最终验证链接将在上述 Gate
全部通过后填写。
