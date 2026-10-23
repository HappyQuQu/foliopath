# 视频资源等待期取消收尾

CR-2026-030 / POST-MVP-5 接入验证阶段 / FR-INT-016 / NFR-INT-007。
Owner resourcecontrol（准入通知）、semantic（有界清理）、SQLite（既有终态 CAS）。
沿用已接受的 worker/任务契约与持久化状态，不增加部署单元、API 或预算。

## 复现与修复

新增 SQLite 组合测试：创建/领取视频任务，用其他持有者占满唯一后台资源槽，
启动真实资源包装器和视频 processor，再接受用户取消并取消 work context。
修复前 Docker 测试失败：job/operation 仍为 cancelling；停机场景通过。

资源包装器现支持可选 AdmissionCancellationHandler，在无法进入 processor 的取消路径
通知能力层清理；自身不判断用户取消，也不操作数据库。视频 processor 实现通知接口，
复用原有最多 5 秒的清理方法；已进入 Process 的取消也走同一方法，无重复终态策略。
数据库不可写或 CAS 失效时仍保留原租约恢复兜底。
没有该接口的处理器保持原行为，本轮不声称修复图片、扫描等所有类型的等待期收尾。

## 本地 Docker 验证

回归镜像构建成功，manifest list：
`sha256:130c3ffa8d60fc4921ef1e538ac32eb70eba9809cee68b8b2f49044db04265b1`。
只使用临时 SQLite 与合成资产，不挂载真实媒体或访问 server222。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "TestVideo(CancellationBeforeResourceAdmission|JobProcessorCancellationFinalizesWithoutLeaseRecovery)$" -test.v -test.timeout=2m && /out/resourcecontrol.test -test.count=20 -test.timeout=2m && /out/semantic.test -test.timeout=3m'
make arch-check
```

新增组合断言：不释放占用槽也能完成取消，推理不启动、处理数 0，job 与 operation
一致；无用户取消的停机保持 running，留待恢复。另回归 4 种原取消交错及第二库隔离。
最终结果：2 个 SQLite 顶层测试、6 个子场景全部通过；资源控制 5 项各 20 轮、
semantic 全包及架构检查全部通过。测试二进制带 race。
未重建运行用主应用镜像，真实混合任务端到端、其他任务类型、容量和发行仍待验收。
