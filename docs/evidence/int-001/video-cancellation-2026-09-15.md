# 视频取消收尾与跨库隔离回归

CR-2026-030 / POST-MVP-5 接入验证阶段 / FR-INT-004/016。
Owner：semantic worker（有界收尾）、SQLite（既有任务终态事务）。
沿用既有视频任务/取消合同，不增加 API、表或状态机；原始媒体只读。

## 已复现及修复

新增确定性交错测试在修复前复现三个失败：

- 心跳取消 work context 后直接返回，任务停在 cancelling，等待租约回收。
- 接受取消后最后一项提交，succeeded 收尾冲突，仍停在 cancelling。
- 接受取消后模型报错，被错误收尾为 failed。

worker 现使用脱离已取消 work context、最多 5 秒的清理上下文尝试收尾。
SQLite 仍以 operation/claimed revision 和持久化 cancelling 状态判定：
已接受的取消优先于随后到达的成功或失败；已提交资产进度保留。
仅进程退出而未接受用户取消时，不误写 cancelled，仍由原租约恢复机制处理。
清理写失败时保留原租约恢复兜底，不能保证数据库不可写时即时完成。

## 验证范围

本地 Docker，临时 SQLite 和合成资产记录，无真实媒体/服务器访问。
四种交错均验证 job、operation 终态；每种都同时保留第二库排队任务，
核对第二库整个任务及覆盖率记录未变化。不是多 worker 压力或浏览器端到端测试。

构建初次因 Dockerfile frontend registry 请求 EOF 失败；去掉传给构建器的
syntax 注释后用内置 frontend 构建成功，未修改仓库 Dockerfile。
回归镜像：`foliopath-remote-ml-regression:local`，manifest list
`sha256:b754d4e3a4636399b4faa36d542773ebf84fab61cc5353e4ef029274eada449c`。

最终执行成功：

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "TestVideo(Job|Status)" -test.v -test.timeout=3m && /out/semantic.test -test.timeout=3m'
make arch-check
```

SQLite 8 个顶层测试（新增取消测试包含 4 个子场景）及 semantic 全包均 PASS；
二者为 Docker 内带 race 的 Linux 测试二进制。架构检查 PASS。
仅对本轮三个 Go 文件运行 gofmt，不声称执行 make fmt。
未更新运行用 `foliopath-app:candidate`，
未执行完整七项验证；人物生产接入、浏览器运行中取消及发行验收仍未完成。
