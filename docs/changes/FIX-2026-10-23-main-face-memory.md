# 人物补齐期间主应用内存 OOM 排查

状态：共享大图准入/回收候选已通过所复现路径的合成 Docker 回归，并于四库结束后单次更新原 6010；
1 GiB 高熵探针仍触及上限，不宣称全部容量/长期 OOM 风险关闭。先前 SQLite 维护修复与本次原因分开。
范围：既有 FR-INT-009/016、R-006/R-024，POST-MVP-5 维护；Owner imagevips/app/face。

## 本轮代码修复与验收（2026-10-23）

用户要求继续查明剩余 27 张原因并修复内存。源文件逐张结果见[手工诊断](FIX-2026-10-23-face-source-diagnosis.md)：
实际输入错误与程序 OOM 分开，不强制放宽解码，也不改变原文件或把失败清零。

按 [ADR-0024](../adr/0024-shared-large-image-native-admission.md) 实施：

- imagevips 统一拥有进程级大图工作集单槽（>=16 MiB 压缩源、>=16 MP 或未知头），
  缩略图、人物、远程/旧语义复用，压缩缓冲/native load 前取得，ImageRef/线程/空闲页回收后释放。
- 等待可取消，不持 SQLite 事务，不创建工作 goroutine；稳定 file/bytes.Reader 最多 1 MiB 头探测并恢复位置，
  自定义取消/空读 reader 不增加读取，保留 canonical read 的预算。
- 大图阶段使用 LockOSThread/ShutdownThread + FreeOSMemory/Linux malloc_trim；没有设置 GOMEMLIMIT、
  新 RAM 上限、额外部署单元、输入落盘或不安全原媒体 mmap。像素/shrink/容错/模型身份与公开 API 不变。
- 非 Linux trim 是开发空实现，不作为 Linux/双架构资源证据；当前只对本机 Docker 平台验证。

### 不是一次初测通过就结束

最初“只回收人物解码”候选在独立探针可跑，但完整应用加入同一合成大 JPEG 后仍 OOM：
`fp-face-diagnostic-93ac7a45` 请求 socket 断开，内核于 09:18:23 记录该临时主容器 15ed533… 被 memcg 杀死；
复查 `fp-face-diagnostic-f69a9c5e` 自动重启 1 次。它们是临时合成环境，不是原 6010 新重启。
进一步加入跨缩略图/人物的大图共享准入，而不是吞错、提高预算或把 socket 断开当“测试不稳定”。
原 6010 本轮没有部署/重启、重提/重建/清理任务或修改原件/人工整理。

### 已执行证据

| 检查 | 实际结果与边界 |
| --- | --- |
| 未修复合成对照，96.8 MP progressive + 256 MiB 常驻堆，1 GiB | 先前 exit 137/OOM=true；不用真实图片作回归 |
| 最终共享准入/回收探针，30 次，同预算同输入 | exit 0/OOM=false，refs=0，RGB SHA256=`38276a09…`、1280×1600 与原实现相同；末尾 cgroup 约 360.7 MB，但 peak=1,073,741,824 B，仍无普遍容量余量保证 |
| 最终 baseline 对照 5 次，同保留堆/预算 | exit 0/OOM=false，RGB SHA256=`4f2c440a…` 相同；peak 522,149,888 B |
| native 短测 | 全部实际执行项通过；大图/显式输入用例的 skip 保留，不当作运行通过；准入取消、位置恢复、caller reader 不被探测、像素/引用/取消/错误回收有专项 |
| 最终大型 native 专项 | 81 MP PNG、JPEG EXIF 1/6/8、108 MP shrink 合同通过；重新编译的最终同源 test binary，exit 0/OOM=false |
| 最终完整应用 `fp-face-diagnostic-d886e552` | 固定 c9f28d7a 主镜像 + fa7750b5 AI；1 CPU/1 GiB 主应用、真实模型、4 张合成 PNG + 固定高熵 JPEG + 空/损坏负例；5 张成功/2 张输入失败，缺失只重试 2 张、幂等、扫描等待分支、重启持久化、隔离/原件 hash 通过；显式重启前主/AI 自动重启计数均 0 |
| Go/架构/生成 | fmt-check、arch-check、generate-check、生产 internal/cmd 包、定向 race/vet 与 contract 执行通过；最终文档/测试收尾再次检查 |

