# 可选 AI 双服务部署入口

## Docker Desktop数据卷覆盖

6010本轮修复已使用第五覆盖`compose.ai.data-volume.yaml`，由独立停写备份复制并核验完整data后切换。其`FOLIOPATH_DATA_VOLUME`必须是已准备的外部卷；四个既有覆盖仍必须全部加载。原件仍只有单一`/library:ro`，AI不接触数据库。切换后的状态以[本轮记录](changes/FIX-2026-10-26-ocr-limits-sqlite.md)为准；后续命令须追加`-f compose.ai.data-volume.yaml`，不可启动旧宿主data副本。详见[ADR-0038](adr/0038-docker-desktop-linux-sqlite-volume.md)。


## 已核验本地增量：OCR显式部署与终态验收

6010当前以已核验OCR overlay `services/inference/Dockerfile.ocr`提供独立PP-OCRv5_server；固定镜像和身份见[部署记录](changes/DEPLOY-2026-10-26-ocr-6010.md)。私有`FOLIOPATH_AI_DEPLOYMENT`新增完整`ocr.identity`，不替换semantic/face/tag身份。新增`FOLIOPATH_OCR_CACHE_PATH`指向已有、按`services/inference/ocr-manifest.json`核验的独立模型缓存，第四覆盖`compose.ai.ocr.yaml`只挂AI `/ocr-cache:ro`，不创建第三服务或公开端口。目录必须存在，禁止将媒体或数据库目录用作OCR缓存或验证scratch。

本地6010后续状态、启停、备份和更新须保留原项目名、私有env及全部五份Compose（base及四个覆盖），例如只读查看：

```sh
docker compose --project-name foliopath-ai-original-review --env-file ai.env \
  -f compose.ai.yaml -f compose.ai.unlimited-memory.yaml \
  -f compose.ai.multilingual.yaml -f compose.ai.ocr.yaml \
  -f compose.ai.data-volume.yaml ps
```

不要省略原不限内存、多语言、OCR或data-volume覆盖，不要输出含私有部署配置的完整Compose结果到公共日志。配置OCR身份不会自动开库或提交任务；当前仅pixiv由管理员启用，补齐已终态998成功／1坏JPEG源／0待处理，原984成功及全部历史结果／任务保留。其余库继续关闭，原10个源问题不操作。坏源不标成功、不循环重试；以后恢复原件后再显式补齐。下文仅使用base的命令是最小profile示例，已经叠加其他profile的实例必须始终追加其全部原有覆盖。


## 本轮全功能AI候选的原生打包入口

本轮新增 `services/inference/Dockerfile.release`，从既有固定多架构上游重现当前多语言／153类标签／OCR profile的41份Python与JSON。来源为仓库canonical文件，不以本地ARM overlay或脱离源码的runtime导出作唯一构建源；模型权重不打入新增层。原Python、OCR逻辑代次常量、模型／输入输出身份保持不变，旧overlay配方保留历史用途。

本地ARM64与原生AMD64打包均与当前7739的41份runtime及OCR身份逐字核对通过；两个架构各8项真实模型端点和实际OCR任务／同job续跑已通过，原五服务维护恢复经独立复核。模型／输入输出身份保持。本地双架构OCI保存SPDX与SLSA，但AMD AI曾重复cached export，严格build-once发布门不声称通过；未提交Git来源、正式CI promotion与发布批准仍未完成，功能结果不等于准确率。完整候选范围和结果见[统一验证](changes/VERIFY-2026-10-26-release-candidate.md)。本轮未提供新的公开AI镜像或模型下载地址；不得把本地image ID当作已发布的registry引用。

## 独立标签 PNG 输入（ADR-0029）

标签大PNG修复使用部署JSON中独立的`tag.image_preprocessing: "png-v4"`，而不是只修改顶层语义字段。
空值保留原v3；合法新值绑定新标签身份和同一imagevips预处理器。保留原tag.identity、92类、AI镜像和
语义/人脸配置，不同时启用下文153类策略。需先完整停写备份、合成Docker验证，再切换并让已启用库按新身份补齐。
人工标签和移除抑制保留；旧run不计新覆盖，坏源不能标成功。具体测试/6010状态以[本轮记录](changes/FIX-2026-10-24-autotag-png.md)为准。

