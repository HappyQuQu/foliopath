# P3 现有能力收口验收

> 本文保留当时状态、失败及撤回记录；最新结论见 [P2/P3 全项复查](p2-p3-final-recheck-2026-09-16.md)：修复后最终固定镜像重新验收，P2 6/6、P3 9/9、18/34；不代表质量、容量或发布完成。

> 复查更正：下文保留原运行记录，但“九项全部完成”已撤回。D03 两项实际 UI 缺陷、D04 验收缺口重新打开，当前 P3 7/9、总计 16/34。以 [P2/P3 复查](p2-p3-review-2026-09-16.md) 为准；历史 229 单测未包含新增的两条失败回归。

范围 POST-MVP-5 / CR-2026-030 / FR-INT-009、010、016 / NFR-INT-002、006；
owner：face/settings、semantic、resourcecontrol、SQLite、web 共享组件及验收探针。
对应剩余清单 B05、C01～C05、D01/D03/D04。**九项全部完成功能验收；当前清单 18/34 关闭、16 项保留。**

## 修复与失败记录

- 实际双容器中断后复现：模型恢复 available，但库仍为 awaiting_model，重试被 409 拒绝。
  修复 `face.SettingsService.JobSettings`：同代次等待模型时，也通过既有 revision-guarded
  Update 入口验证可用性并恢复 building。不绕过仓储拒绝规则，不开启 disabled 库，旧失败任务不改写。
  新仓储回归覆盖 offline 不变、恢复可重试、禁用不重开。
- 新版浅色未分析提示的真实 axe 检出 4.25:1；共享 InlineStatus 改用中央正文 token，
  无功能私有颜色；普通状态 class 不再包含 undefined。组件测试及新版 axe 复验通过。
- 正式停服页面切回文件名后重搜失败：GlobalHeader 把已有 query 的 searchHref 再拼接 `?q`。
  修复共享提交 owner 使用 URLSearchParams.set 替换关键词，保留目录范围/语义模式；两项组件回归覆盖编码和上下文。
- 错部署身份探针复现旧模型仍被 advertised available：bootstrap 原先只更新与新 fingerprint 匹配的记录。
  现在同 purpose 的旧 service 身份不再配置时标记 unavailable，保留旧代次和所有数据；
  新身份校验失败不激活，恢复原配置可恢复 availability。语义/人脸两条 bootstrap 均增加回归。
- 测试准备失败（不算产品缺陷）：应用镜像不含 lavfi 生成器，改用宿主 FFmpeg 只生成临时合成视频；
  Docker internal-only 网络不发布回环端口，主应用增加测试用 edge 网络，AI 仍仅在 internal 网络且无公开端口；
  标签需先勾选才出现审核按钮；旧已审核决定换请求键返回原决定，冲突改由错误 revision 的 pending 建议验证；
  大量重复登录触发真实限流，浏览器各 worker 改为复用首次正式登录的 cookie，不关闭限流；
  重建覆盖候选 tag 后旧 manifest 不再可用于重启，探针用唯一临时 tag 保留被测镜像身份。
- 当前 Playwright Chromium 实测 `canPlayType` 对 H.264 和 AAC 均返回空字符串。
  媒体矩阵明确记录 codec-unavailable 并验证不可播放提示，**没有证明本 runner 的 MP4 解码/播放成功**。
  不取消范围响应合同，不把该限制写成人脸/检索功能失败或全平台播放通过。

## 镜像及边界

- 最终应用 `foliopath-app:p3`：`sha256:18e45a94f14f1fa8b5ec8466ac4fa5dd0e1b95016146ade40e3d7792e9e01b38`。
- 推理 `foliopath-inference:face-candidate`：`sha256:28f8fb6351234b5e94d09477f9d112ef36f2b41bc8537268d50e6bd1459f0d85`。
- 测试浏览器 `foliopath-p3-browser:local`：`sha256:0642cd6f9d369296552d22e5268ea1185ac23e233b218aaa3fad07a6c15bd7df`。
- 仓储回归镜像 `foliopath-remote-ml-regression:p3`：`sha256:2ad03208cae254eb15f274184dee220b4764a1b5da56a287a201fe0c73e8bbac`。
- 全部应用/推理/浏览器测试运行于本地 Docker；合成输入在系统临时目录，应用只读挂载一个 `/library`。
  现有模型缓存只读；损坏权重使用测试私有同大小零字节文件遮盖，不修改缓存。
  每次独立网络、数据卷、随机管理员、最终清理；没有访问真实媒体库或 server222。
  临时 profile 明确为 synthetic-only，不是质量校准。

## 已执行检查

- `p3-business-probe.mjs`：真实图片/视频新增、修改、移出矩阵，失效排除、跨库隔离；
  视频降级仅有汇总归属，成员缩减时退回 pending，重跑重新分类，不冒认逐资产失败账本。
  标签正式 HTTP 接受/忽略、同键重放/异体 409、pending revision 冲突、跨库混合账本清理；
  清理库 A 后 A 旧请求冲突、B 原决定保持，人工标签不删除。
  **没有标签 undo 合同**，该项 N/A；人物审核撤销不是标签撤销。
  人脸占有共享额度时，图片/视频/标签/扫描/人脸派生清理等待任务分别取消，处理计数零，浏览仍可读。