最终应用候选 manifest：`sha256:c9f28d7a6f4deb1d7d83bc7b73f0fd45e41609d546ac77a3b9c9543b652d9ec2`，
本地标签 `foliopath:local-face-memory-20261023`。没有推送/公开发布。
根 Dockerfile 首轮完整构建过早期候选；随后 registry frontend HEAD 两次 EOF，最终采用已固定本地 builder
41cdae58 + 原 cd1435af runtime 离线重编当前 imagevips/Go 模块，保留原嵌入前端与其余组件。
私有复现 Dockerfile 在手工诊断目录 `Dockerfile.app`；这不是最新完整发行构建、双架构或供应链验收通过声明。
最终完整合成数据保留于 `/private/var/folders/21/hjbt152d0vs3n8td7233xd8r0000gn/T/fp-face-diagnostic-d886e552-BkPCFK`。
可重复生成/资源探针代码及说明为 `spikes/int001-remote-ml/facememory`，没有真实媒体输入。
最终 native test/probe 镜像 manifest：`sha256:a7e253cbe321b9dede3509fd62b191c57ce6a30a6bcaf4cc784ac8734a9e0a20`；
最终短测 `fp-face-native-verified-20261023`、大型专项 `fp-face-large-verified-20261023`、
baseline `fp-face-baseline-verified-20261023` 均 exit 0/OOM=false。
收尾已移除本轮核验归属的 16 个退出诊断/探针容器、两个保留复查的合成项目共 4 个容器及其网络，
并删除仅存放临时 encoded spool 的实验卷；未删除原 6010 数据、备份、诊断结果、合成输入或候选镜像。
探针先前的取消读取/空读预算回归失败已修正（仅 peek 稳定 file/bytes.Reader），相应用例在最终短测重新通过；
不是删除失败断言来宣布通过。测试只声明实际执行的条目，未跑完整 lint/integration/E2E/发行矩阵。

### 原 6010 安全部署时序

