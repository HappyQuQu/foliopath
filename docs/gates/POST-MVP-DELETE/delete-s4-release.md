# DELETE-S4 发布验收

2026-10-30；FR-DEL-001～010（以 CR-2026-060 永久删除修订为准）、FR-BRW-001/004/016/017；目标 v1.10.0 / POST-MVP DELETE；Owner：release / files / deletion / Web。用户明确授权更新 README、提交、推送、合并、发布。

本阶段使用 ADR-0044 的主进程、openat2 和单媒体挂载；不回引 ADR-0043 的 Landlock，不增加 PVE、内核或部署单元。原 S1～S3 回收设计及历史验证留档，当前能力为一次弹窗后的永久删除，回收 UI/新准入已移除。

## 验收计划与来源

- 工程：完整 Go 格式/架构/生成/lint/unit/integration、runtime E2E，固定 Node/npm 的完整 Web check；本地输出不提交。此前真实 Linux 合成删除、强杀恢复与通知链见相关修复记录。
- 容量：沿既有 release/capacity_smoke，Linux amd64、4 CPU / 4 GiB 的独立 100k 媒体 / 10k 目录，原预算不放宽。另用独立合成树比较 rm -rf 和网页永久删除（1 万、10 万非媒体文件）、深目录、多个顶层与 200 项提交，记录入队、完成、浏览延迟及内存。回收已撤回，不再要求该产品路径的性能承诺。
- 升级：沿原 upgrade_rollback_smoke 验证旧 schema 48 到新 49、同版本重启以及备份配对回退，原件保持只读。
- 镜像：Release Please 自动版本和正式原生 amd64/arm64 main/AI 四个构建验证；每架构构建一次，同 digest 核验后保留 SBOM/provenance 推广。任一失败不得宣称发布完成。

原生配对结果在自动流水线结束后记录；详细发布提交、PR、版本、摘要及证据见[发行记录](../../changes/RELEASE-2026-10-30-direct-deletion.md)。
