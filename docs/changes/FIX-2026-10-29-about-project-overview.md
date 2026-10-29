# 关于页：项目概览与使用说明

2026-10-29 用户要求“关于里面多加点东西”。目标Post-MVP后续维护版；FR-UI-001～003/006、FR-OPS-001～006及CR-2026-015既有关于/版本能力；Owner release-info/AboutPage与共享BrandMark/InlineStatus等现有原语。当前分支叠加此前未发布的无缝沉浸浏览，不新增API、服务端配置或部署能力。

- 内容：介绍只读目录/压缩图集浏览、本地AI可选、收藏/手动标签；列明已实现格式、归档限制、单管理员和浏览器使用方式，提供固定官方文档/源码/反馈/发行镜像/许可证链接。文案只描述实际功能，AI卡不冒充当前部署已启用。
- 合同：静态内容依据已接受产品要求与README；版本/更新历史仍消费既有SystemStatus/ReleaseInformation生成client和query owner，不增加系统诊断、宿主路径、协议身份或后台操作。外部链接为普通导航，非更新/升级按钮；更新记录正文仍在页内展示。
- 顺带修复：首次加载、读取失败或没有稳定版本数据时不显示“最新”；手动检查失败显示未能检查并保留已有记录。只有成功取得最新版本数据才显示未发现更新，不在前端复制semver比较策略或自动更新容器。
- 布局：保持管理中心和现有全宽容器，宽屏概览与版本并排、四项能力网格，支持说明与项目入口响应式分列；手机顺序保持标题→版本→功能→说明→入口→更新记录。复用tokens，不新增共享primitive或第二套主题。
- Backend Ready沿用既有只读版本合同；先确认本记录内容映射，再实现生产AboutPage与合成Story/回归。普通测试、截图与产物依约本地专用，不扩大25项发行白名单。


## 本地验证

- `npm --prefix web run test -- src/features/release-info/pages/AboutPage.test.tsx`：6项通过，覆盖中英内容/链接、首次等待、读取失败、手动刷新失败保留历史及重试、暂无版本。
- `npm --prefix web run check`：82个文件、559项测试全部通过；包含依赖高危审计、生成一致性、前端架构、视觉引用清单、TypeScript及生产组件Storybook构建。日志`/tmp/foliopath-about-check.log`。
- `make arch-check`、`git diff --check`通过，日志`/tmp/foliopath-about-arch.log`。未运行本轮无变更的Go业务测试、完整集成或E2E命令；实际浏览器验证见下。
- Docker生产arm64构建通过：`foliopath:local-about-overview-20261029`，本地镜像`sha256:a8603e186892710acf8b0e5043fb8958201645133fa511b37be11fde74fe2538`。仅替换现有6020合成库预览容器，保留单一只读媒体mount与原Linux数据volume；健康检查和容器内`database-check`均通过。未构建amd64或执行正式发行验证。
- 实际IAB检查390、768、1440、1920px：均无横向溢出；四项能力在桌面四列、平板两列、窄手机一列；手机版格式说明和五个项目入口可完整阅读。英文深色1440px检查通过，文档入口随语言指向相应README，固定外链均为新页且带`noopener noreferrer`。
- 首次实际读取更新信息失败，版本卡正确显示“暂时无法检查更新”，静态介绍仍可用；真实点击检查更新进入等待，随后成功恢复“未发现可用更新”和真实v1.9.0等页内更新记录。暂无版本/失败保留历史重试状态另由生产组件测试与标注的合成Story覆盖。
- 截图及响应式DOM几何记录：`/tmp/foliopath-about-preview/evidence/`；构建日志同级`image-build.log`。已恢复中文浅色、默认viewport并保留关于页供查看。

状态：本地实现与预览完成，尚未提交、推送或发布。222部署不变；本轮不修复已单独记录的扫描回退循环。
