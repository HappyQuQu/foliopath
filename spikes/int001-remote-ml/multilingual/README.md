# 固定多语言模型本地验证（ADR-0025）

仅适配 `XLM-Roberta-Large-Vit-B-16Plus`，官方固定 revision 在 fetch_model.py 内。
179 文件、2,729,566,266 B，Git/LFS 检查后确定性生成 cache-baseline.json；这是部署工具，不是产品下载器。
权重必须置于仓库外，只读缓存保留 legacy clip/buffalo/JoyTag 完整原字节，新增 XLM 子树不替换旧文件。

先用 fetch_model.py 的 `--cache` 和 `--manifest` 明确外部缓存及本目录清单；可显式 `--proxy`，
不继承环境代理、不降级 TLS。再以已验证本地 legacy 镜像构建 services/inference/Dockerfile.multilingual，
用无网络 `python -m multilingual_contract` 得到新语义部署身份，用 legacy `image_identity` 保留 face/tag 身份。
`Dockerfile.app` 是固定本地 builder/runtime 的增量复现，不冒充完整发行构建或双架构。

设置 `FOLIOPATH_TEST_APP_IMAGE`、`FOLIOPATH_TEST_AI_IMAGE`、`FOLIOPATH_TEST_MODEL_CACHE` 后运行
`node tests/release/multilingual_semantic_smoke.mjs`。脚本只复制仓库合成 fixtures，测试预算由
compose.ai.multilingual-test.yaml 明确为 AI 4 GiB，主应用 1 GiB，并使用独立 scratch volume。
不要改成真实图片或现库 DB。数据保留在唯一临时目录，容器清理由独立项目名约束；无生产凭据。

默认小模型预算不因大模型通过变更。4 GiB 已实测触顶，不声称余量或一般中文质量通过。
实际候选、失败实验、现库部署和未完成门禁见 docs/changes/CR-2026-032-multilingual-semantic-model.md。
