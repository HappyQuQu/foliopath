# 实际混合任务：视频等待资源时取消

CR-2026-030 / POST-MVP-5 接入验证阶段 / FR-INT-016 / NFR-INT-007。
Owner resourcecontrol/semantic/app，沿用现有设置、任务与取消合同，本轮新增验证脚本而非生产逻辑。

## 本地环境

重新构建并启动 `foliopath-app:candidate`：
`sha256:7c1713e30d57689c487d0d68f10be48413207e1ffa8f48c68ccefcc2b120f5c4`。
`foliopath-multilib` 双容器健康，主应用 localhost:6091，AI 无公开端口或媒体/数据库挂载。
临时只读媒体根 `/tmp/foliopath-multilib.uEIluH`，alpha 视频、gamma 合成 PNG。
gamma 来自仓库合成测试夹具；新增独立 `Synthetic gamma` 测试库，未访问用户真实媒体或 server222。

脚本 `spikes/int001-remote-ml/mixed-admission-probe.mjs` 限制固定项目、只读临时媒体挂载，
通过正式 API 登录、读取/设置资源预算、建库扫描、回填、取消、查询和重试。
需要 `FOLIOPATH_TEST_PASSWORD`，密码未写入脚本。

## 观测结果

后台额度暂设为 1，暂停本地 AI，先启动 gamma 图片任务并等待 running；
随后启动 alpha 视频重建，领取到 running 后发起取消。

- 视频在 15256 ms 内达到 cancelled，处理数 0。
- 此时图片仍 running，AI 仍 paused=true。不是等待默认 120 秒租约过期回收。
- 恢复 AI 后，图片任务 succeeded。
- 视频 missing 重试 succeeded，图片和视频覆盖率各 completed=1；原视频操作仍 cancelled。
- 脚本 finally 恢复原 backgroundConcurrency，并恢复暂停的 AI；最终退出码 0。

结合上一轮占满资源槽的 SQLite 组合测试，该实测支持视频资源等待期取消修复。
这里没有直接采集内存信号量的等待队列；采用额度 1、图片先 running 再视频领取的受控顺序，
不把它当作所有调度交错、吞吐或容量证明。

## 验证与清理

```sh
node spikes/int001-remote-ml/mixed-admission-probe.mjs
node --check spikes/int001-remote-ml/mixed-admission-probe.mjs
make arch-check
```

实际脚本全部断言、语法检查和架构检查通过。测试结束停止本地容器，
保留 gamma 合成夹具、配置和派生测试数据以便复跑；没有删除原始媒体。
未重跑完整七项检查或浏览器 E2E。图片、扫描等其他类型自身在等待资源时取消、
压力/容量、人物生产接入和发行验收仍未完成。
