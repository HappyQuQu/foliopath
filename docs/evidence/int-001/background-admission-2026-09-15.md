# 后台资源准入取消检查与共享预算回归

CR-2026-030 / POST-MVP-5 接入验证阶段 / FR-INT-016 / NFR-INT-007。
Owner resourcecontrol（既有唯一额度策略），合同为已有 Controller/Processor 接口。
不改额度、部署单元、API 或持久化语义。

代码检查发现 limiter 在有空槽时未检查已取消 context，可能把已取消请求送入处理器。
现于加锁后的准入前检查 context，并在取得名额后、调用处理器前再检查；
资源释放仍由原幂等 release/defer 管理。不保证处理开始后不会再收到取消。

新增测试覆盖空槽取消拒绝、处理器不被调用、后续请求仍能获得名额；
两个不同处理器共用后台预算，失败释放后等待者进入，前台 content 独立额度仍可用。
原有动态扩容、降限保留持有者及满额取消测试保留。

## 本地 Docker

在现有 remote-ml-regression 构建目标中新增带 race 的 resourcecontrol.test 二进制，
不加入发行镜像。构建成功，manifest list：
`sha256:163e50087a5732e97d4fedc4965b229a24816e90979430aed2db44a7e32e6b8f`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/resourcecontrol.test -test.v -test.count=20 -test.timeout=2m && /out/semantic.test -test.timeout=3m'
make arch-check
```

资源控制 5 项测试各 20 轮及 semantic 全包通过，架构检查通过。
仅格式化本轮两个 Go 文件，不声称运行完整七项检查。

## 未关闭的边界

这是模拟处理器的共享额度回归，不是实际 AI 图片/视频混合任务、吞吐或容量证据。
主应用运行镜像未重建。特别是任务已被 claim、但仍等待资源时取消，
外层资源包装器可能未调用业务 processor，因此需继续验证终态是否依赖租约恢复；
本轮取消检查并未解决该收尾问题。人物生产接入和发行验收仍未完成。
