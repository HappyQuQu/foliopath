# FolioPath 安全模型

ADR-0028 的显式 PNG v4 仅在部署启动选择，不接收浏览器逐图策略。PNG 上限 64 MiB / 40 MP、
1 MiB 有界头、公共边长和 8 MiB 输出验证，在安全已打开的 Linux FD 内解码；没有媒体 pathname 重开、
原件 mmap/写入或任意下载。复用共享 native 准入/回收和取消/空读预算，截断 PNG 不沿用 JPEG 容错。
缺省 v3 及标签输入不变，新身份不覆盖旧向量。合成测量不保证恶意输入恒定峰值或默认预算通过；
详见[修复与风险边界](changes/FIX-2026-10-23-semantic-large-png.md)。

2026-10-23 Node 24 工具链维护重新审计现有 npm 依赖，发现 high/critical 告警（含 Storybook/Vite 等），
`audit:high` / 综合 `npm run check` 未通过；未添加例外、降低阈值或改变原依赖图。
Node 升级与兼容性测试通过不代表漏洞已修复；[快照与边界](changes/FIX-2026-10-23-node24-lts.md#安全审计待处理不是-node-升级引入的新依赖)。

ADR-0027 的交接不是公共恢复后门：仅信号标记且处理器/heartbeat 已退出的精确 running claim 可有审计交回。
旧版本离线工具需部署者先确认 owner 停止/完整备份，使用固定 Linux 数据根和最多 32 项精确清单；
闭集 kind、不接受 SQL/任意路径、不复活取消或历史终态，失败消息不输出底层异常。未增加网络/API/权限。

ADR-0026 的 photo-tag overlay 使用相同固定 JoyTag 权重与已审核类别 key；展示翻译不是自由文本提示词。
类别变化绑定 policy 摘要，运行源变化绑定独立 tag runtime；不重标旧结果，不改变 legacy 搜索/人物身份。
复用唯一 scheduler/killable child、原 snapshot/transform owner；没有额外服务、出网、公开端口或媒体/DB 挂载。
中文检索只展开有界固定别名，不接受 SQL 或模型 URL；人工原文不被推断式改写，用户移除与人工优先照常生效。

ADR-0025 多语言仅从固定 immich-app revision 获取，部署逐文件核对 Git/LFS 与独立 SHA256 清单。
新语义身份与 legacy face/tag 分开验证，仍是唯一 scheduler；未知模型/维度拒绝。
新增 scratch volume 是可丢弃工作区，不挂原媒体/数据库，不信任遗留内容/symlink；
不以原宿主权重 mmap 替代已核验副本。旧 2 GiB 合同不放宽，新语义单独 4 GiB/512 文件预算。
固定镜像/私有网络仍是前提，不声称远程 attestation；权重许可/发行材料没有因本地通过获准。

2026-10-23 / ADR-0023：在线数据库维护仅在主应用容器内执行 `foliopath database-check`，共享 Linux 锁域；
禁止宿主机直接打开活跃 Docker Desktop WAL 文件。命令不接收路径、Linux-only、只读、15 秒上限、固定输出，
不暴露错误原文或数据；不组合服务器/worker/模型，不迁移或写库。主应用 shell 权限没有扩大，AI 仍无 DB/媒体挂载。
运行错误只记录闭集分类、来源/阶段与存储数值码，不能借诊断输出 SQL、路径、凭据或异常原文。

2026-10-23 人物失败诊断：复用鉴权 faces 读取合同，仅返回固定原因码、opaque 资产/任务 ID、文件名与时间；
不返回底层异常、SQL、主机路径、源指纹、模型向量或人脸像素。样本最多 20，原因种类最多 5；原始媒体保持只读。
本地 Docker 故障注入只改脚本创建的临时合成库状态，不能对用户媒体库或线上数据库进行类似注入。

## 自动标签与有界推理增量（2026-10-22）

[CR-2026-031](changes/CR-2026-031-automatic-photo-tags.md) / [ADR-0021](adr/0021-bounded-inference-and-automatic-tags.md)
保持主应用 + AI 两个部署单元；AI 仍无公开端口、媒体/数据库卷、Docker socket 或联网下载路径。
新自动标签端点沿用登录、写操作 CSRF、设置 If-Match、任务幂等键与现有取消合同，浏览器不提供模型
URL、路径、阈值或候选词。部署管理员固定 `foliopath-tag-v1` 模型/输出策略/runtime/cache 身份，
主应用在请求前及响应头复核，不能信任首次 HTTP 返回作为预期身份。

AI 父进程只允许一个实际推理子进程和一个等待请求；26 秒总期限覆盖排队、所用模型子集的逐字节
验证/私有复制、加载与推理。成功、失败、取消、超时均须终止并 wait 回收 child 后释放准入并清理
其专属 tmpfs；不能靠 Python GC 推定 ORT 内存释放。启动验证完整固定 manifest，随后每次仅暂存
当前能力族的完整所需子集，不放宽文件/hash/目录边界或整体身份。保留既有 2304 MiB RAM、
1200 MiB 快照预算；具体输入上限与固定输出过滤以 [AI 服务合同](../services/inference/README.md) 为准。
固定类别仅为客观物体/场景/动作等安全过滤，最多 20 项，不输出敏感身份类别；分数不代表校准概率。

原图只经主应用 `internal/files`/media 边界只读访问，推理前后验证实际打开的媒体身份；AI 仅收到
有界编码内容。数据库源指纹单独不能证明文件未变。写入在同一事务检查开关、取消、租约、claim、
模型及源，再合并标签并推进任务，失败关闭不留半次提交。人工标签与按图同名删除抑制属于 `/app/data`
应用状态：重跑、源变化、模型变化不会消除人工选择；全局标签删除不成为未来图片的黑名单。
失败/取消持久阻止同模型自动重入，避免服务异常循环创建新任务；关机走租约恢复，不伪装用户取消。

运行时代码/策略身份变化须重新核验，不静默沿用旧搜索/人物索引身份。以上为实现边界，最终本地
Docker、模型来源/质量、供应链和发行签署必须分别有证据；不能由单测或此安全说明替代。

P5 根 `compose.ai.yaml` 落实既有双服务边界：AI 仅 internal 网络、无端口或媒体/数据库卷，只读缓存；主应用单只读媒体根和专用状态，默认回环端口。两侧只读根、cap_drop ALL、no-new-privileges、禁 swap，缺失 bind 源不自动创建。主应用保留对外网络用于正常 Web 接入，不声称整套部署完全禁出网；AI 不继承代理，不联网下载。配置/实际挂载检查不等于 G02 全部安全项通过。

P4 大 JPEG 语义预处理修复：仅 JPEG 使用经源尺寸校验的 decoder shrink-on-load，首次 native load 前选倍率并在加载后验证 ≤16 MP；不把 180 MP JPEG 源尺寸预算当完整解码预算。JPEG 源字节预算 64 MiB，其他格式 32 MiB；输出 8 MiB、长边 1024 不变，其他格式仍拒绝 >16 MP。原图只读、失败关闭、既有有界任务并发不变。该限制不保证所有恶意压缩输入的恒定内存，仍须容器资源限制及回归。[变更记录](changes/FIX-2026-09-17-semantic-large-jpeg.md)。

P2 人脸组操作门禁由 face owner 持有，未质量校准时正式 app 的 `assign_cluster` 与带 sourceCluster 的创建人物均拒绝，不能只禁用 UI。逐脸人工归类仍须认证、CSRF、revision 和幂等保护。非空验收使用生成的虚构人物 fixture，不读取开发者真实媒体；AI 仅接收主应用编码内容，无媒体/数据库挂载。测试 profile 和成功下载不代表生产质量或模型授权已通过。

2026-09-15 候选启动组合：部署环境独占 AI origin 和预期身份配置，浏览器不能提供
目标地址。每次发送图片/文本前先核验身份，响应仍验证身份头。共享两请求预算已满
不修改模型可用性。主应用本地测试端口限回环；AI 服务不发布端口。此为可信部署兼容
核验，不提供远程证明，模型来源/许可仍按本地未审查缓存标记。

ADR-0020 专用 AI 候选镜像先按固定清单、目录 fd/O_NOFOLLOW 与 SHA-256 将模型复制到
容器私有快照，推理不继续访问宿主原缓存。主应用不挂模型或 Docker socket。仅可信私网
使用固定任务；身份响应属于部署兼容性证据，不是远程 attestation。候选清单来源是本地
复制缓存，不能冒充上游签名。主应用集成与发行验证未完成；见[服务合同](architecture/independent-ml-service-contract.md)。

ADR-0019 增加同机容器间图片/文本 HTTP 边界：默认私有 Docker 网络，不发布 AI 端口，
不挂载媒体或主数据库；主应用受控发送载荷。模型身份、载荷/响应限额、超时及出站下载策略
必须在生产接入前定义。当前 spike 只有合成输入、只读模型缓存与禁外网网络，不证明生产安全。

POST-MVP-5 显式模型出站代理见 [ADR-0017](adr/0017-explicit-model-download-proxy.md)。
配置 `FOLIOPATH_MODEL_DOWNLOAD_PROXY` 后，目标 DNS 与目标地址策略交给部署信任的代理，不能声称
客户端仍在逐跳锚定目标公网 IP。客户端仅拨显式代理，保留目标私网 IP 字面量/本地域名拒绝、
HTTPS 验证、固定源/重定向清单、文件 hash 和限额；不继承环境代理、不回退直连。
ADR-0018 允许已认证管理员经 CSRF 保护配置可信代理 endpoint；custom 地址可回显给管理员，
部署地址只显示是否配置。API no-store，错误不含地址；固定目标测试限时 20 秒、每实例一个，
不能指定任意探测目标。配置代理意味着信任该 endpoint 的 DNS/IP 策略。

## 范围和信任边界

FolioPath 读取用户提供的目录和媒体文件，并通过 Web 服务展示内容。即使媒体目录只读，路径、文件名、元数据和媒体编码仍是不可信输入。主要边界包括：

- 浏览器与 HTTP API 之间的网络边界。
- `/library` 只读媒体根目录与容器其他文件系统之间的路径边界。
- Go 进程与 libvips、`ffprobe`、FFmpeg 等原生媒体解析器之间的进程/库边界。
- 可重建媒体索引与未来用户配置、认证和分享数据之间的数据边界。

首版安全目标是防止越界读取、未授权网络暴露、媒体解析导致的无界资源消耗，以及扫描故障造成的索引破坏。只读挂载只能防止修改原媒体，不能替代访问控制和输入验证。

## 模型下载出站边界（CR-2026-026）

[ADR-0016](adr/0016-admin-model-download-catalog.md) 允许部署管理员在固定数据文件中配置下载源。
这是可信部署配置，不接受浏览器提交 URL、任意路径或远端动态目录；更改文件等价于更改模型信任清单。
应限制该文件的写权限并纳入备份审核。未配置不出站；无效配置拒绝启动。下载仍需认证、CSRF 与幂等键。

`internal/aimodel/downloadhttp` 只执行 HTTPS GET（443），拒绝凭据、私网/回环/链路本地/保留地址；
解析结果全量验证后直接拨号固定 IP，避免重解析越过校验。重定向最多五跳，只允许原始主机或明确配置的
CDN 主机；每一跳继续验证 HTTPS/IP，不继承环境代理、不上传媒体或查询。单文件最多 30 分钟、
文件大小由 manifest 限制，现有 worker 串行处理；managed store 保留配额/余量与逐文件 SHA-256 校验。
失败不会注册模型，错误只返回脱敏代码 `download_failed`。取消与重启清理遵循现有安装生命周期。
此规则仅替代历史“无下载出站”限制，不提供公共源运营、模型质量/许可批准或推理发布授权。

## 文件系统边界

- `/library` 是唯一允许创建媒体库、也是唯一允许的媒体挂载目标。它自身可以是
  一个只读 mount，但其后代不得嵌套 volume、bind mount 或其他 mount point。
  多个 UI 媒体库必须对应这个单一根中的普通目录。
- API 不接受任意宿主机路径或容器绝对路径；公开读取接口使用媒体库、目录和媒体 ID。
- 用户选择的路径先转换为相对于 `/library` 的规范形式。拒绝绝对路径、NUL、`.`/`..` 越界、编码绕过和平台不支持的路径表示。
- Linux 上不能把“先检查、后打开”作为安全边界。`internal/files` 必须从已锚定的
  `/library` 目录文件描述符出发，以 `openat2` 的
  `RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_XDEV` 原子解析并打开实际
  文件或目录；只做字符串前缀、realpath 或 device/inode 比较均不足够。
- Linux 内核或 seccomp/LSM 不支持所需 `openat2` 行为时必须失败关闭，媒体根不得
  进入可用状态，且不得回退到 `os.Root` 或较弱的用户态检查。非 Linux adapter
  仅提供开发证据，不声明同等级的 mount-boundary 保证。
- 扫描默认不跟随目录符号链接。任何允许符号链接的新设计必须先处理逃逸、循环和检查/使用竞态，并记录 ADR。
- 管理 UI 只显示 allowed-root-relative 库根和安全的 `/library/...` 容器标签；浏览 UI 显示媒体库内
  相对路径。错误响应和日志不得泄露宿主机路径、数据库位置或容器内部敏感路径。
- 目录选择器必须先取得权威媒体库快照，再把相同、祖先和后代目录标为不可选择；无法读取
  或无法规范化该快照时失败关闭，不能在不知道重叠状态时继续枚举成可选目录。

若媒体位于多个独立宿主机卷，只能由部署者先提供一个对容器表现为单一文件系统、
没有后代 mount crossing 的呈现根，再一次性只读挂入 `/library`。FolioPath 不
承诺具体 union 或聚合技术；`NO_XDEV` 拒绝仍是应用的最终边界。完整决策见
[ADR-0009](adr/0009-linux-openat2-single-media-root.md)。

纯相对路径词法规则集中在 `internal/pathpolicy`，实际文件系统身份、解析和打开集中
在 `internal/files`；其他包不得自行拼接或打开真实媒体路径。

`S2-002` 已把已认证目录选择器接到该生产边界：handler 只传 allowed-root-relative
`parent`，`internal/library` 拥有自然排序、页面上限和 query-bound opaque cursor，
`internal/files` 从锚定根按批次读取直接子目录并把 symlink、unreadable 与 mount boundary
映射为有限原因码。游标使用随机进程密钥的 AES-GCM，不包含明文父路径或最后目录名；错误
响应不包含绝对根、errno 或底层路径。

### 自动发现事件边界

Linux watcher 的名称、cookie、顺序和事件类型全部是不可信 hint。adapter 只能产出媒体库
ID、规范相对目录/名称和稳定事件类别；它不能返回给 API 的绝对路径，也不能直接修改索引。
每次定向校准与新 watch 注册仍必须经 `internal/pathpolicy` 和 `internal/files` 的
`openat2(RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_XDEV)` 锚定边界重新确认。

`delete`、`move-out`、`unmount`、overflow 或 watch invalidation 本身没有删除资格。只有库根
身份未变且父目录完整、安全枚举成功后，scanner 才能删除明确缺失的直接子项；否则保留旧
索引并进入 degraded/offline。持久任务、日志和 API 错误不得包含宿主路径、容器绝对路径、
原始 errno 或内核消息。`GET /api/v1/catalog/state` 需要现有管理员 session，只泄露一个
单调通知 revision，并使用 `no-store`。

`S2-105` 把相同失败关闭边界接入生产扫描：媒体库根离线、根 symlink、root identity
变化、后代 mount crossing、部分目录权限失败和一般 I/O 只映射为 OpenAPI
`ScanFailureCode` 中的稳定码。除完整成功扫描外均不得运行 stale cleanup；公开状态和
SQLite 不保存 errno、绝对路径或任意内部错误文本。

## 网络和认证

开发预览版在内建认证完成前默认绑定 `127.0.0.1`，或通过可信认证反向代理访问；不得将其直接暴露到局域网或互联网。内建认证完成后，受信局域网可通过 HTTP 直连，但不提供匿名模式；公网或不可信网络的 TLS 与访问控制由部署者负责。详见 [ADR-0005](adr/0005-built-in-single-admin-auth.md)与 [ADR-0010](adr/0010-authenticated-lan-http.md)。

稳定版内建认证至少需要：

- 首次管理员初始化流程，且不能保留公开默认密码；初始化完成后必须原子关闭再次创建管理员的入口。
- 使用适合密码存储的算法、会话过期和安全 Cookie。
- 对所有状态修改请求实施 CSRF 防护或等价的同源令牌设计。
- 登录、扫描启动、目录浏览和媒体读取的速率/并发限制；退出、改密或安全事件能够撤销会话。
- 分享链接使用高熵、可撤销、可过期的令牌，并明确媒体库与目录范围。

MVP 没有分享链接；上条只约束未来功能。除健康检查与受限的初始化/登录端点外，媒体、搜索、扫描、媒体库和设置 API 均要求有效会话。

`S1-101` 已冻结认证 HTTP 与持久化边界：数据库唯一 `singleton_key` 阻止第二个管理员；
密码只保存带 scheme/参数的 verifier；随机会话 Cookie 与 CSRF 值只保存 32 字节摘要；
session 具有绝对过期、撤销和 `auth_version`。认证 JSON 不允许缓存，setup/login 校验同源
`Origin`，状态修改同时校验 Cookie 与 CSRF。

`S1-102` 已实现密码存储与首次初始化领域边界：使用 Argon2id v19、64 MiB、3 次迭代、
4 lanes、16 字节密码学随机 salt 和 32 字节派生 key；验证器只接受这一组已知参数和严格
长度，不接受调用方提供成本参数。该选择采用
[RFC 9106 的内存受限推荐配置](https://www.rfc-editor.org/rfc/rfc9106.html#section-7.4)和
[Go `x/crypto/argon2` 的 Argon2id 指引](https://pkg.go.dev/golang.org/x/crypto/argon2)。
用户名使用 NFKC 和 Unicode full case folding 形成唯一比较键；创建前进程内串行，最终由
SQLite 写事务与 singleton 约束原子关闭再次初始化。日志和错误不包含密码或 verifier。
首次管理员密码接受 8～128 个 Unicode 字符并拒绝控制字符；不强制大小写、数字或特殊
字符组合。服务端 `internal/auth` 是该规则的最终 owner，前端长度提示只用于即时反馈。
该可用性调整记录于
[FIX-2026-07-29](changes/FIX-2026-07-29-admin-password-minimum.md)，不改变 Argon2id、
登录限流、错误脱敏或会话边界。

`S1-103` 已实现服务端会话：

- 每次初始化或登录使用 Go `crypto/rand` 发放两个独立的 32-byte 随机秘密；复合
  `foliopath_session` Cookie 为 host-only、`Path=/`、HttpOnly、SameSite=Strict，并在经
  验证的 HTTPS 传输上设置 Secure。浏览器脚本只收到 CSRF 部分，不能从它还原 Cookie。
- SQLite 只保存整个 Cookie 和 CSRF 部分各自的 SHA-256 摘要；不存在可直接重放的明文列。
- 固定 7 天服务端绝对期限，不因活动滑动续期；重新认证发放全新 Cookie/CSRF。退出立即
  写入撤销时间，`auth_version` 变化或管理员禁用也使会话失败关闭。
- 过期和撤销记录保留 24 小时用于稳定识别过期状态，随后在创建新会话时有界清理。
- 未知账号、非法用户名和错误密码统一失败；未知账号仍执行虚拟 Argon2id 校验，降低账号
  枚举的时间差。

该设计满足 [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
关于 CSPRNG、高熵标识、服务端绝对过期和退出撤销的要求；7 天短于
[NIST SP 800-63B AAL1](https://pages.nist.gov/800-63-4/sp800-63b.html#reauthentication-1)
建议的 30 天总体上限。随机源使用
[Go `crypto/rand`](https://pkg.go.dev/crypto/rand)。

`S1-104` 已实现认证 HTTP handler 和集中 middleware：

- 仅 health 与精确匹配的 auth status/setup/login 操作匿名；其余 `/api/v1` 请求必须先
  通过唯一 Cookie 的服务端 session 验证，未实现的业务路径也不会在认证前泄露存在性。
- 所有状态修改在执行业务 handler 前常量时间比较 session-bound `X-CSRF-Token`；
  setup/login 尚无 session，因此在解析 JSON 凭据前对实际请求 scheme/host/port 执行
  完整 Origin 同源比较。缺失、`null`、多值、userinfo、path 和不等端口均失败关闭。
- 所有认证 JSON 与公共错误使用 `Cache-Control: no-store`；JSON 只接受
  `application/json`、单个值、已知字段和最多 4 KiB。
- setup/login 每个直连 peer 每分钟 10 次，status/session 120 次，logout 60 次；内存
  bucket 上限 4096，满载失败关闭。限流和同源判断只使用直连地址与真实 TLS，不信任客户端
  转发头。

这符合 [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
关于有状态同步 token、自定义 header、同源 Origin 完整比较和缺失时阻断的建议。`S1-105`
已验证五类认证操作的错误脱敏、CSRF 短路、未知账号/错误密码一致响应、绝对过期边界、
并发 setup 单胜者和并发 session/SQLite 更新；`S1-106` 已通过
[认证 Backend Evidence Ready](gates/MVP-2026-07-23/s1-auth-backend-ready.md)，允许正式
前端连接真实认证 API，但不放宽回环监听、可信代理或发布限制。

`S5-003` 已实现反向代理信任边界。默认直连模式清除并忽略所有客户端转发声明，使用实际
peer、Host 与 TLS 状态。非回环监听可直接提供经认证的 LAN HTTP；配置
`FOLIOPATH_TRUSTED_PROXIES` 的显式 IP CIDR 后进入代理专用模式，只有匹配的直连 peer 才能提交
单跳、单值的 `X-Forwarded-Proto: https`、`X-Forwarded-Host` 和数值
`X-Forwarded-For`。缺失、多值、逗号链、非 HTTPS、非法 authority/IP 或同时提交标准
`Forwarded` 均失败关闭。直接 HTTP 或验证后的 HTTPS transport 都是 Origin 与认证限流
身份的唯一来源；Secure Cookie 和 HSTS 仅由 HTTPS transport 启用。
客户端 key 的唯一输入。

所有 HTTP 响应统一设置限制型 CSP、禁止 frame、`nosniff`、`no-referrer` 和禁用相机/
位置/麦克风的 Permissions Policy；只有验证后的 HTTPS transport 返回一年 HSTS。
部署者仍必须用宿主机端口绑定/firewall 阻止代理旁路，CIDR 本身不替代网络隔离。

Stage 5 候选使用固定 digest 的 Go 1.26.5 trixie build layer 与 Debian 13
distroless runtime；生产 final stage 不含 shell、curl、tar 或包管理器。
固定来源和升级安全修复不能替代镜像审查：
[S5-007 候选供应链 Gate](gates/MVP-2026-07-23/s5-supply-chain-candidate.md)记录的
本机 arm64 修复候选已为 `0 Critical / 0 High`，但发布前仍必须从干净提交在最终原生
双架构 digest 上复扫并完成 provenance 与安全/合规签署。

## 媒体解析

- 按允许的媒体类型和文件魔数识别输入，不能只信任扩展名或客户端 MIME。
- MVP 允许列表为 JPEG、PNG、WebP、GIF，以及 MP4、MOV、MKV；SVG、HEIC/HEIF、AVIF 与 RAW 不进入首版媒体处理契约。容器允许不代表浏览器编码可直放。
- Post-MVP/1 通过 CR-2026-010 将 AVI 加入同一扩展名/MIME allowlist、FFmpeg 资源预算
  和原内容 Range 边界；AVI 容器不构成浏览器可播放承诺。
- libvips、`ffprobe` 和 FFmpeg 操作必须具有超时、取消、输入大小限制和全局并发上限。
- 限制图片像素数、动画帧数、媒体探测时间和输出尺寸，防止像素炸弹及 CPU、内存或磁盘耗尽。
- 调用媒体 CLI 时使用独立参数数组，不拼接 shell 命令；媒体文件名不得成为可解释的命令片段。
- SVG、HTML 或其他主动内容不得作为可信同源页面直接内联。缩略图响应使用明确 MIME、`X-Content-Type-Options: nosniff` 和适当的内容安全策略。
- 文件名、路径、EXIF 和视频标签在进入 HTML、日志或错误消息前必须按上下文编码。

媒体处理任务写入受控缓存目录，使用随机临时文件和原子替换。缓存键不能直接包含未经处理的路径片段。

`S3-007` 的缩略图读取仍经过集中 session middleware；API 只接受 opaque asset ID，
handler 不接收缓存路径、不访问 SQLite/文件系统，也不调用媒体工具。delivery capability
只打开数据库记录的 `/app/data/cache` 相对路径，返回 WebP、ETag、private immutable cache、
`nosniff` 与限制型 CSP。缓存缺失或长度异常只撤销派生 ready 并重排 durable job；
公开错误不包含 cache path，离线源返回稳定 `source_offline`。

`S3-006` 初始解析预算为图片编码最大 256 MiB、视频最大 4 GiB；2026-08-06 的
[大视频探测与失败诊断纠偏](changes/FIX-2026-08-06-large-video-probe-diagnostics.md)保留图片
上限并把视频源上限提高到 1 TiB，使 4 GiB 以上视频仍进入 FFprobe。超过上限使用独立的
`source_too_large` 诊断，不归为媒体损坏。2026-08-12 的
[媒体处理韧性与诊断纠偏](changes/FIX-2026-08-12-media-processing-resilience.md)进一步要求
先核验真实图片格式：非 JPEG 仍最多 100 MP；只有处理器识别为真实 JPEG 时才允许最多
180 MP，并在超过 100 MP 后使用 shrink-on-load，不超过 100 MP 的真实 JPEG 保留原变换路径。
单边最大 32,768 px、工具 stdout 最大 8 MiB、stderr 诊断最多保留 64 KiB。工具退出 0 且
有界产物验证有效时，stderr 超过保留上限不得反向改写成功；非零退出和无效/超限产物仍失败。
分类只使用已保留的脱敏稳定模式，不持久化原始工具输出。0 字节和像素超限分别记录为既有
`source_read / invalid_media_data / filesystem` 与 `probe / source_too_large / libvips`，
不与截断解码或未知工具故障混淆。

[JPEG 有界容错与 MPEG-TS 派生兼容](changes/FIX-2026-08-12-tolerant-jpeg-mpegts-derivation.md)
不放宽输入或输出资源上限。JPEG 容错前必须已严格核验真实格式、尺寸不超过
100 MP，且严格错误命中窄 allowlist；容错只执行一次，产物仍须通过 WebP、字节、
尺寸、容量与原子发布校验。成功仅记录有界的稳定 attempt warning，不保存 libvips
原始输出；当前也不通过 API/UI 暴露 warning/degraded 状态。伪装格式、0 字节、超限和其他
损坏不进入容错。

MPEG-TS 兼容仅向已由 scanner 按现有视频扩展名索引的候选提供有界服务端派生。
它继续使用 `internal/files` 锚定的只读 FD，不新增任意路径、上传、网络输入、remux、
转码或原文件写入。`.ts/.mts/.m2ts` 不进入支持合同，服务端成功也不推断浏览器
可播放。
应用显式启动 govips，固定 native concurrency 1、64 MiB/32 entry cache 和 0 cached files；
一个媒体 worker 是进程级唯一任务并发，避免两个约 100 MP JPEG 在 1 GiB 限额内同时求值。
FFmpeg/ffprobe 使用单 decoder/filter thread、
probe 与 poster 各自 60 秒超时和独立进程组，取消会杀整个组而不是只杀直接子进程。libvips 仍是进程内 native
调用，不能承诺在任意 C 调用中间抢占或隔离 native crash；当前在求值前拒绝超限输入，并在
返回后的第一个安全点重新检查取消。改变到进程隔离需要先接受 ADR。

真实 Linux 8 MiB tmpfs `ENOSPC` fixture 已确认缓存发布失败不留下临时文件、不提交 ready，
释放空间后可恢复；长期 WAL/temp 同卷竞争和最终卷容量仍由 Release Gate 验证。

### Post-MVP 视频故事板处理

[FTR-VID-001](features/video-storyboard-preview.md)不新增信任边界，但把一次 poster 抽取
扩展为最多 10 个目标时间点的派生处理。`VSP-S1/S2` 已固定并验证总 job 超时、
seek/输出上限、峰值 RSS、worker 优先级和 backfill admission。它继续使用
`internal/files` 的锚定只读 FD、参数数组、单 decoder/filter thread、进程组取消、原子缓存
发布、认证 delivery 和错误脱敏。不得用完整顺序解码长视频、API 线程现场生成、任意路径或
更高无界并发实现故事板。

`VSP-S2 Backend Evidence Ready` 已 Go；故事板继续使用独立的自适应预算，
完整 feature 发布继续由 `VSP-S4` 阻断。

2026-08-12 的修复为无帧时间点增加最多 `±1s` 的有界邻帧恢复，随后才沿用 10→4 帧降级；
四帧无帧耗尽记录永久 `media_processing_failed`，预算耗尽记录永久
`media_processing_timeout`，均不进入自动重试循环。1080p/10 帧仍为 45 秒，4K
四帧 fallback 为 120 秒，主计划与 fallback 合计不超过 5 分钟。生产 FFmpeg 使用 external
libdav1d 处理 AV1 派生资源，但该服务端能力不替浏览器决定原视频能否播放。

## 容器和持久数据

- 只把一个媒体 volume 挂到 `/library:ro`，不在其下创建子挂载。根据
  [ADR-0012](adr/0012-root-runtime-bind-data.md)，应用默认以 root 运行以兼容自动创建的
  root-owned 数据 bind；这是已接受的扩大风险，不放宽媒体路径或写入边界。
- 最终镜像只包含运行所需的 Go 服务、libvips、FFmpeg、证书和必要运行库。
- 默认丢弃不需要的 Linux capabilities，并启用 `no-new-privileges`；在兼容的部署环境中使用只读根文件系统和独立可写 `/app/data`。
- 默认 seccomp/LSM 策略必须允许应用使用所需 `openat2` flags。生产容器不需要
  mount capability；`CAP_SYS_ADMIN` 只授予隔离测试容器以构造边界探针。
- `/app/data` 可由 Docker 自动创建为 root-owned bind 目录，并避免放在不可靠的网络文件系统上。
- 数据库迁移、磁盘已满、断电和进程终止必须有恢复测试。缩略图缓存耗尽不能破坏数据库或原媒体。
- 发布镜像使用不可变版本标签，并为 amd64/arm64 维护依赖与许可证清单。
- [ADR-0026](adr/0026-native-release-image-promotion.md)：原生 runner 在同一完整
  SHA/VERSION/image plan 下验证不可变 digest，成功记录配对后才允许更新发布标签。
  合并保留每架构 SPDX SBOM 和 SLSA provenance。缓存是可重建 CI 状态，不作为验收
  证据；仅恢复三个编译/依赖缓存目录，不接入真实媒体或应用数据库。

## 日志和错误

- API 使用稳定、可公开的错误码；客户端不接收 SQL、堆栈或媒体工具原始 stderr。
- 日志默认记录媒体库 ID、任务 ID 和相对路径的安全摘要，不记录认证秘密、分享令牌或宿主机绝对路径。
- 媒体库创建/移除的 `Idempotency-Key` 只以 SHA-256 摘要持久化，不记录明文或原始请求体。
- 媒体库 removal capability 只清理 SQLite、durable jobs 和 `/app/data` 派生缓存；其端口
  不提供打开、写入、移动、改名或删除 `/library` 原件的能力。
- 对扫描失败、路径拒绝、媒体解析超时、队列深度、缓存占用和磁盘余量提供可观察信息。
- 同一损坏媒体的自动重试必须有上限和退避，避免永久占满工作队列。
- 认证的处理结果只读取既有结构化失败事实；媒体 relative path 在 HTML 中按文本编码，
  cursor/limit 有界；invalid/unsupported 保持永久失败，手动恢复只重新接纳 transient code。
- 派生尝试只持久化白名单 stage/reason/tool、退出码和耗时，每个 job 最多 10 条；FFmpeg/ffprobe
  stderr 只在进程内匹配稳定原因后丢弃，不保存原文、命令参数或输入路径。未知输出统一归类为
  `tool_failed`，不得为了“更详细”而回传或落盘任意工具文本。
- 系统日志只持久化白名单字段（级别、模块、稳定事件码、request ID、route pattern、状态码和
  耗时），不保存 query/body/header、任意错误文本、媒体或宿主路径、logger 原文；最多保留
  5,000 条并使用有界 keyset 分页。
- 更新检测只访问固定官方 HTTPS Release API，使用 5 秒超时、1 MiB 响应上限和六小时缓存；
  不接受用户提供 URL，不发送媒体路径、库名、账户或实例标识，失败不改变 readiness。

S4-005B 已验证生产原媒体 route 只接受资产 ID，并经 session、SQLite 索引、
`internal/media` fingerprint 校验和 `internal/files.Root.Open` 读取；poisoned traversal、
源变化/缺失/offline、错误脱敏、16-stream admission 与 cooperative cancellation 均有回归
证据。Linux arm64 production route/nested mount 已通过；当前 amd64 QEMU 缺失 `openat2`
时应用在启动阶段失败关闭，不能用该模拟环境代替恢复后的 native amd64 PR CI。

## 必需测试

`FTR-UIF-001` 已补充并通过以下安全证据：

- 管理员改密验证当前密码、限流、CSRF、`no-store`、请求体上限和统一凭据错误；成功时当前
  session 保留、其他 session 在同一事务语义中撤销，事务失败不得留下半更新状态；
- 目录 `q` 只查询可靠索引并只接受资源 ID、规范文本和受控枚举，不接受路径或触发文件系统
  遍历；错误与日志不得泄露宿主机路径或 SQL；
- 缓存清理只调用 thumbnail/cache owner，只删除 `/app/data` 下可重建派生物；不接受路径，
  盘满、权限失败、重启和重复请求都必须保留配置、索引和原媒体；
- 合成媒体 sentinel 在账户、目录过滤和缓存清理纵向链前后逐项、逐字节一致。

账户事务、目录索引边界和 cache owner 证据由
[UIF-S2 Backend Evidence Ready](gates/MVP-2026-07-23/uif-s2-backend-evidence-ready.md)持有；
真实浏览器链在只读 `/library` 前后比较全部路径与 SHA-256，见
[`UIF-403`](evidence/uif-403/README.md)。这不关闭 S5-006B 的物理辅助功能，也不替代最终
发布镜像、网络拓扑与供应链安全签署。

- `..`、绝对路径、双重 URL 编码、NUL 和符号链接逃逸。
- Linux 同设备、跨设备与 self-bind 子挂载拒绝，以及 `openat2`/resolve flags
  被内核或 seccomp 阻断时的失败关闭。
- 目录选择器未认证短路、恶意 parent、文件过滤、隐藏/Unicode 目录、自然分页、游标篡改/
  跨 scope 复用、symlink 和 nested mount 原因码及公开错误脱敏。
- 相同、祖先和后代媒体库重叠，以及两个数据库连接并发创建时只能有一个成功。
- 挂载离线、权限错误、扫描中断和扫描期间路径变化。
- 畸形图片、像素炸弹、超大输入/尺寸、长动画、损坏视频、工具输出超限、超时进程树和
  native 调用返回后的取消安全点。
- HTTP Range、条件请求、无效 Range 和取消播放。
- 管理员初始化竞态、未认证访问、登录限流、会话过期/撤销、退出、CSRF 和反向代理配置。
- root runtime、单一只读媒体根挂载、无 mount capability、真实 tmpfs 磁盘已满和数据库恢复的
  Docker 集成测试；非 Linux 结果不能代替 Linux mount-boundary 证据。

## POST-MVP-5 revision 1 AI 安全边界

- revision 1 不联网下载模型、不上传媒体或查询；容器不新增出站网络要求。API 拒绝 URL、绝对路径、
  相对路径和任意模型名，只接受服务端扫描产生且绑定 scan revision 的 opaque candidate ID。
- `/models` 是可选独立只读来源，不能位于 `/library` 下。扫描和直接读取必须经 `internal/files` 的
  专用 anchored boundary，拒绝 symlink、hardlink、特殊文件、后代 mount、可写来源、目录层级逃逸、
  文件替换竞态和非 allowlist 文件；不得执行包内代码、脚本、动态库或 ONNX external-data。
- managed copy 先做空间安全余量检查，在 `/app/data/models` 同文件系统 staging 中逐文件验证固定大小
  与 SHA-256，再 no-replace 原子发布并 fsync parent。验证失败、取消、盘满或重启不能改变 active pointer。
- direct 模型每次进程启动、session load 和显式刷新前重验来源身份、只读状态与 hash；变化后标记
  unavailable，不自动选择别的包。已有派生数据可保留，但依赖文本 encoder 的查询失败关闭。
- 查询原文属于敏感临时输入：不写请求日志、系统事件、任务、metrics 标签、trace、诊断包或 cursor；
  只允许内存中有界处理。embedding 和 query fingerprint 不通过 API 返回。
- 日志仅允许 opaque library/model/generation/operation/request ID、稳定阶段、计数、耗时与稳定错误码；
  不记录媒体路径、模型文件名、容器/宿主路径、tokenizer 内容、runtime 原错、向量或相似查询样本。
- activation runtime 只接收来源 owner 已打开的 Linux 文件句柄和 `/proc/self/fd/<fd>` 临时引用；不接收
  `/models`、`/app/data/models` 或宿主路径，不把完整 ONNX 图复制进 Go heap 或二次临时包。句柄在
  session load 完成前保持打开，非 Linux 生产边界失败关闭。
- ORT adapter 只在显式 `onnxruntime` build tag 下启用并要求精确 1.28.0；默认构建返回稳定 unavailable。
  adapter 不读取或传播 ORT 原始错误消息，只保留稳定操作名与数值错误码。`CreateSession` 不能由 ORT
  1.28 C API 中断，因此取消只在 native load 前后生效；不得用泄漏 native session 的孤儿 goroutine
  伪造 hard timeout。若硬超时成为发布要求，必须先用 ADR 接受进程隔离及其生命周期边界。
- 管理模型、开关、重建、取消与清除均要求管理员 session、CSRF、请求体上限、操作限流、幂等键和
  revision/ETag 并发控制；强 `If-Match` 只接受服务端签发的 canonical positive-decimal revision，
  不把 `r01`、`r+1` 等文本别名解释为同一 validator。语义查询按验证后客户端地址独立限制为每分钟 30 次，不与目录浏览或文件名
  搜索共享 operation key；进程内 interactive admission 容量为 1，已占用时立即返回 `semantic_busy`，
  不在 native inference/session 前累计等待请求。
- AI/semantic mutation JSON 只允许一个严格 value 且硬上限为 4 KiB；decoder 必须观察第 4,097 个字节，
  不能把 `LimitReader` 截断造成的 EOF 当成合法结束，也不能用超长尾随空白绕过 admission；
  `Content-Type` 按媒体类型解析，接受 `application/json` 的合法参数但拒绝其他或畸形媒体类型。
- AI operation 的 Get/create/transition repository 返回值必须由状态机 owner 校验 state、phase、
  finished time 和稳定 error code 的合法组合，并绑定请求的 operation identity、owner 与单步 revision；
  不一致状态或另一条合法 operation 均不得进入 worker/API，外部只返回脱敏内部错误。
- 安装/激活 admission 即使不经过 OperationService，也必须在 ManagementService 出口校验 operation
  kind、激活 model 绑定、Created/Replayed 语义及新建 queued/revision-1 状态，才可生成 Location/ETag。
- install/activation admission 必须在 worker wake 前验证 queue 返回的 idempotency/request、
  candidate/source 或 model/availability owner 及 operation；失败不得发出 wake signal。
- 固定窗口 limiter 遇到 wall-clock 回拨时必须保留已消耗额度并把未来 bucket 起点重基准到当前时间；
  不因回拨恢复登录或 semantic 配额，`Retry-After` 不越过窗口，4,096 bucket 满载也须在一个窗口后恢复。
- semantic cursor 必须在 snapshot 或解码前满足 8～2,048 bytes；搜索 HTTP request-target 的 raw query
  在 `url.ParseQuery` 前受 16 KiB 硬上限约束。repository page 必须有界、有限、唯一、有序且属于当前
  snapshot/scope；catalog hydration 还须逐项匹配 asset/library ID 并保持 online，否则按损坏、stale 或
  not-ready 失败关闭，不能利用畸形 token、异常 adapter 或 ID 复用放大资源或返回跨库资产。
- 所有共享 cursor 只接受 canonical Raw URL Base64；解码后重编码必须与输入逐字节相同，禁止利用末尾
  未使用 bit 为同一 AEAD bytes 制造多个 token 表示。semantic snapshot 的 generation、coverage、成员与
  排除集合也必须由 capability owner 复核，多库 coverage 聚合必须拒绝整数溢出；内部损坏不得映射成
  客户端输入错误或返回回绕计数。
- transport 日志只记录 method、status、request ID 与 duration，不记录 URL/query string；即使 tokenizer、
  encoder 或 vector adapter 返回包含查询正文的内部错误，public error 与结构化日志仍必须由 HTTP owner
  收敛为稳定代码，禁止透传 runtime message。
- semantic canonical owner 在 native tokenizer 前以大小写不敏感方式拒绝 literal `</s>` 和 `<unk>`；
  禁止用户查询注入注册 control token，拒绝结果统一映射为不回显原文的 `invalid_request`。
- 清除只覆盖所选库可重建语义状态；原媒体、收藏、人工标签和全局模型不在删除端口能力内。安全测试
  必须比较操作前后 synthetic `/library` sentinel 的路径、mtime 和 SHA-256。
- runtime/权重在 Release Gate 前必须有双架构 SBOM、许可证/再分发、notices、provenance、漏洞与 VEX
  签署。签名发行 catalog 与在线镜像不是 revision 1 安全边界，未来增加时必须重新评审。

人脸和人物数据不在 revision 1，因此本 migration/API 不得创建或暴露 biometric 表、裁剪、endpoint
或诊断字段。未来 Slice E 必须先通过独立隐私 Gate。

## POST-MVP-5 revision 2 C+D+E 安全与隐私合同

Revision 2 已把 C/D/E 纳入冻结范围；上一段只描述 revision 1 历史边界。`INT-S1R2` 通过前仅允许
OpenAPI/data 合同；当前 S2C fail-closed production migration、handler、worker 与 `internal/face` 已
Backend Ready。CR-2026-022 没有取消 `INT-015` 隐私和合法数据要求，而是把最终批准归入 S4 Release Gate。

- C 的词表只引用管理员现有 tag ID；模型不能写自由文本。接受 suggestion 必须原子复核 generation、
  vocabulary、source fingerprint、suggestion revision 和 curation revision。dismiss/accepted 是应用状态，
  不作训练输入；普通 disable/derived clear 不删除它。
- D 只读消费 thumbnail/media owner 发布的完整 active storyboard，不接受 API 路径或直接调用 FFmpeg。
  query 正文仍不得写日志/cursor；返回只含 asset ID、有限 score 和命中帧 ordinal/timestamp/plan size。
- E 默认关闭且按库明确告知。embedding、observation、crop cache、anonymous cluster 归为高敏感可重建
  数据；person name、manual assignment/exclusion/cannot-link/audit 归为不可重建应用状态。向量仅存受控的
  应用数据库，不进入公共 API、日志、诊断或支持包；经认证的人脸审核 API 可按既有合同返回相对框、
  人物管理 API 可返回人工姓名。不得把内部向量、裁剪字节/宿主路径、原始模型错误或推测身份属性泄露到外部。
- 每个 SQLite 连接强制 `secure_delete=ON`；清除验收在成功删除并 truncate WAL 后扫描活动 DB/WAL/SHM
  的唯一 canary。该边界不承诺擦除 operator backup、storage snapshot、文件系统 journal 或合同明确保留的
  应用状态，相关副本仍由部署者按保留/销毁策略处理。
- face runtime 只能产生匿名 core/edge，不能创建姓名或自动合并 named person。只有合法真实 ground truth
  证明 core precision ≥99.5% 时才开放整组操作，否则强制逐脸或小组确认；edge 永远逐项确认。
- core component 必须满足 smallest-ID anchor coherence，不能因 `A≈B`、`B≈C` 就传递合并未达到
  core 阈值的 `A/C`；bridge candidate 只能降为 edge。cannot-link 仍在 anchor/model 判断前失败关闭。
- 未注册的 production candidate adapter 仍只接受 capability owner 已打开的媒体流和模型 FD：libvips 解码
  强制格式/字节/像素/长边上限，ONNX session 只接受 `/proc/self/fd`、精确 role/大小和冻结 tensor ABI；
  detector/align/embed 输出只在内存中流转。模型、输入 path、crop、landmark、向量和 raw runtime error
  不得进入日志/诊断。S4 Release Gate 前 app composition 继续禁止注入该 adapter。
- face format v3 只接受不可拆分的 detector/embedder/threshold-profile 组合包。threshold profile 由严格
  canonical owner 解析并绑定质量 summary hash，不接受 manifest 或 profile 自报 pass/整组授权；face activation
  只更新独立 face generation，不得退休 semantic generation 或提前切换任何库的可见 cluster。内建 catalog
  没有获批 entry 时 application composition 必须继续失败关闭。
- manual constraint 在换代/重聚类前优先应用；cannot-link 冲突、stale revision、ambiguous lineage 均失败
  关闭。人物 merge 全事务，不能自动或部分合并；undo 仅允许最近且未被后续 revision 修改的事件。
- derived clear 与 manual relationship clear 是两个权限相同但危险度不同的操作；后者必须显示范围与影响
  计数、精确二次确认、CSRF、If-Match 和 idempotency。两者均不得触碰 `/library`。
- E 的 fixture intake 必须记录来源、授权、purpose、最小保留期、访问 owner 和删除证明；真实 biometric
  ground truth 不进入 git、普通 CI artifact、日志或公共对象存储。访问与导出失败关闭并有审计。

所有 C/D/E transport 使用独立固定窗口 bucket，并共享有界总 bucket；后台 admission 低于浏览任务且全局
有界。原模型/runtime、双架构最终镜像必须具备审核 hash/license、SBOM、VEX、notices、provenance 和安全
签署；缺签、漏洞无处置或来源失效时能力 unavailable，保留最后可靠派生状态与人工应用状态。