## 可选 PNG 输入 profile（ADR-0028）

已验证的新主应用支持在私有 `FOLIOPATH_AI_DEPLOYMENT` JSON 中增加 `"image_preprocessing":"png-v4"`。
省略保持旧 v3；不要替换模型/face/tag identity。新 profile 仅扩展语义 PNG 至 64 MiB / 40 MP，仍有界解码；
AI 无需重建，自动标签输入限制不变。切换会建立新语义代次，必须在现有任务终态、停写全备份和明确重算授权后执行。
旧向量不挪用、不清理；6010已于2026-10-24完成授权切换及四库语义终态，8690成功、10源失败。保留原三份Compose覆盖。
详细步骤、固定镜像与验收边界见[修复记录](changes/FIX-2026-10-23-semantic-large-png.md)。

## 已核验本地增量：153 类标签（CR-2026-033）

中文/英文/双语是浏览器展示偏好，不增加 Compose 字段或模型商店；原 92 类结果也能通过新主应用展示。
153 类识别须使用 `services/inference/Dockerfile.photo-tags` 覆盖固定多语言镜像：

```sh
docker build --pull=false -f services/inference/Dockerfile.photo-tags \
  --build-arg FOLIOPATH_INFERENCE_BASE=foliopath:local-multilingual-20261023@sha256:3631b43219c74bc4e18df503d53e144e9bc86f80bc9490efa1d1e65d356ea903 \
  -t foliopath-inference:photo-tags-v2 .
docker run --rm --network none --entrypoint python foliopath-inference:photo-tags-v2 -m photo_policy
```

该 profile 只把最后命令的结果填入 `FOLIOPATH_AI_DEPLOYMENT.tag.identity`，不要用旧 `image_identity --tag`
替代（其有意保留 legacy 身份）。保留其他 identity、模型缓存、原媒体只读挂载及当前 multilingual/unlimited
覆盖；更新遵循停写全备份、AI healthy 再启动主应用，不用重建/清理真实库作为验证。新身份会补齐已启用库，
不是“翻译需要重跑”。候选/测试/6010 是否更新以 [CR-2026-033](changes/CR-2026-033-localized-photo-tags.md) 为准。

POST-MVP-5 / P5 / ADR-0019、0020。根 `compose.ai.yaml` 是独立双服务配置，**不要与 `compose.yaml` 叠加**。这是交付候选，不是镜像已公开发布或质量已校准；不自动修改现有部署。

## 准备

### 构建配套镜像

从同一份经审核的源码快照分别构建两个镜像，不推送、不运行线上更新脚本：

```sh
docker build --pull=false --build-arg VERSION=p5-delivery -t foliopath-app:p5-delivery .
docker build --pull=false --build-arg FOLIOPATH_INCLUDE_FACE=1 \
  -f services/inference/Dockerfile -t foliopath-inference:p5-delivery .
docker image inspect foliopath-app:p5-delivery --format '{{.Id}}'
docker image inspect foliopath-inference:p5-delivery --format '{{.Id}}'
```

将得到的两个不可变 ID 写入配置，而非把可变构建标签当发布凭据。`FOLIOPATH_INCLUDE_FACE=0` 可只构建 CLIP 运行服务，此时部署 JSON 不得含 `face`。当前主应用根 Dockerfile 已包含远程服务组合，不需要开启旧内置 ONNX 编译路线。构建过程可能需要下载固定依赖，`--pull=false` 不代表离线构建；离线运行与离线源码构建是两种要求。本轮交付验证前者，不承诺全新机器无缓存离线编译。

构建网络需要代理时，可对 `docker build` 显式传入 `--build-arg HTTP_PROXY=http://proxy-host:port --build-arg HTTPS_PROXY=http://proxy-host:port`；保持证书校验。此为构建依赖下载代理，不是 AI 运行配置，也不是旧模型商店的代理设置；不要把凭据写入 Dockerfile 或公开日志。运行 Compose 不继承这两个 build arg。