10:16 后续：四库已全部结束，本轮 8,452 成功/41 invalid_media（coser 最后多 3 个已逐张确认的 JPEG 标记异常）。
依照下述顺序停止主应用、确认退出并完整备份后，仅替换为上述固定 c9f28d7a 候选；
AI fa7750b5 原容器、模型身份与 unlimited-memory 配置保留。主/AI healthy、restart=0，容器内 database-check passed，
更新后正式页面四库原终态/计数保留。没有重提/清理任务；细节与备份见[终态记录](FIX-2026-10-23-face-library-retry.md#后台跟进终态与单次更新-1016)。
这是已复现场景修复的本地部署，不关闭 1 GiB 压力余量/长期/发行风险。

09:31 刷新正式界面后，coser 原任务为 5,491/6,202（成功 5,462、失败 29），仍运行且没有任务级错误；
主/AI healthy、当前容器 restart=0、容器内 database-check passed。浏览器长时后台快照曾停在 08:54，
本次主动刷新后才记录上述真实时间点，不把旧页面当实时结果。
07:43 停写备份显示 coser 当时 attempt_count=2，之后取消限额重建已发生一次断点恢复；
再次中断可能消耗既有有限恢复次数，故本轮不为部署候选额外重启原 6010。

后续安全顺序：先核对原四库终态/剩余原因，再停止主应用并完成全数据备份，单次更新至上述固定候选，
保留 fa7750b5 AI、原身份和 `compose.ai.unlimited-memory.yaml`；核对健康/容器内完整性、原终态与人工状态后更新记录。
不要重复提交全量工作或为了验证这次回收而盲目重试仍未恢复的 38 张输入。
原实例未更新前不能宣称“6010 已运行内存修复”，也不能因候选局部通过而关闭质量/容量/长期/发行门禁。

以下原始排查时间线保留为历史，早期“未修复”描述不覆盖上方候选进展。

## 当前运行选择（2026-10-23 07:44 +08:00）

用户明确要求“不要限制内存了”。已为本地 6010 叠加 `compose.ai.unlimited-memory.yaml`，
取消主应用和 AI 容器的 RAM/swap 上限；保留 CPU、并发、PID、输入大小、超时、tmpfs 和只读/网络边界。
这是操作者运行配置选择，不是解码峰值修复，也不关闭既有默认预算/容量/发行门禁。
Docker Desktop VM 总量仍为 8,318,976,000 字节（约 7.75 GiB），没有调整 VM/宿主额度或保证不再整机 OOM。

先在无媒体/数据库挂载的临时 Docker Python 容器验证：创建时 0/0 对应 cgroup max/max，
256 MiB 合成内存分配可运行；`docker update --memory 0` 虽成功但保留旧 128 MiB，故不用于正式实例。
使用相同配对镜像重建两个原服务，无构建、模型变更、迁移、重复提交或源文件/人工整理变更。
重建前主应用已停止，完整备份复制结束后才恢复：
`/Volumes/extended/foliopath/backup-6010-unlimited-memory-20261023-GDxoRm`，含 ai.env、Compose 与 data。
仅对该停写独立备份检查：quick_check=ok，外键检查无行，4 库/8,719 媒体。

07:44 两服务 healthy、HostConfig.Memory=0/MemorySwap=0，AI cgroup memory.max/memory.swap.max=max/max，
主应用容器内 database-check passed。主镜像仍 cd1435af、AI 仍 fa7750b5。
新主容器 4cd42824749c…、AI f276fc1d60de… 的 restart=0 是重建后的新生命周期，不能抹去旧主容器 OOM/restart=1。
后续同项目更新必须带两个 Compose 文件，防止覆盖回默认限制；详见[运行说明](../ai-compose.md#操作者显式取消容器内存上限)。
下文固定 1 GiB 探针与 OOM 结论保留为历史证据，不套用为当前不限额配置的验收。
07:46 正式 UI 原任务已从停机前 3,195 递增至 3,213/6,202，成功 3,184、失败 29、任务级错误为空，
不是重新提交或清空进度。Compose 语法、定向 TestAICompose、完整 make arch-check、合成 Docker 取消上限检查通过；
仅配置/测试/文档变更，未重跑完整应用集成/E2E 或发行验收。

## 证据

- 2026-10-23 06:31:21 +08:00，原 6010 主容器 `f854040b354c…` 从 restart=0 变为 1。
  同版 cd1435af，AI 容器未重启。当前双服务 healthy、database-check passed，coser 断点恢复继续。
- Docker Desktop `Data/log/vm/init.log` 的 kmsg 明确指向该容器 CONSTRAINT_MEMCG：
  `Memory cgroup out of memory: Killed process 87057 (foliopath)`，anon-rss=1,033,216 KiB、file-rss=23,940 KiB。
  随后 daemon 记录 exitCode=137、manualRestart=false、restartCount=1。主容器额度仍为 1 GiB。
  这是主应用的内存上限，不是 AI 模型服务 OOM，也不是已有 scan-worker/SQLite 短读证据。
- 应用日志只见重新 application.starting，无新的 component_stopped/storage_code。
  重启后 `.State.OOMKilled=false` 不能覆盖上述内核历史记录，不得拿当前布尔值排除已发生 OOM。
- 同一时段处理区附近旧停写元数据有 33 MP 和约 102 MP、高压缩文件大小的 JPEG；这是排查线索，
  无逐图在途记录，不能据此断言是哪一张导致。主图解码峰值、native allocator/TLS 保留、Go 堆与长期增长须分开测量。

## 安全实施顺序

1. 保留当前任务/断点与源文件，禁止重复提交、手动循环重启、增大额度掩盖问题或 host 打开活跃 DB。
2. 使用临时、纯合成图片，在同版 runtime 与 1 GiB 限额下独立复现，记录 Go heap/native refs/cgroup memory。
   比较现实现与线程资源清理等不改变输出语义的候选，先证明原因；不能把真实媒体放进自动化回归。
3. 只有证据足够才改 canonical imagevips owner，并加合成资源与像素兼容/取消/异常回归。
   若需改变输入空间/重采样或进程边界，先走既有身份/ADR 合同，不能直接落地新部署单元。
4. 合成 Docker 验证通过后再考虑单次更新 6010，保留当前 operation、断点和人工整理。
   更新前仍须停写完整备份；未验证前明确保留开放风险，不标“已解决内存问题”。

## 06:52–07:00 独立合成初测（未完成修复）

纯程序生成高熵 JPEG，未使用真实媒体。临时工具/输入保留于
`/Volumes/extended/foliopath/face-memory-synthetic-lEGo8L`，基于同版 cd1435af runtime/Go native 构建依赖。
解码经 internal/files 只读打开 `/library` 合成树，完整调用生产 DecodeFaceImage 后按正式路径构建 RGB→NRGBA→PNG。
每个正式测量容器 1 CPU/1 GiB、swap=1 GiB（无额外 swap）、无网络、只读 root；大型测量 `/tmp`=128 MiB，与正式 Compose 一致。
合成 fixture 生成单独用 2 GiB 和 `/tmp`=512 MiB，不当作受测应用通过预算的证据。

| 合成输入/控制 | 次数 | cgroup memory.peak（字节） | 结果及边界 |
| --- | ---: | ---: | --- |
| 4672×7008 progressive，现实现 | 120 | 403,767,296 | exit 0、无 OOM；末尾约 140 MB，未复现无限线性增长 |
| 同输入，加 LockOSThread/ShutdownThread 实验 | 120 | 364,961,792 | exit 0、无 OOM；RGB SHA256 与现实现一致，仅降低部分资源，不能算 OOM 修复 |
| 8800×11000 baseline，现实现 | 5 | 366,067,712 | exit 0、无 OOM |
| 同 baseline，加线程清理 | 5 | 366,006,272 | exit 0、无 OOM；单帧峰值没有实质改善 |
| 8800×11000 progressive，现实现 | 5 | 1,035,448,320 | exit 0、无 OOM，但几乎耗尽独立 1 GiB 容器，完整应用没有安全余量 |
| 同 progressive，仅加 GOMEMLIMIT=256MiB | 5 | 986,591,232 | exit 0、RGB SHA256 一致，但余量仍不足，不据此修改生产配置或宣称通过 |

33 MP progressive 两组在每个采样点 vips.Mem=0、open refs=0，Go HeapAlloc 约 58.7 MB。
96.8 MP progressive RGB SHA256 均为 `38276a09c8aba0676f368ceeeeac97d125b3fb7202267ca1b039bdfacf9b18f7`，输出 1280×1600。
这说明现样本没有明显未关闭 ImageRef，但不排除完整应用的 native allocator/TLS 保留或 Go/native 峰值叠加。
独立探针未触发 OOM，不拿它冒称复现了 6010 的具体资产/完整负载。

额外发现：锁定的 govips v2.18.0 `LoadImageFromFile` 实际先 `os.ReadFile`，随后 `LoadImageFromBuffer`。
所以现有 `/proc/self/fd` 调用仍会整段读取压缩源，源码中“stream”注释不等于零拷贝/真正 native 文件流。
应复核缓冲及压缩源/原生解码并存峰值；不能直接换 thumbnail loader 或重采样策略而静默改变向量输入空间。
下一步优先在同一合成矩阵比较 native allocator 回收/完整应用保留堆与固定预算安全余量；仍不足时须明确资源拒绝或
经过身份/ADR 合同评审的 native 输入改造。当前未改生产代码、模型身份、内存额度或自动提交新任务。

初次大图 fixture 的 progressive 导出因诊断容器没有可写 `/tmp` 而失败；已在独立 fixture 生成环境修正后重跑。
该失败不计为生产解码 OOM或通过，不混淆 fixture 生成与应用预算。

上述均是 CLI 资源探针，未启动完整 HTTP/DB/扫描服务；临时镜像继承的 readiness healthcheck 会失败，
不是本次解码错误或应用联验结果。本次只报告实际 exit/OOM/peak 和像素比较，不声称完整服务通过。
六个已退出、名称/镜像/状态核验归本轮的临时测量容器已移除，合成输入/工具/镜像保留；原 6010 未重建或重启。
07:00 再核对，原主容器仍 restart=1、AI restart=0，双服务 healthy、database-check passed；
coser 续跑至 2,208/6,202（成功 2,181、既有 invalid_media 27），无新的任务级错误。

## 07:18–07:22 合成常驻堆与回收对照（已复现 OOM，仍未修复）

在同一临时工具增加有界 RETAIN_MIB（0–256）：预先分配、逐页触碰、通过 runtime.KeepAlive
保留纯合成 Go 字节，不声称等价于原应用所有常驻对象。输入仍为上述 8800×11000 高熵渐进式 JPEG，
调用未修改的生产 DecodeFaceImage 和 RGB→PNG 路径；同版 runtime、1 CPU/1 GiB、无额外 swap、
只读 root/library、无网络、/tmp=128 MiB。所有本轮 CLI 容器均关闭继承的 HTTP healthcheck。

回收实验组合为每次处理前 debug.FreeOSMemory + glibc malloc_trim(0)，处理过程中
LockOSThread/ShutdownThread，进程 GOMEMLIMIT=256MiB；仅临时探针，不进入生产代码。
它不能代表单个开关的因果效果，也不改变 JPEG shrink/像素输入身份。
镜像 manifest：`sha256:24a8ffa8745f03a2830761b4c9c217d9ed4cb3898632072ece980de43a75c94c`。

| 常驻堆 / 控制 | 请求次数 | 实际结果 | cgroup peak（字节） |
| --- | ---: | --- | ---: |
| 128 MiB / 当前路径 | 3 | exit 0、OOM=false，但触及完整上限 | 1,073,741,824 |
| 128 MiB / 回收组合 | 5 | exit 0、OOM=false，RGB 哈希不变 | 982,675,456 |
| 256 MiB / 当前路径 | 3 | 第 1 次输出后、后续处理中被杀，exit 137、OOM=true；不是 3 次通过 | 已打印 1,073,741,824 |
| 256 MiB / 回收组合 | 3 | exit 0、OOM=false、RGB 哈希不变，但仍触及完整上限 | 1,073,741,824 |

RGB 均与前述 `38276a09…` 相符，native 采样 refs=0/Mem=0。这是**可复现的合成组合 OOM**，
证明即使没有真实模型推理或未关闭 ImageRef，主应用大图解码与保留堆叠加也能超限。
不能据此认定已找出 06:31 的那张真实图片，亦不能排除原应用其他资源叠加。
回收组合虽在本组避免被杀，仍顶满上限，没有完整 HTTP/DB/扫描/浏览的安全余量，
不能当作通过方案或直接发布到 6010。下一步须验证可保留输入身份的解码峰值降低方案，或按合同定义
有界资源拒绝；若改变解码策略/输入空间或进程边界，先记录相应身份与 ADR 决策。

本轮只改临时探针和诊断/进度文档，未改生产代码、原 6010 镜像/额度/模型/任务。
07:22 原服务仍 healthy，主 restart=1、AI restart=0，容器内 database-check passed。
coser 2,716/6,202（成功 2,687、invalid_media 29），没有新任务级错误；新增两条未在最多 20 个早期样本中暴露，
不套用旧样本的精确 JPEG 标记原因。四库整体仍未完成。
四个本轮已退出探针容器经镜像/状态核验后删除；没有删除合成输入/工具/镜像，原 6010 容器未删除或重启。
