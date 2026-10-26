# 2026-10-26 发布候选统一验证

承接POST-MVP-5、既有FR-SEARCH/FR-OCR/FR-AUTOTAG和NFR-OPS/INT；owner：release engineering。用户授权固定版本和发行构建检查，然后并行全套回归、升级/恢复/任务续跑、原生AMD64/ARM64验证及交付文档。容量与24小时混合负载明确取消；质量数据与漏洞修复/复扫暂缓，均不记通过。此记录不授权正式发布、推送或改写历史发布门。

## 最终本地候选.3：约定的工程验证已收尾

当前交付版本为 `post-mvp5-local-rc-20261026.3`。主源码清单SHA `54673775cf02e1ee0ad1a0e172d52c3858a975d71e754d6525e9fe0aa0c82697`；独立源码包SHA `1b8f059afa60913f0cdfb44ac2c2fe78440e583e923b170c5d945602be018ab5`（48,075,365字节、2572文件，不含用户媒体或私有env）。AI canonical打包源清单SHA `2d27c7041954f6adbb2bc86d7b434298ecce3cd73568d99c06ea35667c90b272`。这是审阅过未提交变更形成的本地候选，不是干净Git release commit、正式CI promotion或公开稳定版本。**6010仍为主b3419001／AI7739486c，未部署本候选。**

.3继承6010已验收范围，额外纳入两项实际回归发现的CSS修复：长媒体库名称的窄屏撑宽，以及深色照片预览内“再次检查”按钮在悬停／触摸状态的对比度。复用既有token，不改API、迁移、模型、权重、输入输出身份、调度或任务语义。工作区后续人物审核、合并预检和撤销面板等仍不默认纳入。

| 部件 | 原生平台 | 本轮实际验证的不可变引用 |
| --- | --- | --- |
| 主应用 | ARM64 | 本地受检index `sha256:7de02f92ba1a6588a259675997a684a5768cf0b00c9270aafee557ecf4ba5d2f`；平台manifest `sha256:1d0a6c5b7fef66dda5f56da15ce6c380c5acb92067aef2e75d41e115b05a5fa4` |
| 主应用 | AMD64 | 平台manifest `sha256:c7d244fbcd41e526270e6f15c65e474f9d81e0d4565e80bbc0810a11b89ca7b9`；config／本地image `sha256:bf36584a84d65c40ad3c4309bf451b08904fabe39452935b0cf293f92e5aa116` |
| AI | ARM64 | 本地受检image `sha256:07ab113c0c1d0ec87b8db13322402c3158dc1cdd0c94279bfaef507d4b3cd6a2`；平台manifest `sha256:176b22c7ea89fb0283120797fa6e81e8c1337622be86e175156f46f61821d728` |
| AI | AMD64 | 平台manifest `sha256:a3fc002417d209ecc8ccf4d7044e229a245f88362bedb4c0b2a43395c400e1b3`；config／本地image `sha256:4f4b83894225937d76ee7e1028e3afce19b364c7f0b2966968bce3b86825209b` |

平台manifest、image config与本地含attestation的index是不同对象，不互相替代为公开registry引用。

| 约定事项 | 最终实际结果 | 证据边界 |
| --- | --- | --- |
| 固定版本／发行构建检查 | 同.3源清单，Go1.26.5／Node24.22.1；fmt、architecture、生成检查、合同及vet通过；正式Dockerfile双原生构建与字节核验完成 | 未提交源码，不冒称clean Git或正式CI来源 |
| 应用与前端回归 | .3 Go非integration全包、72文件454前端单测、类型及生产构建通过 | Linux同源19 integration／19 race、359 SQLite race及native图像包保留原版本与源连续性记录，未称全部在.3重跑 |
| 精确主镜像交付／恢复 | 同7de02嵌入SPA、image_smoke、b341→7de02升级、停写备份／新Linux卷恢复、旧镜像配对回退通过 | 仅独立合成库，未用6010做fixture |
| 最终嵌入UI浏览器 | 去重285项：231通过、54条件SKIP、0失败／0flaky；含原219＋新增64个悬停／触摸用例＋认证2项 | 三桌面引擎和mobile；普通用例多使用HTTP fixtures，认证包含真实后端链路；54未运行项不记通过 |
| AI真实原生协议 | 两架构同41份runtime／身份；各8项严格串行真实模型请求通过：640图像／中英文向量、非零人脸512向量、153类标签、OCR原文与空白图 | 功能输出不是准确率或容量；AMD另有独立人物／标签／OCR内存场景 |
| OCR实际任务 | 两架构实际worker、同job续跑、非空人工保护、坏源分类、OCR-only／融合HTTP及清理后隐藏通过 | 固定合成图和独立数据；不重算6010 |
| 原生图像处理 | 两架构48个顶层＋29个子测试，实际二进制exit0、0失败／0跳过 | AMD supervisor将77条PASS误与48比较返回1；已用原日志和Docker die=0独立校正，原错误结果保留 |
| 维护恢复 | 原五服务ID／镜像／挂载保留并healthy；root独立复核；PVE原8 GiB／配置SHA不变，测试资源已清理 | 原件与生产DB未被测试读取；未把历史OOM／restart计作本次新错误 |
| 安装归档 | 主和AI双架构OCI索引已本地合并，46／117个blob均逐字节SHA核验，两平台SPDX与SLSA保留 | 未重建运行层，未push或promote正式tag；Docker-load包的独立SHA另列交付清单 |

