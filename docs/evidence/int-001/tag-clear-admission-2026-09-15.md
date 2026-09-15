# 标签审核记录清理：资源准入取消

CR-2026-030 / POST-MVP-5 接入验证 / FR-INT-016 / NFR-INT-007。
Owner semantic/SQLite；沿用资源通知接口和原标签审核清理合同，无 API/迁移变化。

Docker 先复现两个失败：占满后台额度、已取消 context 进入包装器时，已接受的
用户取消仍停在 cancelling；无用户请求的 running 任务却可直接 Finish 为 cancelled。

TagReviewClearProcessor 接入取消通知，并在 Process 退出时复用最多 5 秒收尾；
FinishTagReviewClear 要求持久化状态 cancelling 才接受取消终态。
单纯停机维持 running，依赖既有租约恢复，不假冒用户取消。

新增测试使用无审核记录的合成库、真实 SQLite 和占满的资源槽，验证用户取消与
停机两条路径、job/operation 状态一致且删除数为 0。context 在调用前已取消，
不声称覆盖真实 worker 等待后由心跳取消，或非空批次提交后的取消交错。

## 本地 Docker

修复后回归镜像 manifest list：
`sha256:2f16aa92546be29a581d5580d01f8e983dd973c537de23c35c797dee05d4258c`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run TestTagReviewClear -test.v -test.timeout=2m && /out/semantic.test -test.timeout=3m && /out/resourcecontrol.test -test.count=20 -test.timeout=2m'
make arch-check
```

使用 race 二进制、临时 SQLite，无网络、真实媒体或 server222。
最终 SQLite 4 个顶层测试、semantic 全包、资源控制 20 轮及架构检查通过。
未重建实际运行镜像，未执行全套七项检查。
标签审核清理最终成功/失败与取消交错仍有待修边界，本轮未修改该分支；
非空批次/实际环境、其他任务矩阵、人物生产接入、容量及发行仍待完成。