向离线机器传递镜像时，在有镜像的机器执行 `docker image save -o foliopath-p5-images.tar foliopath-app:p5-delivery foliopath-inference:p5-delivery`，另记录文件 SHA-256、两镜像 ID、架构和源码快照；接收端先核对归档摘要，再 `docker image load -i foliopath-p5-images.tar`，核对镜像 ID 后运行。归档包含镜像，不包含模型缓存、媒体或用户数据库。不要将 arm64 单架构镜像冒充 amd64 镜像；正式多架构发行归 P6。

### 模型与目录

准备同一候选版本的主应用镜像、专用 AI 镜像、与 AI 镜像内清单精确匹配的模型缓存。先加载/核验镜像；Compose 使用 `pull_policy: never`，不会自动拉取或下载模型。离线主机须先通过受信路径取得镜像和模型；不提供未经核实的下载地址。

显式设置以下环境变量（也可放入专用、限制读取权限的 env 文件并通过 `--env-file` 指定）：

可从 [`deploy/ai.env.example`](../deploy/ai.env.example) 复制到私有 `ai.env` 后填写；模板必填值故意为空，未填写时 Compose 失败关闭。根 `ai.env` 已从 Git 和 Docker 构建上下文排除，但自定义文件名仍需自行防止提交或传入构建上下文。

| 变量 | 要求 |
| --- | --- |
| `FOLIOPATH_IMAGE` | 已加载且核验的主应用 `repository@sha256:…` 或本机 `sha256:…` image ID；不要使用可变标签 |
| `FOLIOPATH_AI_IMAGE` | 配套 AI 镜像的不可变引用；不能替换为原版 Immich 镜像 |
| `FOLIOPATH_LIBRARY_PATH` | 已存在的单一媒体根目录，只读；其下不得有嵌套挂载 |
| `FOLIOPATH_DATA_PATH` | 已存在、专用、可写的本地数据目录；不要与另一个实例同时使用 |
| `FOLIOPATH_MODEL_CACHE_PATH` | 已存在的匹配模型缓存，只读；不是媒体目录 |
| `FOLIOPATH_AI_DEPLOYMENT` | 单行 JSON，沿用 [启动配置合同](deployment.md)；`origin` 必须为 `http://machine-learning:3003` |
| `FOLIOPATH_BIND_ADDRESS` / `FOLIOPATH_PORT` | 默认 `127.0.0.1` / `8080`；经认证可信 LAN 才显式改绑定地址 |

JSON 的 `identity` 必须从已核验的固定 AI 镜像离线读取，不从运行中的 HTTP 服务首次信任：

```sh
docker run --rm --network none --read-only --entrypoint python "$FOLIOPATH_AI_IMAGE" -m image_identity
```

将输出作为 JSON 的 `identity` 对象，补 `origin`、与实际模型对应的 `model_bytes`、真实 `license_id`；字段含义见部署合同，未知许可不能冒称已批准。人脸是可选 `face` 对象，必须明确填写模型摘要和阈值合同；不能将 P3 测试阈值当作生产质量结论。Compose 本身不注入人脸测试阈值，也不恢复模型商店。

基础镜像支持 `ViT-B-32__openai`（512 维），ADR-0025 的可选覆盖镜像支持
`XLM-Roberta-Large-Vit-B-16Plus`（640 维）；人脸构建可选 `buffalo_l`。
不是任意 HF/ONNX 加载器，旧 SigLIP/512 维索引不冒充新结果。
基础清单 `/opt/foliopath-inference/cache-manifest.json`，多语言独立清单 `multilingual-manifest.json`，
分别来自 `spikes/int001-remote-ml/cache-baseline.json` 与 `multilingual/cache-baseline.json`。
这是字节核验，不是授权/来源签名或发行批准；权重不随镜像公开发布。只读树中未列入清单的文件不进入推理快照。

### 多语言覆盖镜像（ADR-0025）