浏览器固定验证源SHA `0717843f358f758b727e58fb4a4bf15baf5895f28b59d71767b4f2f603a9f940`。唯一额外测试输入修正按canonical `hasTouch`选择tap／click，没有删断言或跳过桌面；生产冻结.3源未变。额外独立全E2E文件tsc仍遇到既有Axe／Page重复playwright-core类型不兼容，原日志保留，未声称通过；生产类型检查与新增hasTouch类型probe通过，两者不能混写。54项条件SKIP为：每个桌面引擎17项（mobile-only 1、D04开关未开1、P3未开5、face未开1、storyboard未开1、layered fixture缺8），mobile另3项（desktop-only 2、D04未开1）。这些带真AI／特定fixture的浏览器分支未由普通矩阵覆盖；独立原生AI／worker证据只关闭其自身范围。

失败和流程偏差没有删除：.1窄屏标题问题、.2移动预览对比度问题、测试浏览器OOM、测试tap/click装配问题、首轮维护控制器参数错误、3 GiB中文请求503，以及第三轮77／48统计错误均保留。最终成功测试不倒填旧失败。AMD AI重复执行过一次cached export，保留偏差记录，因此**不声称严格“每架构仅构建一次”的正式发布门已通过**；本地41文件／最终digest实测不能替代该门。

私有最终机器证据：`v3/`、`browser-runtime/final-v3-browser-summary.json`、`native/native-final-evidence-summary.json`、`native/maintenance-attempt3/vips-corrected-verification.json`及两个`root-independent-*.json`。下文.1／.2是历史过程，不能当当前状态。历史readiness和No-Go原样保留；本轮关闭的是用户约定的本地工程验证及交付准备，**不等于正式发布批准**。

### 本地安装归档

外置目录 `rc-images-20261026/`中：

| 归档 | 大小 | SHA256 |
| --- | ---: | --- |
| `foliopath-main-rc3-multiarch.oci.tar` | 63,303,680字节 | `8f3eed387d19c7795479eca70112534f862042daf87570412e3ce7e89c591d16` |
| `foliopath-ai-rc-multiarch.oci.tar` | 869,969,920字节 | `efbd009734268470878736abc4b68f88bc7f739874b5197c9596b5a51e4e5725` |

主OCI index为`sha256:31f0c5b7708cf309ed1183f5762e7c32f79c71568b0b949231c085eda865ee43`，AI为`sha256:3589385d9d7c5b9826d815ded04a2c3557ac914748f93d1e9ab15b3e31900019`。它们保存双架构与构建证明，不等同于已上传镜像或模型包；权重不在新增镜像层。供Docker直接加载的四份架构归档也已下载并核对完整SHA、config、架构及本地tag；它们不是正式发布镜像。SPDX／SLSA以原始和配对OCI归档为准，不声称load包自带完整证明。

| 部件／架构 | Docker-load归档 | 大小 | SHA256 |
| --- | --- | ---: | --- |
| main／arm64 | `main-rc3-arm64.docker.tar` | 31,493,120字节 | `b8ba2e47d73dd730faccef8411edc94a8554e05bc5c917e9608188bf652a89ce` |
| main／amd64 | `main-rc3-amd64.docker.tar` | 31,518,720字节 | `9f85463dc9cffce391a34469b903b91adac99a4093eed0b2a3d487dc301c5e9d` |
| ai／arm64 | `ai-arm64.tar` | 416,432,128字节 | `8d6bb5467d4bcaf0075791ed817c1c466397dd2c26f4b3da2f53ea0658da4014` |
| ai／amd64 | `ai-rc-amd64.docker.tar` | 449,720,320字节 | `b90f88b9fd89d34d6bf8b197f9b02b74ac38c37a13a70c028480f21c37096df0` |

加载须选择目标架构的主和AI配对包，核对归档清单后使用`docker load --input <归档>`；再核对local tag、image config及模型身份。包内local tag只供本地使用，不能写成公开可拉取的registry地址。完整config、local tag和两个OCI索引见交付清单。

