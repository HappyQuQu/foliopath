# server222 智能功能前端部署一致性

- 日期：2026-09-09
- 状态：已部署；候选纵向与线上产物一致性验证通过
- 类型：既有切片的部署回归修复；用户授权恢复当前分支并保留扫描修复
- Requirement：FR-INT-015/017/019/020、NFR-REL-001、NFR-SEC-002
- 目标：POST-MVP-5 / S3 消费者与部署维护；不是 S4 Release Go
- Owner：`internal/webassets` / 构建与部署
- Gate：[S3 UI 与模型获取扩展](CR-2026-026-ai-model-download-and-ui.md)

## 原因与证据

9 月 7 日扫描续租修复构建没有重新生成 Vite dist，而是同步本机已存在的 dist 再编译 Go。
Go embed 把不含 AI 页面的旧 SPA 一并嵌入新二进制。随后 `docker tag` 把修复镜像重新指向
`server222-aifeature-9fb5a54-full`，导致名称与内容不再对应。

9 月 9 日直接请求服务器，`index-C2uROwQB.js` 与当时临时构建目录的文件哈希相同：
`0f75aeaaa2e035de133af1299a3cdfc17209c4bd83d2bce2aebdd0c0babdb57d`。
其中没有 `/settings/ai`、`/intelligence` 和 AI 文案。实际镜像为 `34630b4de56d`，
运行版本 `media-lease-writegate-20260907`；不能以健康检查或镜像标签代替前端内容验证。

## 本次措施

- 保留 `media_jobs` 续租写入串行化修复，重新构建当前工作树的 Vite 与 Go。
- 独立镜像版本 `aifeature-9fb5a54-ui-20260909`，不覆盖旧标签；明确源码含未提交改动。
- 正常 Dockerfile 构建首次因 Docker Hub 鉴权 EOF 失败；候选复用服务器上已存在的
  `45a479c39bd7` 媒体 runtime，Go 1.26.5 在原生 amd64 编译，依赖来自生成的 vendor。
  新 dist 在编译前比对 index SHA-256；镜像写入源码/前端哈希 manifest。
  这不是重新建立全量 runtime 供应链或最终双架构发行证据。
- 新增 `scripts/verify-deployed-web.mjs`，逐文件校验构建目录与候选/线上响应，并核对 SPA 路由。
  测试覆盖正确版本、健康但旧 JS、404 和非法校验目标。上线前已确认旧服务会被该检查拒绝。
- 升级前停止单个 FolioPath 服务并备份应用数据、Compose 和镜像 ID；不写 `/library`。
  migration 36 仅追加模型下载历史索引；回滚仍使用旧镜像与升级前数据配对。

## 构建依赖

