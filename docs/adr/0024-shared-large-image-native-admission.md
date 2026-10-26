# ADR-0024：大图 native 工作内存共享准入

- 状态：Accepted for implementation，2026-10-23；不等于所有容量/长期/发行门禁通过。
- 范围：FR-INT-009/016、R-006/R-024，POST-MVP-5 维护；依据用户要求修复人物解析与主应用内存。
- Owner：imagevips native runtime；app 继续拥有既有 worker/资源组合，jobs 继续拥有任务与租约。

## 背景

同版依赖下，纯合成 96.8 MP 高熵渐进式 JPEG 与 256 MiB 保留堆可使主应用解码路径超过 1 GiB。
只在人物解码前后回收虽然通过独立重复探针，但完整 HTTP/扫描/缩略图/人物联验仍 OOM：
各功能自己的单 worker 不等于整个进程只有一个大图 native 工作集。Go GC 不完整计量 libvips/libjpeg 内存。
用户已选择取消本地容器上限；这个部署选择不能代替代码回收/准入，也不改变默认容量门禁。

## 决定

1. libvips 是进程级 runtime；imagevips 唯一拥有共享的单槽大图工作集准入，不能按媒体库、功能或 Processor
   各建一个。>=16 MiB 压缩源、>=16 MP 图片或未知头进入此槽；小图不因此调整既有后台并发配置。
   等待由已有有界 worker/内容请求拥有，不创建新 goroutine/部署 worker。等待支持 context 取消，不持 SQLite 事务。
2. 缩略图、人物、远程语义/自动标签预处理、旧语义预处理复用同一个 owner；在整段压缩源分配/native load
   之前取得槽，在 ImageRef 关闭、线程资源清理、Go/native 空闲页回收之后释放。
   使用 LockOSThread/ShutdownThread，Linux glibc malloc_trim 与 Go FreeOSMemory 配合，不设置 GOMEMLIMIT 或新 RAM cap。
3. 只读头探测仅用于资源调度：稳定文件/bytes.Reader 最多读取 1 MiB 并恢复原位置；自定义取消/无进展 reader
   不被额外读取，直接使用保守槽，保留原 canonical read 的取消/空读预算。探测失败不自动接受或改判格式。
4. 不改变像素、shrink、滤波、错误容忍、模型身份/代次、原件、人工整理、任务状态机或公开 API。
   不通过宿主 mmap 活跃 SQLite，不对可被外部截断的原媒体直接 mmap，不增加持久输入副本或新的部署单元。
5. 不把单次测试后的 OOMKilled=false 当历史无 OOM。合成联验须检查自动重启计数；显式恢复重启另记。

## 替代方案与后果

- 单独加线程清理/提高上限/只改 Go GC：未解决完整应用并行 native 工作集问题，不能据此关闭风险。
- 磁盘快照 mmap：单独候选仍 OOM，又增加输入临时落盘/故障边界，本轮不采用。
- 改 JPEG shrink/模型输入或接受损坏码流：会改变输入身份或安全语义，不为本次回收修复静默实施。
- 大图吞吐会串行，显式 GC/trim 有成本；需要后续更大容量和浏览并发测试，不能宣称所有 500 MP/4 GiB 场景已通过。
- 资源探针与完整合成 Docker 都是局部回归，不替代真实人物质量、双架构、长期或发行验收。

## 证据入口

[内存修复记录](../changes/FIX-2026-10-23-main-face-memory.md)、`memory*_test.go`、
`spikes/int001-remote-ml/facememory`、`tests/release/face_diagnostics_smoke.mjs` 的固定合成大图模式。