## 最初冻结基线与候选.1（历史）

候选基线为已验收6010主b3419001与AI7739486c。当前工作区另有人物功能等未部署改动，不能直接将整个工作区打包上线；生产代码来自该部署冻结输入，保留OCR分类修复、四卡布局与原有模型身份，API/迁移不变。移除归档中误带的一次性迁移诊断程序，仅构建cmd/foliopath。

原本地overlay镜像不满足两项正式构建包装检查。本轮使用仓库canonical Dockerfile真实重建，不改断言来放行overlay，固定Go1.26.5、Node24.22.1及既有native组件。完整源清单、SHA和独立源码包存私有`.artifacts/release-candidate-20261026/`；本地候选名post-mvp5-local-rc-20261026.1不是已发布tag。源码包含已审阅未提交变更，不能冒充已有Git release commit或干净来源证明。

## 首轮安排与数据边界（历史）

固定候选后，回归/原生验证/交付文档分别记录实际结果。所有测试用独立合成媒体和Linux数据卷；6010及原数据保持，原件只读。升级、恢复和镜像烟测复用现有scripts并将Desktop活跃SQLite默认改为独立Linux卷；备份在主进程exit0后完整复制并逐文件核验，回退配对旧镜像和旧备份。存储helper是测试脚本唯一owner，不改产品存储合同。

用户新提供192.168.2.222用于原生AMD64验证，覆盖本轮此前不连接该机的旧限制，仅限独立测试；只读预检为Linux x86_64/Docker amd64，现有服务不是空机。须保留其服务/数据，独有builder、容器和卷，1CPU/1GiB单任务；先核对真实cgroup限制，不将模拟amd64或历史OOM/restart计为本次证据。当前不运行完整AI高内存组合，不能因此宣称模型双架构已验收。

## 首轮进展（历史）

canonical ARM64预构建通过，移出诊断工具后fmt与全架构检查通过；454项完整前端单测及生产构建通过。最终带源清单绑定、SBOM/provenance的镜像正在生成。其余结果待实际执行后追加，未完成项不预填成功。

## 回归发现与候选换版

候选.1的真实认证/建库/浏览长链在390px发现：128字符媒体库名打开照片预览后document宽1913px。DOM诊断证明BrowsePage内容grid的隐式auto列与h1不可断长词撑宽，不是预览本体或真实媒体损坏。owner CSS仅增加minmax(0,1fr)内容列及标题min-width/overflow-wrap；不隐藏整页overflow，不更改API/Go/身份/任务。原.1结果保留，修复在预候选.2验证后重新冻结，不能把旧版本通过套作新版本全部通过。

同轮测试工具问题分开记录：旧路径/工程文案/搜索radio等断言与已验收产品不符，只在私有验证补丁对齐并保留原失败；普通219浏览器发现用例的1.5GiB测试容器出现oom_kill，主应用没有OOM，后续按spec分组独立进程释放资源，不把此称作产品内存泄漏。


## 候选.2交付快照（历史，已由.3取代）

本节记录.2当时交付准备状态，之前.1的通过／失败保持原记录，不套用为.2全部通过；当前终态以上方.3为准。候选版本为 `post-mvp5-local-rc-20261026.2`；源清单SHA `dbca62c77c2402f2005aec285d02e4d7ede1558d0ed0a39e126585c3cb78ddfb`，独立源码包SHA `a53975eff27252a38dbe14bc2e71c82ebf5a414f5943d618bd911ebd94e3b3d6`（48,075,038 B、2572文件，不含私有env或用户媒体）。它不是现有Git发布commit或稳定tag。

- 本地ARM64当前受检引用：`sha256:4827e615221010764d6c954f1bca0fbe15d1130091a352dbb2d7b17e7f449efa`。
- 同次构建OCI内的ARM64运行manifest：`sha256:f23cbc100af118cd8dd373743f723304b8240b9e4617afead33163fd85250987`；归档SHA `871fb6ea4fe552302514ba5b7bd7903ff6f6fbcf360ec5f513c97c6e3e740fda`。已逐blob大小／SHA核验并保留SPDX与SLSA attestations；附带构建材料不等于有干净Git release commit或审批签署。
- .2相对.1仅纳入已定位的BrowsePage窄屏长标题CSS修复及测试／打包维护；后端API／迁移／AI Python和身份不变。工作区额外人物审核／合并预检／撤销面板等不默认纳入。