`npm ci` 新报告的 js-yaml 高危 finding 使用现有 override 的补丁升级 `4.3.1 → 4.3.2`
解决，锁文件由 npm 10.9.7 生成，不添加豁免。依据：
[js-yaml advisory](https://github.com/advisories/GHSA-2883-xcg3-v3hh)。
Vitest 4.1.10 的两个关联 moderate finding 仍保留；尝试 4.1.11 时固定 npm 的 peer resolver
报 `edgesOut` 内部异常，未使用 `--force` 或 `--legacy-peer-deps` 绕过，也未扩大到跨大版本升级。
Vitest 只用于开发测试，不进入最终无 Node 的 runtime。

## 验证记录

- 通过：`make fmt`、`make fmt-check`、`make arch-check`、`make lint`、`make test`、`make test-integration`、
  `GOPROXY=https://goproxy.cn make generate-check`。
- 首次架构测试中 `go list` 子进程失败，随后单独查询和重新执行通过；未隐藏首次失败。
  首次全量测试中视频独立超时用例 deadline exceeded；该用例单独重复 3 次和全量重跑均通过，
  未修改测试超时或生产代码。
- 通过：npm 安装、`audit:high`（0 high/critical）、生成一致性、前端架构、187 项测试、类型与 Vite 构建。
- 通过：`node --test scripts/verify-deployed-web.test.mjs`，3 项。
- 通过：`go test -race ./internal/aimodel/... ./internal/api/... ./internal/store/sqlite/... ./tests/integration/...`；
  SQLite race 全包运行约 471 秒。
- `make test-e2e` 未通过：本机 Docker 已运行，但 smoke 构建下载 Go 模块连续 3 次被
  `proxy.golang.org` EOF 阻断。没有把下面的独立候选验证冒充该目标通过。

## 正式切换结果

- 2026-09-09 19:23:49 +08:00 启动正式容器；19:28:28 复查仍 healthy / ready、restart count 0，
  已超过历史约 110 秒故障窗口。这仅为短期部署观察，不是长时间扫描/容量验收。
- 镜像：`foliopath:server222-aifeature-9fb5a54-ui-20260909`，100,197,241 bytes；
  image ID：`sha256:1bafc5349167a581d5a64dcc3235ba46489d25e103f89098b50495c609d008d2`。
- 源码快照（包含 vendor 与 dist）SHA-256：
  `278ba8711f2167d1b9ee0fd5e5015c130229f4a192227132351bafe473e45158`。
  manifest 明确标注 `9fb5a54-dirty`；本次没有提交或推送 Git。
- `/app/build-manifest.json` 随镜像保存；Compose 改用独立标签，旧标签和镜像均保留。
- 上线前隔离候选与上线后 9090：5 个静态文件和 `/settings/ai`、`/intelligence` 的字节校验通过。
  新 JS `index-CwEt1PPB.js` 的 SHA-256 为
  `2d17b01e473bf1d8186b360a3c48d0a311a129d5433d93981e5bccc72650dec8`。
- 原生 amd64 候选用独立空数据库、合成 PNG、临时测试账户验证 setup/login、未登录 AI API 401、
  已登录下载目录 200、媒体库创建、成功扫描、WebP 缩略图 200 与原图哈希不变；浏览器实际登录
  并打开新版概览和模型下载页。候选随后优雅退出（exit 0），测试容器已移除。
  临时 SSH 转发已关闭，备份校验用的数据库副本已删除；完整回滚归档未删除。
- 正式浏览器入口返回登录页（当前浏览器没有有效会话），没有重置或修改正式管理员凭据；
  正式登录后的页面交互未冒称已验证。正式 `auth/status` 显示已有管理员、需要认证。
- migration `35 → 36` 已完成；升级前后 users=1、libraries=3、tags=2，收藏、标签关联、人物及人工
  关系计数相同。原媒体挂载仍为 `/library:ro`，应用数据挂载保持原位置。

## 备份与回滚定位

服务器受保护目录：`/root/foliopath/backups/pre-ai-ui-20260909.HOF9YQ`。
其中 `data.tar` 为停服后的完整应用数据归档（约 1.2 GiB，含缓存），另有旧 Compose、新 build manifest
和实际候选 Dockerfile；归档已可读，抽出数据库 `PRAGMA quick_check` 为 `ok`、schema 为 35。
归档 SHA-256：`7094c7c6be15c708871513b8b84a93d956b82e8a0a4fe82d13393480ee53f6ed`。
这是同机升级回滚备份，不声称完成异地灾备。

旧镜像 `foliopath:rollback-before-ai-ui-20260909` 对应
`sha256:34630b4de56d618c09b02d67714e57a9474140e2c720cbba4f25982f89182349`。
如需回滚，先停止新容器并另存当前数据，再恢复旧 Compose 与升级前完整归档，不能仅换旧镜像直接打开
schema 36。此次没有执行回滚，也没有删除任何原媒体。

模型源仍未配置，推理 runtime 仍未纳入本次候选；真实下载/启用/推理不是本次完成项。
S4 Closed Unverified / Release No-Go 不变。