- `p3-fault-probe.mjs` 最终镜像全部通过：模型缺失、同大小坏权重、64 KiB 快照盘满、只读快照目录、
  128 MiB AI cgroup 实际 OOM；主应用不误激活、不清空旧图片/人脸/代次，逐项恢复服务，最终分析成功。
- `recovery_smoke.sh foliopath-app:p3` 通过：真实管理员备份恢复、迁移重复启动、WAL 强杀恢复、
  数据盘满、坏数据库、数据只读、权限不足拒绝启动；只读失败前后备份数据库哈希不变，恢复管理员可读，媒体 sentinel 不变。
- `upgrade_rollback_smoke.sh foliopath-app:face-worker foliopath-app:p3` 通过：
  前版 `sha256:992fd5a6d817230d9bae440b07ec3f3bb4376ecb6cb5039da47e62df1bec6682`，
  升级前备份 SHA-256 `827d6eb40627c4d01189c5b1dfe75973d836e74e0dbdab5f9601ce21b648954d`。
  回退使用旧镜像与升级前备份配对，不向旧镜像直接交升级后的库。
- 迁移 40 新回归 `TestFaceServiceMigrationRefusesLossyDowngrade` 通过：空态 39→40 可用；
  存在 service face model/generation 时降到 39 拒绝，回滚事务后引用与记录保留。
- Docker race：resourcecontrol、face、semantic 全包及 SQLite 人脸/视频失效/标签清理/租约相关集合通过；
  face 容量测试按既有开关跳过，未当成 P4 通过。
  remoteml 取消/响应体超时/槽释放/身份/人脸失败分类测试通过。
- 最新源码前端生成检查、边界检查、类型检查、52 文件 **229 单测**通过。
  Docker `go test ./tests/architecture/...` 通过；首次 network-none 因依赖缓存缺失失败，联网取得依赖后复跑通过。
  全部 OpenAPI 合同测试（含 revision hash）通过，`git diff --check` 通过。
  `make fmt-check`、五个新探针 `node --check`、恢复脚本 `sh -n` 通过（均为静态检查）。

### 复现命令索引

```sh
node spikes/int001-remote-ml/p3-business-probe.mjs
node spikes/int001-remote-ml/p3-recovery-probe.mjs
node spikes/int001-remote-ml/p3-fault-probe.mjs
node spikes/int001-remote-ml/p3-browser-probe.mjs
FOLIOPATH_RELEASE_BUILD_HELPERS=0 sh tests/release/recovery_smoke.sh foliopath-app:p3
FOLIOPATH_RELEASE_BUILD_HELPERS=0 sh tests/release/upgrade_rollback_smoke.sh foliopath-app:face-worker foliopath-app:p3
```

回归镜像 `/out` 二进制均由 `go test -c -race` 编译。本轮执行：
`resourcecontrol.test`、`face.test`、`semantic.test` 全包；
`sqlite.test -test.run 'Test.*(Face|Video.*(Removal|Scan|Coverage|Cancel)|TagReview|Recovery|Lease)'`；
新增修复后复跑 `sqlite.test -test.run 'TestFace.*(Settings|Admission|Retry|Terminal|Service|Rebind)|TestCancelFaceJob'`、
`app.test -test.run 'Test.*(Deployment|Face)'`、`api.test -test.run Face`、`contract.test`；
`remote-image-probe.test -test.run 'Test(Cancellation|Response|FaceRuntime|FaceClient|Configured|Identity)'`。
这是明确选择的回归集合，不是所有 Go 包或 G01 的完整命令集。

## 最终浏览器与恢复结果

最终三浏览器在线/移动组合 **20 passed / 8 条按项目条件跳过**，停服专项 **3 passed**。
8 条为在线阶段跳过的 3 个离线专项、桌面项目不执行的 3 个移动专用用例、移动项目不执行的 2 个桌面专用用例；对应离线/移动专项已单独执行，不掩盖失败。
其中正式后端 P3 用例在线 9 条、离线 3 条；既有 mocked viewer/触控矩阵 11 条，两类证据不混同。
设置矩阵含 24 个主题/语言/断点组合；实际标签/人物页面另做 axe；真实 400 项图库证明跨页加载且 DOM <100，不是 100k 容量结论。
截图位于忽略的 `test-results/p3-real` 和 `test-results/p3-real-offline`，未提交为生产资源。

最终 `p3-recovery-probe.mjs` 完整 exit 0：正常停机与 SIGKILL 的实际持久租约恢复、AI 连接中断后失败/恢复重试、暂停服务造成真实 HTTP deadline 后失败/恢复、错 runtime 身份保留旧代次但不可入队、恢复正确配置后可用均通过。没有手工推进数据库时钟、修改任务状态或把停机标成用户取消。
随后停止主应用，备份整个应用状态到独立卷（不带 cache/tmp），再恢复至另一独立卷；原会话、非空人物的一个人工归类、人工标签及四个资产可读。正式派生清理、重新开启和真实人脸重建后，人工归类/标签仍在，合成原图 SHA-256 不变。所有临时应用/AI 容器、网络和数据卷均随探针退出清理。
完整七项仓库门禁仍属于 G01；P4 质量、100k 容量、原生双架构和发行批准不在本页冒充完成。
