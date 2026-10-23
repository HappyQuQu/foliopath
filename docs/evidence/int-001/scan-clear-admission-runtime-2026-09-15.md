# 扫描与语义清理：实际混合任务取消验收

CR-2026-030 / POST-MVP-5 接入验证 / FR-INT-016 / NFR-INT-007。
扫描取消同时关联既有 FR-SCN-004 / FR-SCN-008 / NFR-REL-001。
Owner scanner/semantic/resourcecontrol/app；本轮只扩展验证脚本，无生产合同变化。

## 环境

本地 Docker `foliopath-multilib`，合成 alpha 视频和 gamma PNG，媒体只读。
主应用已重建并实际启动，镜像 manifest list：
`sha256:5a7b11ab114764953f163a92742e29ae272c7cc543fd6b83c96589f40b338c43`。
主应用仅 127.0.0.1:6091，AI 私网，不接触 server222 或真实媒体。

脚本 `spikes/int001-remote-ml/mixed-admission-probe.mjs` 增加 scan/clear 目标。
首次调用被挂载安全断言拦截：Docker Desktop 返回
`/host_mnt/private/tmp/foliopath-multilib.uEIluH`，并非原检查预期的 `/tmp/...`。
核实只读挂载后，检查兼容 macOS 规范化前缀，仍限定合成目录命名与项目。
首次失败不算通过；以下均为修正后的独立成功运行。

## 结果

后台额度临时设 1，视频先运行，暂停 AI 后提交并等扫描/清理进入 running，再取消。

| 场景 | 已验证结果 |
| --- | --- |
| 扫描 scan_16 | 15187 ms 进入 cancelled，发现资产 0；原媒体列表不变，视频仍 running、AI 仍 paused；恢复后视频成功、重新扫描成功，原取消记录保留 |
| 清理 aio_93145fa62c236606f1fa1782e6bab1be | 15158 ms 进入 cancelled，完成项 0；媒体列表不变，图片覆盖记录仍 completed=1，视频仍 running、AI 仍 paused；恢复后视频成功，重新启用图片语义并补齐成功，取消记录保留 |

两次运行最终图片/视频覆盖各 1/1，脚本 exit 0。清理恢复测试是重新启用和回填，
不是重新执行清理至删除完成；不声称已经删除的派生向量可由取消回滚。
脚本 finally 恢复 AI 和原后台额度。该结果是受控启动顺序与 HTTP 观测，
不是直接观测信号量队列，也不覆盖所有交错或压力容量。

```sh
# 设置隔离测试管理员 FOLIOPATH_TEST_PASSWORD 后执行
FOLIOPATH_TEST_CANCEL_TARGET=scan node spikes/int001-remote-ml/mixed-admission-probe.mjs
FOLIOPATH_TEST_CANCEL_TARGET=clear node spikes/int001-remote-ml/mixed-admission-probe.mjs
node --check spikes/int001-remote-ml/mixed-admission-probe.mjs
make arch-check
```

语法检查及架构检查通过。本轮未执行完整七项验证、浏览器 E2E 或压力测试。
清理批次交错、图片最终提交取消竞态、人物生产接入及发行仍待完成。
