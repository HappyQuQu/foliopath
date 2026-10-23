# P4 约 100 样本独立功能验证

范围：[用户调整记录](../../changes/CR-2026-09-16-p4-small-sample-validation.md)；POST-MVP-5 / CR-2026-030 / B01～B04、E01～E04 增量，Owner app/测试。不是训练、全量压力或准确率验收。

## 采样与隔离

`spikes/int001-remote-ml/sampleprep` 仅在显式 opt-in 且 Linux `/library` 为只读时执行；访问复用 `internal/files`，拒绝 symlink/mount 穿越。最多遍历 20000 条目，按路径哈希排序保留有界候选，优先 90 图片/10 视频、不足时另一类补齐 100。单图不超过 64 MiB、单视频 256 MiB、总副本不超过 2 GiB；超限失败，不继续整库处理。这是图片/视频类别分层，不是人工场景代表性标注，也不保证源文件内容互异。

原路径只在采样进程内存在；输出匿名 sample ID、字节数及 SHA-256。源目录只读，副本在独立临时目录中；应用只挂载副本，AI 无媒体/数据库挂载。报告不包含原文件名、身份或图片。原始 manifest/report 保留于忽略的运行目录，不提交真实媒体。

本地来源 `/Volumes/r0data/picture/temp`：遍历 4484 条目、跳过 163 条，提取 90 图片/10 视频，695406706 bytes。跳过包括尺寸/格式边界或文件边界问题，不能把它们称为模型失败。副本 `/tmp/foliopath-p4-small.cWunjH/samples`；报告 `test-results/p4-small/local-report.json`。

## 本地 arm64 已完成

使用先前固定应用 `59521dc239c2076b861fa4a9beb9281b2ffcbb504d182fa26aa00a37ce3add3d` 与 AI `28f8fb6351234b5e94d09477f9d112ef36f2b41bc8537268d50e6bd1459f0d85`，配置 app 1 CPU/768 MiB、AI 3 CPU/2560 MiB，无 swap。复用原 P3 隔离环境先跑合成预检，再创建独立 100 样本库；合成样本不计入 100。

| 项目 | 实际结果 |
| --- | --- |
| 扫描 | 100/100 入库 |
| 图片语义 | 90/90，失败 0，27337 ms，期间 28 次浏览成功 |
| 人脸处理 | 90/90 图片，失败/降级 0，33271 ms，期间 34 次浏览成功；不是检测到 90 张脸 |
| 视频语义 | 10/10，失败/降级 0，10077 ms，期间 11 次浏览成功 |
| 标签任务 | 90/90，失败/降级 0；person/outdoor/indoors/landscape/animal 词表，没有自动接受建议 |
| 搜索 | person/outdoor/人物/室外 × 图片/视频共 8 请求成功；图片各返回 20、视频各 10；单次 34～55 ms，不是 P95 |
| 副本完整性 | 运行前后逐文件 hash 相同 |
| 结束时内存快照 | app 299 MiB、AI 2.315 GiB；不是峰值或泄漏结论 |

本地探针退出 0，并清理自建应用/AI、数据库卷、网络；匿名采样副本保留供复核。没有人工 ground truth，不计算/声称准确率、召回或聚类质量；没有断言所有建议正确，也未测试两种视频帧计划的质量比较。

## server222

从 `/mnt/r0data/picture/` 通过同工具独立采样完成，目录 `/tmp/foliopath-p4-small.Jrkuq0`。首次 `/tmp/foliopath-p4-native.Xa56fS` 失败：100 样本入库后，图片分析期间主应用断连。Docker 事件明确记录 `oom`、exit 137，触发 app 768 MiB 限制；不是测试通过。退出脚本 `test_exit=1 / services_restored=1`，原四服务健康恢复。保留失败报告。

第二次 `/tmp/foliopath-p4-native.fflz0L` 保持相同镜像/样本/内存配额，通过正式 settings API 将 backgroundConcurrency/contentReadConcurrency 降为 1，仍失败。语义任务 settled/succeeded，但 coverage 为 54 completed / 36 failed / complete=false；人脸阶段 app 再次 OOMKilled=true、exit 137，AI 仍健康。视频/标签/查询未执行。结束 `test_exit=1 / services_restored=1`，另行 SSH 确认原四服务均 healthy、镜像前后相同。未提高配额、替换大图或将 failed 样本排除来宣称通过。

只读样本头部审计：90 图片全部头部有效，36 张 >16000000 像素，1 张 >100000000，最大 101896752 像素。当前 `semantic.RemoteImageMaxPixels=16000000`，超限数量与语义失败数一致；尚无逐资产失败映射，不把聚合匹配当逐项证明。人脸 OOM 的具体输入/分配热点仍需有界复现，不能仅靠头部审计认定唯一根因。两侧样本不同，不能用两份报告计算跨架构数值误差。

本地同样串行配置复跑退出 0：图片 90/90（16109 ms）、人脸图片 90/90（32220 ms）、视频 10/10（10078 ms）、标签 90/90；期间浏览 17/33/11 次成功。结束内存快照 app 219.7 MiB / AI 2.009 GiB，不是峰值。原始报告 `test-results/p4-small/local-serial-report.json`。

## 检查与剩余

两架构采样工具交叉编译成功；Docker 实测未 opt-in、非只读挂载均以 1 拒绝执行；`make arch-check`、`go vet` 采样工具、Node/Bash 语法检查、`git diff --check` 通过。新增 3 个结果判定回归通过，拒绝“任务 succeeded 但覆盖率失败”、缺阶段、降级、空查询或哈希不符。原探针仅检查 settled state 的不足已修正，两份本地实际报告经新判定器独立复核均通过；远端两轮均实际非零退出，不存在远端通过声明。最新采样工具增加总字节预检查及只读头部审计；主分析流程未因该工具调整重跑。未跑全部七项发行检查。

下一步优先定位并修复大尺寸样本人脸预处理 OOM，使用合成同尺寸回归复现后复测原远端样本；不扩大原图写权限、不静默降低尺寸合同。独立人工相关性/标签/匿名人脸标注、小样本长时间循环及同输入双架构比较仍未完成。原完整验收进度保持 18/34，不将本报告记作原 8 项全部关闭。