`services/inference/Dockerfile.multilingual` 的 `FOLIOPATH_INFERENCE_BASE` 必须是核验过的
legacy `repository@sha256:…`；Dockerfile FROM 不接受裸 `sha256:…` image ID。
新语义预期身份通过无网络镜像命令 `python -m multilingual_contract` 独立读取，
填入私有部署 JSON 的 identity，model_bytes=2729566266。保留 face/tag 原身份，不能以新语义身份覆盖它们，
也不能从在线服务反填受信摘要。模型和权重不在界面下载/选择。

固定 179 文件约 2.73 GB（十进制）。`spikes/int001-remote-ml/multilingual/fetch_model.py`
仅获取固定源/revision 并核对 Git/LFS 摘要，缓存置于仓库外；旧模型保留供 legacy 字节合同校验。
必须叠加 `compose.ai.multilingual.yaml`，将 `/verified-cache` 改为专用可丢弃 disk scratch volume，
不是媒体、DB 或模型真相；启动拒绝未知内容/symlink，仅清理已知遗留工作目录。
该覆盖不改变 RAM/swap。历史大模型验证使用过独立 4 GiB AI 测试预算，不是基础 2304 MiB 通过，
也不是程序固定上限；当前模型的实测与资源预留建议见下文。
原 6010 用户选择不限额：base + unlimited-memory + multilingual，RAM/swap 仍为 0。

有在途任务时先停主应用完成全数据备份；**先更新 AI 并等 healthy，再启动主应用**，
防止已恢复的标签任务早于 AI 就绪后终止。不为核心浏览添加 depends_on。
激活后逐库提交新空间的图片语义 missing，旧结果保留；运行中不标完成。
详见[本次验证与切换记录](changes/CR-2026-032-multilingual-semantic-model.md)。

## AI 功能内存参考

下面先给资源预留建议，再列实际观测。主应用与AI服务分开运行；模型推理内存不等于主应用常驻占用，
也不等于整台机器需要的内存。GiB按1024³字节计算。

| 启用的AI功能 | 建议先为AI预留 | 使用方式 |
| --- | ---: | --- |
| 人物分析（`buffalo_l`）或153类自动标签（`JoyTag`） | 约2 GiB | 单类任务的工程余量参考 |
| 图片文字识别（`PP-OCRv5_server`） | 约3 GiB | 包含模型加载与临时工作内存 |
| 中文语义搜索（640维XLM模型）或包含四类能力的多语言配置 | 约6 GiB | 按较大的语义模型预留，不把各功能峰值相加 |

这些是**工程余量建议，不是实测最低门槛、程序硬上限或稳定性保证**。主应用另外先预留约512 MiB–1 GiB，
扫描、缩略图和大图处理可能需要更多；再给操作系统、Docker虚拟机及其他服务留出空间。
专用机器可从8 GiB整机内存考虑；与Immich等多个服务同机时，16 GiB更宽裕，仍需结合实际工作负载调整。

2026-10-26固定镜像的CPU小样本实测如下。使用合成人物、文字和空白图片，推理严格串行；
人物、标签和OCR专项各自重新启动独立AI容器，并未清空系统文件缓存。

| 测试场景 | 原生平台 | AI容器内存高水位 | 当次AI测试限额 |
| --- | --- | ---: | ---: |
| 中文/英文/图片语义、人物、153类标签及OCR完整8项端点 | AMD64 | 2.82 GiB（3,031,310,336字节） | 5 GiB |
| 人物专项，实际检出2张人脸并返回识别向量 | AMD64 | 0.85 GiB（914,399,232字节） | 5 GiB |
| 153类标签专项 | AMD64 | 0.74 GiB（799,170,560字节） | 5 GiB |
| OCR文字及空白图片专项 | AMD64 | 1.42 GiB（1,525,428,224字节） | 5 GiB |
| 实际OCR任务、同任务续跑与搜索验证 | AMD64 | 1.43 GiB（1,539,919,872字节） | 5 GiB |
| 同一组完整8项端点 | ARM64 | 3.86 GiB（4,139,548,672字节） | 4 GiB |
| 独立的实际OCR任务及续跑验证 | ARM64 | 2.12 GiB（2,273,927,168字节） | 4 GiB |
| 最终正式CI候选 `rc-20261026.7` 的完整8项端点 | ARM64 | 3.92 GiB（4,210,307,072字节） | 6 GiB |

