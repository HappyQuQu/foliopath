# 子目录布局修复与发行依赖恢复

- 授权：用户接受本地 Docker 效果，明确要求合并 PR #64、发布新版本，并仅恢复构建和发行验证必需文件。
- 范围：既有 POST-MVP-5 / S5-007/S5-008；FR-BRW-001/004/005、FR-UI-001～003/006、NFR-UIF-001；发行输入完整性由 native release 工作流和精确 `.gitignore` 白名单负责。没有新部署单元、持久化或 API 变化。
- UI：标题独立、等宽子目录卡、两行默认展示、展开/收起和原游标分页；配色与侧栏不变。实现和验证见[修复记录](FIX-2026-10-27-directory-layout.md)。
- 前次清理 `1e50f4d` 后，流水线 [40460184407](https://github.com/HappyQuQu/foliopath/actions/runs/40460184407) 因缺失构建文件，四个原生镜像构建失败。保留此失败记录，不将其改成成功。

## 精确恢复范围

25 个文件全部与 `1e50f4d^` 的已跟踪内容逐字节一致；只恢复路径跟踪，没有改测试逻辑或跳过验证：

- `tests/release/glib_cve_2026_58016.c`：主镜像构建中的原有 GLib 验证输入。
- `services/inference/{ocr_native_tests,ocr_tests,test_face,test_runtime_manifest,test_scheduler,test_service,test_tag,test_tcp_disconnect}.py`：原 AI 发行配方已经显式复制的 8 个输入。
- `services/inference/test_model_install.py`：原 AI 发行镜像验证必须运行的安装器检查。
- `tests/release/{image_smoke,image_media_smoke,http_client,storage,compose_smoke,recovery_smoke}.sh`：主镜像原有媒体、部署、备份恢复验证链。
- `tests/release/{managed_ai_image,managed_ai_real,test_managed_ai_real}.py`：AI 原有身份/安装器/缓存清理与真实合成推理验证链。
- `tests/fixtures/http-client/Dockerfile`、`tests/fixtures/media-matrix/{Dockerfile,generate-release-fixtures.sh}`：上述主镜像 smoke 的辅助镜像与确定性合成数据生成器。
- `tests/fixtures/ai-faces/synthetic-pair.png`、`tests/fixtures/ocr/{bilingual,blank}.png`：上述 AI 真实合成推理的最小夹具。

普通 Go/前端测试、集成/E2E 目录、其他夹具、截图、日志及输出保持忽略。使用精确白名单正常 `git add`，没有整体恢复测试树或 force-add。

## 验证与发布

- 原本地前端 526 项测试、类型/架构/生成、生产与 Storybook 构建通过。
- 原本地 Docker ARM64 实例通过真实扫描、目录分页、缩略图、原图 Range、重启持久化、容器锁域数据库与静态包核验；390/768/1280/1920px 无横向溢出。独立命名卷与合成媒体；没有修改其他实例。
- 本轮 25 文件历史字节比对、shell 语法/Python AST、精确忽略规则、`make arch-check generate-check`、`git diff --cached --check` 通过。
- 只含待提交文件的干净源码 Docker ARM64 构建成功；其现有 `image_smoke.sh` 全流程通过，包含媒体格式、Compose、代理、完整备份恢复、同版本重启/WAL恢复及错误数据拒绝。不依赖未跟踪的普通测试文件。AI 发行配方 38 个直接输入在干净快照均存在。
- 正式原生 AMD64/ARM64 镜像结果将在流水线完成后补充；不提前声明发布成功。
