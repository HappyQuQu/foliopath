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

## 尚未关闭的任务与精确阻塞

- `INT-401`：缺最终审核 semantic/face package，因而不能完成真实 inference、人物创建、升级/回滚和
  来源损坏的完整产品纵向。
- `INT-402～403`：缺同一 source SHA、model package 与 final image digest 的原生 amd64/arm64 配对
  artifact，也缺最终模型联合 100k/10k 容量和 governed semantic/tag/video/face 质量报告。
- `INT-404`：缺最终双架构 SBOM、签名 provenance、license/notices、漏洞/VEX 和模型权重再分发签署。
- `INT-406`：代码级日志/API/诊断隐私边界通过，但最终 privacy/compliance/security 发布批准缺失。
- `INT-408`：现有自动化不替代三浏览器、物理触摸、读屏和大集合虚拟化的最终人工/目标设备签署。
- `INT-410`：当前没有最终发行源和真实部署镜像；在线/国内镜像入口已从范围和 UI 删除，只保留离线
  `/models:ro` 合同。
- `INT-411`：五个 final verifier 的真实 summary 和 product/ML/QA/privacy/compliance/security/release
  批准没有同时存在，不能签署 Integrated Slice Done。

GitHub 上的 intelligent-media native workflow 只存在于 `aifeature`，不在默认分支；2026-09-02 对
`aifeature` 发起 `workflow_dispatch` 返回 HTTP 404。未经明确授权不得把功能分支合入默认分支来绕过该
限制。即使运行 baseline workflow，它也不会产生严格 final model/quality/supply-chain 证据。

## Gate 结论

S4 可以继续接收真实最终产物，但当前不得发布或宣称 A～E 全范围完成。reviewed catalog 保持空，face
production composition 和未签署发行 UI 继续失败关闭；普通只读媒体浏览不受影响。
