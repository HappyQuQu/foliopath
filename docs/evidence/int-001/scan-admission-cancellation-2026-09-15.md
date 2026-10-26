# 扫描准入取消：复用扫描收尾规则

CR-2026-030 / POST-MVP-5 接入验证 / FR-INT-016 / NFR-INT-007。
Owner scanner；resourcecontrol 只通知，SQLite 的取消事务仍为原实现。
不修改 API、数据合同、扫描调度或媒体边界。

RunClaimedFullScan 原先在入口 context 已结束时直接返回，留下 running 记录。
现复用 scanner.abort 的有界收尾（默认 5 秒）；ClaimedProcessor 同时接入资源
准入取消通知。取消及超时遵守扫描已有 cancelled 规则，不套用 AI 的停机恢复策略。
这不改变扫描运行中既有取消语义；后续全量扫描仍负责协调媒体真相。

新增 SQLite 组合测试覆盖直接入口、资源槽占满两种路径，各验证取消和超时：
扫描终态 cancelled/completed、租约清除、不打开媒体、旧资产/目录数量及可靠代次
保留，随后可重新提交扫描。context 在调用前已经结束；不将其算作实际 worker
等待后由心跳触发取消的端到端证据。

## 本地 Docker 验证

回归镜像 manifest list：
`sha256:ff4b38eebb45f352267c59a8fe43c2598b145682151035c525d8dafef81f0b98`。
所有测试使用合成夹具和临时 SQLite，二进制带 race，容器禁止网络。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "TestScanCancellationBeforeAdmissionPreservesCatalog|TestSemanticClear|TestImageBackfillCancellationBeforeAdmission|TestVideoCancellationBeforeResourceAdmission" -test.v -test.timeout=3m && /out/resourcecontrol.test -test.count=20 -test.timeout=2m'
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "TestScannerService|TestOfflineAndInterruptedScans|TestRestartMarksRunningScan|TestCancelledScanPreserves" -test.v -test.timeout=3m'
make arch-check
```

第一组 SQLite 10 个顶层测试及资源控制 20 轮通过。
第二组扫描既有收尾 9 个顶层测试及 make arch-check 通过。
未重建实际运行镜像，未执行全套七项检查，未访问 server222 或真实媒体。
扫描/清理实际混合等待取消、清理批次交错、图片最终提交竞态、人物与发行仍待完成。
