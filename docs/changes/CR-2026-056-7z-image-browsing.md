# CR-2026-056：7z 图片浏览

- 2026-10-29 用户明确要求“支持7z”；分支 `codex/7z-image-browsing`。
- 目标：POST-MVP / ARCHIVE-S2，Owner：files（清单/解码/资源/临时流）、scanner/SQLite（既有来源与代次）、media（既有归档指纹与内容）、browse（格式标记和恢复文案）。
- FR-7Z-001：未加密单卷 7z 作为虚拟图集，保留层级/空目录/自然排序及既有浏览整理。
- FR-7Z-002：固实与非固实成员按请求读取，既有缩略图/原图/Range；不展开整包到磁盘。
- FR-7Z-003：坏包/加密/超限显示既有状态并保留旧索引；包内视频/嵌套和 AI 不准入。
- NFR-7Z-001：沿用 ZIP 大小/路径/数量限制；原生子进程有硬内存、CPU、墙钟、并发、输出额度与取消回收。
- 合同：[ADR-0041](../adr/0041-read-only-7z-images.md)、[归档来源合同](../architecture/zip-source-contract.md)、`api/openapi.yaml` 的兼容描述扩展；无 migration。

## Gate 与证据

S0：本记录和 ADR 先于实现。先验证真实清单、扫描、解码/Range、失败保留、取消和资源边界，再修改正式 browse 文案和 Story。固实随机/连续读取用合成包测量，结果实施后补充；测试与夹具保持本地，不扩大发行白名单。现有实例升级及正式发布以本次后续明确交付为准。

Backend Ready（2026-10-29）：真实 Linux arm64 / 7zip 25.01 的 `go test -count=1 ./internal/files ./internal/scanner` 与组合 `TestComposedSevenZipBrowseAndRange`、既有 ZIP 组合 HTTP 回归全部通过。覆盖固实/非固实中文目录/空目录、连续/倒序/重复内容读取、临时回收、加密头/成员拒绝、清单入口限制、进程等待取消、输出额度立即终止，以及真实认证、索引、自然名称顺序、Range、ETag和源变更拒绝。无 migration/API字段变化；允许消费正式 browse 组件。

## 实现与最终本地验证

- `internal/files/archive_7z.go` 拥有7z清单与内容读取。7zip/prlimit仅继承锚定原包fd，字面成员参数配合 `-spd`，单线程解码、两个活跃进程、256MiB地址空间、60秒CPU/墙钟、8MiB清单及64MiB实际图片输出。大小/CRC/来源版本核对后复用原有可seek临时流，关闭回收；既有ZIP路径与身份保持兼容。
- 真实大字典回归构造合法7z头、4GiB LZMA2字典声明，观察到解码器在256MiB地址空间额度内拒绝分配（实际退出码2），已映射为超限并验证失败临时文件删除。输出额度超限会立即取消并等待子进程结束。此为资源隔离测试，不把原生解码器称为安全沙箱。
- 原生Linux arm64真实测试通过：`go test -count=1 ./internal/files ./internal/scanner`；`go test -count=1 ./internal/store/sqlite -run 'Test(SevenZipScan|ZIPScan)'`；`go test -count=1 ./internal/app -run '^TestComposed(SevenZip|ZIP)BrowseAndRange$'`；files全包与组合HTTP另以 `-race -count=1` 通过。包含固实/非固实、字面通配字符文件名、中文/空目录、加密头/成员、CRC、Range/ETag、源更新、收藏/坏包保留、删除清理和四类AI候选排除。
- 16MiB、32个512KiB合成图片文件的固实包，4次顺序与4次随机读取实测均为几十毫秒级；这是本机arm64小型合成工作负载，不证明4GiB包、实际图片图集或222上的浏览延迟。原图不跨请求缓存，后续是否增加缓存需另接受额度/失效合同。
- 实际执行 `make fmt`、`CGO_ENABLED=0 make arch-check contract-check generate-check lint test test-integration`、`make test-e2e` 全部通过。OpenAPI兼容检查通过，无breaking changes；仅描述归档格式扩展，摘要与生成客户端从源更新，无migration。macOS不执行Linux原生7z解码测试，以上Linux证据单列。
- 前端 `npm run check:types`、BrowsePage 24项测试、完整 `npm run check` 通过：79文件539测试及审计/类型/生成/架构/Storybook。ZIP和7z卡片格式、封面加载失败后的导航均由生产组件测试；新增两个7z稳定Story。
- 生产Dockerfile最终Linux arm64镜像 `foliopath:local-7z-preview-20261029`，Docker image ID `sha256:9c11f84ef9fd50124d77c30d621b9b552ea8605a8081f9177a99516bfd1aaa84`。以该不可变ID执行既有 `tests/release/image_smoke.sh` 通过，含新增原生7z固实包清单/原图字节比对、媒体矩阵、认证LAN Compose、恢复与只读边界。本次只扩展现有白名单内的 `image_media_smoke.sh`，25项白名单不变。尚未执行原生amd64发行验证或正式发布。
- 本地6020试用镜像已更新，沿用独立Linux数据卷与合成媒体。真实界面显示21张图片（原9张加两份7z各6张），ZIP/7Z格式标记、封面、中文目录、空目录、预览/完整查看器1→2→10自然排序、加密包中英提示正常。390/768/1440/1920宽度截图及DOM读数均无横向溢出，实际缩略图/原图加载成功；英文深色错误状态另验，浏览器error日志为空。容器healthy、容器内database-check通过。
- 原有ZIP与新增合成7z逐文件SHA/字节校验不变，临时成员文件检查为空。所有测试源、夹具、截图、登录状态和日志留在本地 `/tmp/foliopath-7z-spike` 与既有合成预览目录，不入Git。预览为 `http://127.0.0.1:6020/libraries/lib_1/browse/dir_19`；222仍为既有正式v1.8.0，本轮不操作真实部署。

Gate：S0 Accepted → Backend Ready → 本地生产组件纵向验收完成；正式发布仍须原生双架构验证。代码保留在独立分支，尚未提交/推送/发布。

最终复查修正7z失败读取时的空接口返回值，追加失败返回nil断言；Linux files全包与组合ZIP/7z HTTP再次以 `-race -count=1` 通过。随后重建并仅使用上述最终不可变image ID复跑完整image_smoke，全部通过，6020已切至同一image ID。

## 后续正式发布

2026-10-29 用户随后要求提交、推送、合并及发布。此项已随v1.9.0正式发布，main/AI双原生验证与镜像推广均通过，见[发行记录](RELEASE-2026-10-29-v1.9.0.md#正式发布结果)。上文“本地分支/尚未发布”为当时实施阶段状态；不改写历史验证和限制。222未在此发布中升级。