上述成功场景的cgroup内存上限触发、OOM和OOM kill计数均为0。AMD64测试目标原本配置8 GiB，未修改PVE资源。
新空主应用实例在原生AMD64仅完成启动、健康和数据库检查时，观测高水位37 MiB；这不代表图库扫描或图像处理峰值，
不能拿空实例占用替代资源预留。精确镜像、字节及测量边界见[内存实测记录](evidence/int-001/ai-memory-reference-2026-10-26.json)。

表中高水位来自容器`memory.peak`，包含进程、模型文件缓存和临时工作内存，不是单进程RSS。
完整8项端点中的每次读取都是该容器启动以来的累计高水位，不能把后续人物、标签和OCR的累计值当作各自独立用量。
不同场景、匿名内存与文件缓存的各自最大值也不能直接相加；平台和缓存状态不同，观测会变化。
历史英语512维模型的旧记录不作为当前中文640维、153类标签及OCR配置的内存需求。

4 GiB、5 GiB、6 GiB都是当次测试容器限额，**不是程序对AI功能的固定硬上限**，也不证明整台4 GiB机器足够。
操作者现有不限内存选择保持；宿主/虚拟机总内存仍约束进程。输入大小、调度、取消和推理期限保护继续有效。
早先3 GiB中文请求返回503的失败记录保留；后续5 GiB成功不能单凭这一差异就称为已证实并修复OOM。
本轮未运行容量、24小时混合负载或正式准确率评测，不据此作最低配置、并发或长期稳定性承诺。

## 启动与检查

```sh
docker compose -f compose.ai.yaml config --quiet
docker compose -f compose.ai.yaml up -d
docker compose -f compose.ai.yaml ps
docker compose -f compose.ai.yaml logs --tail 50 machine-learning
```

`config --quiet` 仅检查配置语法，不证明镜像/模型一致或推理成功。完整 `config` 会包含部署 JSON 和目录，不应贴入公开诊断材料。打开所配置地址，首次创建管理员，再在设置中添加媒体库、启用所需 AI 功能。AI 故障不阻止主应用初始化或普通图库；AI 健康也不等于实际任务或质量已验收。

两个容器根目录只读、禁提权且丢弃 capabilities。AI 只在内部网络，不挂载媒体或数据库；模型启动时校验并复制至私有 tmpfs，不自动联网补模型。媒体与缓存路径不存在会失败，避免拼错目录后自动创建空目录。

预算为主应用 1 CPU/1024 MiB、AI 3 CPU/2304 MiB，均不使用 swap；AI 1200 MiB 私有快照和两个容器临时目录均计入各自内存限制，宿主仍需保留系统余量。这是小样本预算，不是大规模容量承诺。旧单容器安装入口保留，不会自动转为双服务。

在这组 1 GiB 主应用预算下，从“配置”把**后台并发**设为 `1`，再同时启用智能搜索和人物分析。
每个 AI worker 自身虽为单 worker，但后台并发 `2` 会允许两条 libvips 预处理链路并行；真实大 JPEG
全库复验中该组合两次以 exit 137 被 Docker 恢复。设为 `1` 后两类任务通过既有全局 admission
串行执行，前台内容读取仍使用独立额度。不要靠关闭重启策略或扩大 swap 掩盖峰值；若改回 2～4，
必须在实际库和目标内存上重新完成联合容量与稳定性验证。

### 操作者显式取消容器内存上限

2026-10-23 用户要求本地 6010 不再限制容器内存。可在原配置之后叠加专用覆盖文件：

```sh
docker compose --env-file ai.env -p foliopath-ai-original-review \
  -f compose.ai.yaml -f compose.ai.unlimited-memory.yaml config --quiet
docker compose --env-file ai.env -p foliopath-ai-original-review \
  -f compose.ai.yaml -f compose.ai.unlimited-memory.yaml up -d
```

覆盖只将两个服务的 `mem_limit`/`memswap_limit` 置 0，不增加 CPU/并发，不移除 PID、输入大小、
推理期限、tmpfs 或只读/内部网络保护。后续更新仍须使用两个 `-f`，否则会恢复基准限制。
已有容器需要重建才可可靠清除原上限；当前 Docker 实测 `docker update --memory 0` 会保留旧值，
不能以命令 exit 0 冒称已取消。更新前保留停写完整备份，不删卷，不重复发起补齐任务。

