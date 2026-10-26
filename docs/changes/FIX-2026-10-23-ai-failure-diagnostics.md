# AI 人物解析失败诊断与扫描等待

后续维护：本记录末次观察的 scan-worker 风险随后进行了独立定位，已在合成库复现跨宿主机/容器 SQLite 检查
引发 522 短读，并改为容器内只读维护；6010 更新与最新工程证据见
[scan-worker 修复记录](FIX-2026-10-23-scan-worker-exit.md)。下文为当时验收事实，不逐次回写旧原因。

状态：功能实现和本地专项通过；6010 持续运行验收未全通过（scan-worker 自动退出复现），不等于 AI 全能力或发行通过。
用户同意补齐失败原因和重试入口。
POST-MVP-5 / FR-INT-009/016，Owner：Codex。

## 本次批准切片

- 先修复真实出现问题的人物解析链路；不把它宣称为搜索/视频/自动标签全能力诊断。
- 既有 faces 设置读取合同增加可选 diagnostics：当前 generation 最近一次任务、当前任务逐图失败总数、
  按固定原因码统计，以及最多 20 张失败样本（资产 ID、文件名、时间、固定原因码）。这是有界诊断摘要，不是完整错误日志浏览器。
- 历史缺少逐图记录的数量单独显示，不补造原因。只返回脱敏原因，不返回异常原文、主机路径、SQL 或人脸数据。
- 继续复用原 missing 任务：重试未完成/失败图片，不重做已有成功结果；不增加按任意资产批量执行的 API。
- 任务中止、用户取消、图片失败分别显示。查看详情原位展开；原型直接使用正式组件和显式合成合同。
- 在途人物任务碰到 scanning 短暂等待最多 60 秒，每秒复核取消/库/模型并续租；超时明确停止为 library_scanning。
  新任务仍要求库 ready；离线/禁用/清理不等待，不开放自动重试。策略见 ADR-0022。

## 合同和持久化

api/openapi.yaml 为 wire 合同；internal/face 拥有原因码映射和诊断结构；SQLite migration 43 保存
当前任务、generation、库、资产和源指纹绑定的失败记录，与 checkpoint 原子提交。新任务只展示本轮诊断，
上一轮未恢复记录继续作为 missing 候选 fence，不能被旧成功结果覆盖；重放/合并请求不得清除。
成功处理删除该图失败标记，清理、库/资产删除不留下可见旧诊断。

## 验证清单

- 原因分类、扫描等待/超时/取消/离线、CAS 失败回滚、历史原因缺失、跨库/generation 隔离与样本上限。
- API 鉴权和无敏感异常泄露；生成客户端；正式组件错误详情/原位切换/重试参数与断点。
- 本地 Docker 合成媒体，扫描等待与失败后恢复；不得通过改真实库数据伪造通过。完成后补证据和实际边界。

## 实际验证与边界（2026-10-23）

- 后端：`go test ./internal/... ./cmd/...`、`go vet ./internal/... ./cmd/...`、face/SQLite/API/remoteml/app/contract
  专项及定向 race 通过。覆盖等待恢复、超时、取消、离线/禁用/模型失联、checkpoint CAS 回滚、跨库/代次、
  源变化、30 条失败只取 20 样本、旧成功后新失败仍纳入 missing、逐批清理。公共 operation 合同补齐四个
  人物任务错误码并增加回归；生成 TypeScript 与中英文错误消息同步。
- 工程：`make fmt`、`make arch-check`、`make generate-check`、`make contract-check` 通过。
  `make test-integration` 在扩展盘临时目录通过；默认内部盘临时目录触发模型存储安全余量拒绝，未绕过保护。
  `make test-e2e` 是 Docker runtime/application smoke，通过，但不是完整 Playwright 产品矩阵。
- 前端：65 文件 / 409 测试、应用/工具类型、生成一致性、前端架构及 Storybook 构建通过。
  `make lint` 和无限定 `make test` 仍受本机缺少 pkg-config/libvips 的 native spike 依赖阻塞；不能写作全过。
- 最终主镜像 `sha256:ce6911fb7c63f4974f21043bdd5463ac33d0a80cce65e598180f6a4731e9f7b5`，AI 保持
  `sha256:fa7750b57500031a3282181e3b728646717885b339a0640a76b121468165e0b8`。
  最终人物 smoke `fp-face-diagnostic-d412c639` 通过：真实模型、4 张合成有效图＋1 张故意损坏 JPEG，原因 invalid_media、
  只重试 1 张、幂等重放、重启持久化、鉴权/库隔离/脱敏、只读原件 hash 不变和无 OOM。
  扫描等待通过**仅临时合成库**注入 scanning 状态三秒验证，不冒称真实 scanner 的全量并发/长期时序验收。
- 最终同镜像自动标签完整故障 smoke `fp-auto-tags-d2073e67` 通过，含人工保护、删除抑制、运行中重启、取消、
  停 AI 后显式恢复、关闭持久化与源 hash。不将新人物诊断误称为自动标签/搜索逐图错误诊断。
