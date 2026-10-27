# CR-2026-053 / ADR-0039：原生 AI CI 清理权限修复

2026-10-27；POST-MVP-5 P6；Owner：发行验证；影响 NFR-INT-009/010 的测试隔离与清理，不改生产运行时、模型身份、Compose目录映射或作业行为。

失败流水线：40442626094（v1.6.0源码5ef8852）。两架构主镜像检查通过；两架构AI均完成186文件下载、取消恢复、离线重启及真实合成推理（640维文字/图片、2脸、3标签、4行OCR、空白图）。最终宿主Python清理临时bind目录时，runner无权删除容器root创建的模型文件，抛出PermissionError。未生成成功AI记录，也未推广主/AI镜像标签；不能将该流水线记为通过。

修复：仅测试自建的缓存改为唯一命名、生命周期明确的三个Docker临时卷。先删除测试容器，再由Docker移除这些卷，宿主不递归删除容器文件。不使用sudo、不忽略清理错误、不修改生产文件权限。复用外部隔离验证容器时仍不清理调用者资源。

回归：真实Docker创建root拥有且不可写的模型子目录，覆盖正常退出和测试异常两条路径，检查容器/卷都已删除、原始测试异常保留。`make test-managed-ai-cleanup FOLIOPATH_TEST_AI_IMAGE=<verified-image>` 为入口，原生镜像验证在同一实际镜像上自动运行该回归。

后续流水线结果以GitHub实际运行记录为准；本记录不将之前的推理通过等同于镜像已发布。

本地验证：真实Docker清理回归（正常/异常两路径）、实际AI镜像profile及安装测试、18项原生发布协议测试、make arch-check、make generate-check、make lint均通过。

修复提交 `bd4ace6`；[流水线40444109761](https://github.com/HappyQuQu/foliopath/actions/runs/40444109761)整体success。v1.6.1源码 `b4157b2`，AMD64/ARM64的主/AI四个验证任务全部通过，两个发布任务success；1.6.1/1.6.1-ai及latest/latest-ai均已推广。原失败记录保留，不重写为通过。