应核对两个容器 `HostConfig.Memory=0`，并在 AI 容器确认 cgroup `memory.max=max`。
没有单容器限额不等于无限物理内存：Docker Desktop VM/宿主总量和可用 swap 仍约束进程，可能影响其他应用或发生整机 OOM。
本机这次是操作者选择的运行配置，不是默认 1 GiB/2304 MiB 预算的修复、容量通过或发行批准；
基准 Compose 与资源验收不变。需要恢复基准时，在安全停写/备份后仅使用 `compose.ai.yaml` 重建。

## 模型异常与恢复（F02）

| 现象 | 检查与处理 |
| --- | --- |
| Compose 报 bind 源目录不存在 | 核对媒体、数据、模型目录变量；不要通过新建空模型目录冒充修复 |
| AI 反复退出，日志为 `InvalidCache` | 停止 AI；用可信原始包恢复与镜像内清单匹配的缓存。缺文件、长度/摘要不符、符号链接均拒绝；不关闭校验，不在线自动下载补齐 |
| AI healthy，但模型显示不可用 | 核对主应用 `FOLIOPATH_AI_DEPLOYMENT` 与已核验镜像离线身份、模型配置及服务 origin；健康检查不证明兼容。恢复正确配置后重建主应用 |
| 更正目录或 JSON 后仍使用旧配置 | `restart` 不载入新环境变量或挂载源；使用同一项目配置的 `up -d` 重建对应服务 |

操作顺序：先记录配对镜像和配置，再 `docker compose -f compose.ai.yaml stop machine-learning`，在容器外恢复模型缓存或更正目录；执行 `docker compose -f compose.ai.yaml up -d machine-learning`。若改主应用身份 JSON，另执行 `docker compose -f compose.ai.yaml up -d foliopath`。每次均使用原项目名和 env 文件，避免新建另一个实例。

正常情况下主应用每 30 秒重新核验模型；等待服务 ready 和状态刷新后再重试失败任务。旧失败历史不会伪装为成功。同身份恢复应保留代次、已分析结果和人工数据；换模型或预处理身份应走新代次，不得手改数据库绕过。AI 无法服务时，普通图库仍可浏览，新 AI 任务返回 `model_unavailable` / `face_model_unavailable`。公共接口只提供安全的不可用提示，不公开宿主路径、内部堆栈或模型向量；运维区分具体原因需检查受信配置与容器日志。

此模式预加载固定缓存、AI 无外网，不需要模型下载代理。旧模型商店的代理配置不用于自动修复此缓存。模型许可、阈值质量与公开发布批准仍单独处理。

诊断时注意：图片语义设置 API 在模型离线时将 `activeGenerationId` 返回 null，表示当前不可用于新任务，不代表删除了持久化代次；已记录覆盖率仍应保留。恢复同身份后应重新出现原代次。人脸设置 API 的代次字段含义不同，不能用同一条“离线必须返回原 ID”规则判断两者。

## 停机、升级及恢复边界（配对备份）

下面假设配置保存在专用 `ai.env`，固定项目名 `foliopath-ai`。所有命令必须复用这两个值。`ai.env` 包含部署路径/身份，限制读取权限，不上传支持包。数据库目录不可与旧单容器同时使用。