- 正式组件真实浏览器：合成 Docker 上重试后 operation 为 1/1，焦点在人物标题；文件链接→重试的键盘顺序、
  切库隔离、四档 390/768/1440/1920 及英文深色 390 无横向溢出，console error 为 0。
  新诊断 Story 与 TaskWorkspacePrototype 复用生产组件，合成数据明确标注。
  上述合成 UI 四断点截图对应 `68c651d1…` 候选；最终 `ce6911fb…` 仅补 operation 错误码枚举与其缺失的
  中英文消息，不改布局，重新通过 409 测试/类型/生成/Storybook，并在原 6010 桌面复核。
- 原 6010 已停写备份后迁移到 schema 43；4 库 / 8,719 媒体保留，quick_check=ok、外键检查无问题，双服务 healthy、
  初次检查 restart=0/OOM=false，最终检查主应用 restart=1（详见下文），AI 仍为 0/OOM=false。
  未发起真实库分析或清除；历史计数未人为归零。用户重新登录后，已只读核对 pixiv 的
  4/999、失败 0 与任务中断，以及 coser 的 469/6202、失败 18 与任务中断，详情正确显示 library_not_ready
  和历史逐图原因缺失。最终镜像又只读核对 8k：785/792、失败 7，但最近任务正常结束，未误标成任务中断。
  原库截图在 `docs/evidence/ui-audits/ai-failure-diagnostics-2026-10-23/`，最终 coser 为
  `original-6010-coser-final.png`，console error=[]。

旧版本没有保存的逐图原因无法还原；补齐只能记录新失败，损坏源文件不能自动修复。诊断仅展示当前代次最近任务
及最多 20 条失败样本，不是完整服务器任务历史或日志系统。扫描超过 60 秒会明确停止，用户应在库 ready 后重试。
普通照片质量、容量、AMD64、完整浏览器矩阵、发行门禁不变；旧候选 scan-worker 退出风险仍开放，
本次人物任务扫描等待修复不证明该独立问题已修复。

### 最后运行复查：开放问题，不能标记全部通过

最终 6010 主镜像 `ce6911fb…` 在 2026-10-23 02:46:44 +08:00 再次记录
`application.component_stopped`（component=`scan-worker`）、`application.shutdown_failed` 后自动重启。
最终主容器 restart=1、AI=0，均 healthy/OOM=false；ready=ready，SQLite quick_check=ok、外键检查为空，
8,719 项媒体保留。说明不是只存在于旧候选的历史风险，**本轮仍可复现，尚未修复**。
现有日志只暴露组件名而没有底层固定原因码，无法据此断言是 SQLite 锁、扫描源或其他故障。
`scanWorkerComponent` 将 worker/startup admission/scheduler 任一错误传到生命周期，随后进程退出；
这是当前可证实的退出传播路径，不是已确定的底层原因。未用吞掉错误或无限重试掩盖。
重启前后最近两批 startup scan 均 succeeded、error_code 为空；不能拿扫描行成功证明 worker 无错。
后续需独立补齐安全原因码、复现扫描/数据库竞争与恢复回归；本轮功能专项通过不能关闭此运行验收项。

### 可复跑命令

在本地 Docker 与已复制并核验的缓存下执行，脚本创建独立合成项目，结束只移除自己项目的容器和网络，
合成数据保留；没有删除原媒体、原库或用户数据。两个 smoke 均接收相同三个变量：

```sh
FOLIOPATH_TEST_APP_IMAGE=sha256:ce6911fb7c63f4974f21043bdd5463ac33d0a80cce65e598180f6a4731e9f7b5 \
FOLIOPATH_TEST_AI_IMAGE=sha256:fa7750b57500031a3282181e3b728646717885b339a0640a76b121468165e0b8 \
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-experiments/automatic-tags-integration-cache \
node tests/release/face_diagnostics_smoke.mjs
```

自动标签回归将最后一行换为 `node tests/release/automatic_tags_smoke.mjs`。默认不保留容器，不能把
脚本打印的已清理临时端口当作当前可访问环境。原 6010 的已部署入口是
`http://127.0.0.1:6010/settings/ai?library=lib_1&view=tasks`；不需要重新初始化管理员。

最终两次输出的合成数据保留目录：

- 人物：`/private/var/folders/21/hjbt152d0vs3n8td7233xd8r0000gn/T/fp-face-diagnostic-d412c639-j4bIN8`。
- 标签：`/private/var/folders/21/hjbt152d0vs3n8td7233xd8r0000gn/T/fp-auto-tags-d2073e67-NUgvTJ`。

此前 `68c651d1…` 的 `fp-face-diagnostic-e27e389a` / `fp-auto-tags-a6c22ce5` 也通过，属于前一候选历史，
不与本次最终镜像 ID 混写。两次最终脚本拥有的临时容器/网络均已移除，未删除数据目录。