| 本轮事项 | 已执行事实 | 当前边界 |
| --- | --- | --- |
| 固定源码／canonical构建 | 同一.2源清单、Go1.26.5／Node24.22.1、ARM64正式Dockerfile构建及OCI完整性通过 | 不是公开稳定版本 |
| .2前端／源码检查 | 454项前端及类型／生产构建通过；source-checks记录格式、架构、生成、合同和vet | 不能用.1浏览器通过代替.2所有页面 |
| .2同镜像基础交付 | image_smoke及b341→4827升级／配对回退通过，独立合成Linux卷 | 不等于AI所有任务恢复已经验收 |
| .2真实认证浏览长链 | 同一4827嵌入SPA／真实HTTP的2项Chromium通过，原件hash／路径、数据库检查保持，app restart0／OOMfalse | 其余浏览器spec／引擎按分组继续，未完不计通过 |
| 原生主应用AMD64 | 已提供独立原生环境；.1构建曾通过但已被替代 | .2同源结果待本轮实际验证，不用.1抵充 |
| AI打包 | 新canonical入口从固定上游原生重现7739的41文件，ARM64打包／OCR身份通过 | AI双原生真实推理、维护和服务恢复待收尾 |
| 已部署6010 | 仍b341主／7739 AI，OCR998成功／1坏源／0pending，五Compose／Linux卷保留 | 没有因测试候选而更新6010或重算原任务 |

私有机器证据在 `.artifacts/release-candidate-20261026/v2/`、`browser-runtime/auth-embedded-v2/`、`frontend/`及`native/`；只认各记录明确绑定的版本、镜像和source。旧.1的359项SQLite race分片、Story72和其他局部记录可说明不变模块覆盖，但不得改写为.2全产品／原生AI验收。

## 安装和升级交付说明

本轮交付目标为主应用＋独立AI两服务，不增加第三部署单元。.3及配套AI的原生AMD64／ARM64引用和本地OCI归档已在上方列明；未上传前不提供猜测的下载地址或把本地image ID标成公开发布。正式安装入口遵循[部署文档](../deployment.md)和[AI Compose](../ai-compose.md)。

1. 核对同一候选source／版本／架构、镜像和模型缓存清单。单一媒体根只读挂到 `/library`；主应用独占data，AI没有媒体／数据库挂载或公开端口。
2. Docker Desktop使用已准备的Linux数据卷，保持base＋unlimited-memory＋multilingual＋ocr＋data-volume五份Compose及私有env；原生Linux本地bind继续遵守停写备份。不要拿旧宿主data覆盖在线Linux卷。
3. 升级前停止主应用并确认正常exit0，完整备份当前data和全部私有配置／镜像／身份记录；逐文件核验并在新空Linux卷恢复检查后再切换。不能在线复制SQLite family或从macOS打开活跃库。
4. 用相同项目／媒体根／数据卷启动配对镜像。单独主应用更新不重启AI；需要更新AI时，在停写备份后先确认AI真实模型ready再启动主应用，不能仅凭ping健康认定兼容。
5. 登录检查媒体库、人工标签、删除抑制、人物、原任务和已完成结果。任务恢复使用既有同job／missing语义，取消／失败历史不复活，不清claim／job表、不把旧身份结果重新贴为新身份。
6. 回退停止新主并保留失败后数据，将升级前完整备份恢复到另一新空卷，使用配对旧镜像／身份；不让旧镜像直接打开新schema，不覆盖唯一副本。保留备份，清理须另有保留期决定。

本轮升级／恢复脚本已增加独立Linux卷适配和停写清单核对，未降低空间保留或原件只读限制。执行测试与更新真实实例是两件事；本记录不授权自动部署、提交、推送、创建稳定tag或公开发布。

## 本轮已知问题与未执行项

- 容量与24小时混合负载按用户要求取消，不执行、不宣称通过，不承诺本候选的10万媒体／1万目录或4GiB联合容量与24小时长稳。
- 质量数据按用户要求暂缓，不将无文字结果、标签输出数量或评分工具单测当作搜索／标签／人物／OCR准确率验收。
- 漏洞修复／复扫暂缓；已有b341和7739报告保留，不套用为.3新扫描或漏洞已清零。此前主19 High、AI17 Critical／141 High是版本扫描发现，尚无逐项可达性豁免。
- 模型授权材料不作为本地开发等待项；没有代签或编造模型再分发批准，不以此更换模型／身份。
- 6010的pixiv剩余1张明确JPEG源损坏，不是当前待排队；原件未修复或改写，恢复可靠原件后才能另行显式补齐。其他库原10张坏／空源继续跳过。
- 本轮本地原生／浏览器／任务恢复证据已收齐，按上方实际范围验收；54个浏览器条件SKIP、AI重复export偏差、未提交Git来源与正式CI promotion缺口保留。历史release gate和No-Go不被本轮本地功能结果改写。
