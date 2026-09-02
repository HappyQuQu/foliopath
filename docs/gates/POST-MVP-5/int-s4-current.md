# INT-S4：Integrated Slice Current

- 日期：2026-09-02
- 目标版本：`POST-MVP-5` revision 2
- 范围：`INT-401～411`
- 进度：**3 / 11（27%）**
- 判断：**In Progress / Release No-Go**
- 已完成：`INT-405`、`INT-407`、`INT-409`

## 本次关闭的任务

`INT-405` 的恢复边界已由 SQLite 人物状态备份/恢复、数据库打开时自动修复缺失/不一致 catalog FTS、
missing/stale semantic backfill、进程强杀 lease 恢复、managed model 实际 ENOSPC，以及真实候选容器的
离线恢复、SIGKILL/WAL、磁盘写满和损坏数据库失败关闭共同覆盖。可重建索引可以缺省，人物和人工关系
不可缺省；两者没有被混为同一种备份数据。

`INT-407` 的原件边界已由真实候选容器的只读 rootfs、`/library:ro`、无 capability 与前后 sentinel
SHA-256 对照覆盖。文件边界全包回归继续覆盖 traversal、重复编码、NUL、symlink/hardlink、跨设备、
nested mount、目录替换竞态及 poisoned catalog path；clear 回归还保持 source size/mtime/fingerprint。

`INT-409` 已把中英文用户 README 和部署文档收敛到 revision 2 当前事实：功能已实现但尚未获准发行，
匿名相似分组不等于现实身份识别；只支持 reviewed catalog 精确匹配的离线 `/models:ro`，没有下载或
镜像入口。模型来源、升级/配对回滚、人物状态恢复、容量限制、稳定故障排查、隐私诊断和清除语义均
有用户/运维入口，安全细节继续由 `docs/security.md` 权威持有。

## 实际执行证据

以下命令在当前 `aifeature` 工作树实际成功：

```text
go test -count=1 ./internal/store/sqlite -run 'TestFace.*(Backup|Clear|Restore|Lifecycle)|TestSemantic.*Clear|TestFace.*(Unavailable|Offline|Disabled)'
go test -count=1 ./tests/architecture -run 'TestFacePrivacyProjectionRemainsClosed|TestIntelligentMedia'
go test -count=1 ./internal/files -run 'TestManagedModelStore.*(ENOSPC|Corrupt|Space)|TestModelSource'
go test -count=1 ./internal/store/sqlite ./internal/semantic ./internal/files ./internal/pathpolicy ./tests/integration
FOLIOPATH_RELEASE_IMAGE=foliopath:s4-local FOLIOPATH_RELEASE_EXPECTED_VERSION=stage5-local FOLIOPATH_RELEASE_BUILD_IMAGE=1 tests/release/image_smoke.sh
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
make test-intelligent-media-offline
make test-intelligent-media-privacy
make test-web-release-e2e
make test-browser-capacity
```

最后一项在本机原生 `linux/arm64` Docker 上完成镜像构建、认证媒体矩阵、Compose、可信代理和
`recovery_smoke.sh`，结果为 `Stage 5 candidate image smoke passed`；候选 manifest-list digest 为
`sha256:b4393613bad42cb6c96e6b71c9a76ea0ec90cd8c90d12b6f48ce54ad5500caea`。这是本地候选证据，
不是最终双架构发行 digest。

新增的离线拓扑检查也在本机原生 `linux/arm64` 通过：候选容器使用 `--network none`、只读 rootfs、
`/library:ro` 和 `/models:ro`，完成管理员初始化、空 reviewed catalog/candidate scan、重启以及媒体/模型
sentinel SHA-256 不变；本次临时候选 digest 为
`sha256:680d3ba0740fc2f449ed9a3d7da995e37f8cd5b00bd0cc729ac1a8531fa24b91`。该结果只推进
`INT-410` 的无外网与只读模型拓扑子项，不是最终发行镜像或双架构验收。

本机随后安装锁定 Playwright 1.61.1 所需的 Firefox 151 与 WebKit 26.5，release E2E 的键盘/焦点、降级、
200% 等效重排与 storyboard 行为通过。100k 虚拟集合在 Chromium/Firefox/WebKit 均只挂载 60 项，超过
59 FPS，P95 不高于 21 ms，且低于 1.5 GiB RSS。详见
[浏览器自动化证据](../../evidence/int-001/int-s4-browser-automation-darwin-arm64-2026-09-02.md)。该证据推进
`INT-408` 的多引擎自动化和大集合子项，但不替代 retail Safari 或物理触控验收。

