# 智能功能设置：方案一交互原型

POST-MVP-5 / ADR-0019 / CR-2026-030；FR-INT-017/019/020。用户于 2026-09-14
选择第一张设计图。Owner：前端 AI 设置；仅限可丢弃原型，不进入生产 import graph。

复用 `web/src/components` 的 ManagementShell、Button、Switch、Select、Dialog、
InlineStatus、主题/语言及中央 tokens。未更改共享组件，也未添加生产路由或 API。
模型对应关系修正为 CLIP → 智能搜索、buffalo_l → 人物分析；开关明确作用于当前媒体库。

```sh
npm install --prefer-offline --no-audit --no-fund
npm run dev -- --port 6010 --strictPort
npm run build
```

必须在本仓库中运行：原型直接引用规范组件，不能单独复制目录运行。
默认仅监听 127.0.0.1；没有部署到远程或操作 server222。

可操作：按媒体库切换能力、查看库详情、查看/停止/继续示例任务、展开部署信息、
模拟服务离线/未配置、中英切换、深浅色主题。状态仅保存在 React 内存，刷新重置；
主题/语言复用现有本地偏好机制。没有读取 Docker、模型、SQLite 或媒体，没有发送推理请求。
全局搜索/其他管理导航是复用壳层展示，不实现这些跨页面流程。

`evidence/selected-design.png` 是用户选择的原始图；该图的双模型混用文本不是模型合同。
`design-qa.md` 记录视觉对照和真实浏览器操作。原型不证明实际分析、模型激活或发行就绪。