1. **备份**：暂停创建新任务，执行 `docker compose --env-file ai.env -p foliopath-ai -f compose.ai.yaml stop foliopath`；通过 `ps -a` 确认主应用已退出，再复制整个数据目录到一个全新、仅管理员可读的备份目录，同时保存旧 `ai.env`、两个镜像 ID、架构及匹配模型缓存清单。不要复制正在写入的 SQLite 主文件而遗漏 WAL；不要修改媒体根。详见 [备份与恢复](deployment.md#备份与恢复)。
2. **升级**：预加载并核验新主应用/AI 镜像及缓存，更新 `ai.env` 的配对引用和离线身份。执行 `docker compose --env-file ai.env -p foliopath-ai -f compose.ai.yaml up -d`；核对两个服务健康、登录、媒体库、人物/标签和 AI 状态。身份或预处理变化会建立新语义代次，须重新分析；旧完成数不能冒充新代次进度。
3. **失败时回退**：先停止主应用，将升级后的数据目录保留用于诊断，**不要覆盖或删除它**。从停写备份复制到另一个全新恢复目录，恢复旧镜像/身份配置，将 `FOLIOPATH_DATA_PATH` 指向该恢复目录，再执行相同项目的 `up -d`。核对旧代次、人物/人工标签和媒体浏览；禁止让旧镜像直接打开已升级的库。
4. **确认**：重启后再次核对。不要因一次健康检查成功就删除备份。只有用户决定保留期后才清理备份和升级失败副本，既有副本中的人工信息不会被在线“清除”操作自动擦除。

单纯 `restart` 不载入变更后的环境变量或挂载源，需要 `up -d` 重建。迁移 39/40 的 service 模型记录不支持强行有损降级；恢复必须配对备份。未配置 AI 的旧单容器入口仍可独立使用；从单容器迁到双服务也应先停旧进程、做备份，不能把两套 Compose 同时指向同一个数据目录。本轮回退证据只证明选定两版镜像及合成旧库，不承诺任意历史数据库无条件兼容。

### 数据清除与残留边界

“清除派生人脸数据”删除当前库的观察/向量等可重建数据，保留人物和人工锚点，恢复分析后可重新关联。“清除人工关系”是另一项二次确认操作，删除当前库的人物关系/约束/相关审核数据，但人物实体、媒体原件和人工标签不会一起删除。不能把这两个操作描述为“删除所有人物信息”。

SQLite 使用 secure_delete；活动库的物理残留验证还要求 WAL checkpoint，单个任务显示成功不承诺历史备份、卷快照、文件系统日志或旧导出中的字节被擦除。在线诊断不显示向量/查询/姓名；经认证的人物管理 API 必须显示用户输入的姓名，此为业务返回，不是日志泄露。

图库通用缩略图和原媒体不属于人脸派生清除范围，仍可能包含人的画面。当前人脸推理不持久化独立裁剪缓存；清除观察/向量不能宣称从磁盘抹除所有包含人脸的图片。数据库检查应使用停写后的完整 DB/WAL 副本，不要用忽略 WAL 的只读方式得出错误结论。

两类清除都会使当前人脸结果不再处于可用投影状态；此时人脸 API 可返回 `face_not_ready`，需按界面重新开启/分析。这一状态本身不证明底层向量已删除：派生清除删除观察行，人工清除则保留观察，必须按各自操作语义判断。

## 本地验证

```sh
FOLIOPATH_TEST_MODEL_CACHE=/absolute/path/to/copied-model-cache \
  node tests/release/ai_compose_smoke.mjs
```

默认使用上文构建的 `foliopath-app:p5-delivery` / `foliopath-inference:p5-delivery`，也可用 `FOLIOPATH_TEST_APP_IMAGE` / `FOLIOPATH_TEST_AI_IMAGE` 选择预加载候选。脚本先解析不可变 image ID，再运行。只使用仓库合成媒体，在独立临时目录创建数据库，唯一项目名，退出清理自己的测试容器和数据；不会连接 server222。

追加 `--faults` 执行 F02：使用支持人脸的候选镜像，显式配置仅合成测试阈值，验证两个模型实际分析后，空缓存、同长度损坏缓存和错误身份的失败关闭、结果保留及逐项恢复。坏文件只写临时副本，原模型缓存始终只读；退出清除临时坏文件。不证明准确率或 G02 全部安全项通过。

追加 `--delivery --offline`（与 `--faults` 分开），并显式设置 `FOLIOPATH_TEST_APP_IMAGE` 为旧版、`FOLIOPATH_TEST_UPGRADE_IMAGE` 为新版本，执行配对升级/回退、安全检查及两类清除。离线测试叠加 `tests/release/compose.ai.offline.yaml` 将主应用网络也设为 internal，生产配置不变。已加载镜像及缓存就绪后执行，不下载依赖，不操作现有数据。

## 简化自动安装入口（CR-2026-053，已发布）

`compose.ai.simple.yaml` 已随1.6.1发布，与旧离线入口分别使用，不叠加。配对镜像 `evanqu/foliopath:1.6.1` / `:1.6.1-ai` 以及 `:latest` / `:latest-ai` 均通过AMD64与ARM64原生验证和推广。1.5.0不支持 `FOLIOPATH_AI_PROFILE`，不能把新配置用于旧镜像。

用户只需保存这一份 Compose，修改媒体目录、data目录及端口，然后执行 `docker compose -f compose.ai.simple.yaml up -d`。无必填 .env、手工身份JSON或生成程序。普通图库先启动；首次自动准备约3.3GB模型，建议为缓存和验证临时副本留出至少8GiB空间，AI运行内存先预留约6GiB。服务与模型页面显示检查、下载进度、准备完成和失败重试提示。

映射为单一 `/library:ro`、`/app/data`，以及AI专用的 `/cache`、`/ocr-cache`、`/verified-cache`。示例目录位于 `/root/foliopath`；现有照片位于别处时只修改 media 映射左侧，不能移动原件或用软链接代替媒体挂载。删除models会触发重新下载；删除data会丢失账号、设置、人工收藏/标签等。scratch可重建，不能与模型真相、媒体或data共用。

AI自动下载使用镜像内固定HTTPS源、版本、大小与SHA256；先验证再原子发布，崩溃遗留的同名临时文件下次重新写入，已完成文件复用。失败间隔60秒重试，退出取消下载。默认不继承HTTP_PROXY等；需要出站代理时，在AI服务显式设置 `FOLIOPATH_MODEL_DOWNLOAD_PROXY=http://代理主机:端口`，当前自动安装支持无认证HTTP代理，保留目标TLS，不自动回退直连。旧模型管理界面的下载代理设置属于主应用，不控制AI容器。

自动准备日志包含时间、直连/代理模式、代理主机、当前模型文件和来源域名。每个文件记录下载开始与校验完成；下载读取有进展时每15秒记录文件/总字节进度，缓存校验完成后汇总复用文件数并进入加载阶段。失败日志注明阶段、文件与HTTP状态码、连接/读取超时、DNS、TLS或校验原因，随后60秒重试；取消单独记录。日志不输出代理凭据、完整下载URL或重定向签名参数。使用 `docker compose logs -f machine-learning` 查看；此说明随日志改进镜像生效，已发布的1.6.1仍只输出简短失败提示。

profile 固定为 `standard-v1`，与 `FOLIOPATH_AI_DEPLOYMENT` 互斥。主镜像嵌入的预期身份来自canonical runtime；准备状态没有身份/信任作用。旧离线镜像、配置、缓存和历史向量不会被修改。新profile只下载多语言语义、公开版buffalo_l、JoyTag和OCR实际运行文件，不下载旧英文CLIP或Hugging Face本机元数据；buffalo_l识别权重与历史私有缓存不同，清单和face代次独立。迁移现有实例前正常停写全备份，不可重标旧结果。

模型说明：语义模型MIT；JoyTag/PaddleOCR按上游Apache-2.0说明；buffalo_l预训练权重受InsightFace研究用途限制，不代表获得商业许可。人物聚类使用版本化工程默认阈值，不声称通过正式标注准确率验收；每库仍须管理员显式启用，自动下载不会为未启用的库提交分析；将已有启用库切换到新身份时，按既有规则补齐/重新分析，迁移前应正常停写备份。

源码开发者用 `docker build -f services/inference/Dockerfile.managed -t foliopath-managed-ai:dev .` 构建AI；主应用按根Dockerfile构建，通过 `FOLIOPATH_VERSION` 或Compose覆盖选择本地配对标签。修改runtime、模型或策略后，在包含FastAPI/uvicorn/python-multipart的推理Python环境运行 `python scripts/generate_managed_ai.py`；普通 `--check` 只需标准库。生成器复用既有身份owner，不重写旧runtime源。生成JSON、catalog、Dockerfile.managed及lock必须一起提交。