随后在同一 Mac 的 retail Safari 26.6.2 上以只读临时 fixture 库通过完整键盘顺序、预览、查看器、`I`
信息、`Escape` 焦点恢复与真实 200% 页面缩放；媒体 SHA-256 前后相同。详见
[retail Safari evidence](../../evidence/int-001/int-s4-retail-safari-darwin-arm64-2026-09-02.md)。该结果关闭
retail Safari/物理键盘子项，但没有开启或代替物理触控设备。

产品用户随后明确 S4 不需要 VoiceOver；
[CR-2026-024](../../changes/CR-2026-024-int-s4-screen-reader-waiver.md)从 `INT-408` 删除真实屏幕阅读器人工验收，
但不把未执行的测试写成通过，并保留语义、键盘和 axe 门槛。该决定不补足仍缺的物理触控/目标设备证据。

`INT-406` 的工程隐私入口随后通过：敏感日志属性 canary 全部脱敏，face/semantic/diagnostic HTTP 投影保持
封闭，所有 SQLite 连接强制 `secure_delete=ON`，删除 canary 经 WAL truncate 后不再存在于活动 DB/WAL/SHM；
derived/manual clear 仍保持分类边界和原媒体不变。详见
[隐私工程证据](../../evidence/int-001/int-s4-privacy-engineering-darwin-arm64-2026-09-02.md)。该结果不替代
privacy/compliance/security owner 的最终发布批准，因此不增加 S4 完成分子。

同一 source commit `5af4da0…` 的 GitHub Actions run
[`33616238888`](https://github.com/HappyQuQu/foliopath/actions/runs/33616238888) 随后在原生
`ubuntu-24.04` x86_64 与 `ubuntu-24.04-arm` aarch64 runner 上全部通过。两个 publish job 均跳过；repository、
production libvips、候选 face pipeline、100k×512 synthetic face、两库 order-first matrix 和强制 10k/100k
容量均成功，paired verifier 也通过。扫描为 `51,738 / 45,441 ms`，production keyset P95 为
`182,299 / 247,994 µs`，两端零预算违规。详见
[原生配对 baseline 证据](../../evidence/int-001/int-s4-native-baseline-linux-amd64-arm64-2026-09-02.md)。
summary 明确 `finalModelEvidence=false`，因此不增加 S4 主任务完成分子。

## 尚未关闭的任务与精确阻塞

- `INT-401`：缺最终审核 semantic/face package，因而不能完成真实 inference、人物创建、升级/回滚和
  来源损坏的完整产品纵向。
- `INT-402～403`：同一 source SHA 的原生 amd64/arm64 baseline artifact 已配对通过；仍缺同一最终 model
  package/final image digest 的严格 artifact、最终模型联合 100k/10k 容量和 governed semantic/tag/video/face
  质量报告。
- `INT-404`：缺最终双架构 SBOM、签名 provenance、license/notices、漏洞/VEX 和模型权重再分发签署。
- `INT-406`：日志/API/诊断、活动 SQLite 删除残留和无 face crop cache 的工程边界通过，但最终
  privacy/compliance/security 发布批准缺失。
- `INT-408`：三引擎自动化、100k 虚拟化、retail Safari 与物理键盘通过；VoiceOver 已由产品明确豁免，
  仍缺物理触摸和目标设备证据。
- `INT-410`：当前没有最终发行源和真实部署镜像；在线/国内镜像入口已从范围和 UI 删除，只保留离线
  `/models:ro` 合同。
- `INT-411`：五个 final verifier 的真实 summary 和 product/ML/QA/privacy/compliance/security/release
  批准没有同时存在，不能签署 Integrated Slice Done。

默认分支已注册的 Docker workflow 通过精确 evidence-only sentinel 在 `aifeature` 的同一 commit 复用
native workflow，并由互斥条件跳过所有发布 job；run `33616238888` 已成功。该路径仍只生成 baseline
artifact，不产生严格 final model/quality/supply-chain 证据。

## Gate 结论

S4 可以继续接收真实最终产物，但当前不得发布或宣称 A～E 全范围完成。reviewed catalog 保持空，face
production composition 和未签署发行 UI 继续失败关闭；普通只读媒体浏览不受影响。
