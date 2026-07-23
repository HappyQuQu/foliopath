# FS-01：路径边界 spike 报告

## 结论

- **状态：Conditional（有条件通过）**
- **验证日期：2026-07-23**
- **验证环境：macOS（Darwin/arm64）与 Linux 6.12.76-linuxkit/arm64、Go 1.26.4**

当前实现已在 Darwin 和官方 `golang:1.26.4-bookworm` Linux/arm64
容器中验证相对路径与多重百分号编码拒绝、符号链接拒绝、根目录移除/替换检测、
特殊节点跳过、设备号/inode 身份捕获、遍历取消和错误脱敏。新增的真实
`httptest.Server` harness 还验证了不透明 asset ID 到测试 media capability
再到 `internal/files` 的读取链路、单 Range、`HEAD`、`If-Modified-Since`、
非法输入和错误脱敏。该 harness 不是生产 handler，也不证明 handler 已实现或符合现有
权威 `api/openapi.yaml`。

Linux 适配器现已用 `openat2` 的 `RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS |
RESOLVE_NO_XDEV` 打开所有实际文件和目录句柄。高权限 mount namespace 验收证明，
跨设备和同设备 bind mount 都会失败关闭；`Open`、`OpenDir`、`CaptureAt`、
`VerifyAt`、从挂载点开始的 `Walk` 以及从媒体根遍历时都不会进入挂载内容。
修复前同设备用例曾确定失败，修复后相同探针通过。

FS-01 仍为 Conditional：生产 HTTP handler、认证/错误 envelope、只读发布 volume、
运行期挂载消失和 Linux/amd64 尚未存在或验证；非 Linux fallback 也不宣称具备 Linux
`openat2` 的原子 no-mount-crossing 保证。

## 目标与范围

本 spike 验证 FolioPath 是否能在一个可信媒体根目录下：

- 只接受媒体库相对路径，不把用户输入解释为宿主机绝对路径；
- 拒绝 `..`、空组件、反斜杠、NUL、无效 UTF-8 和重复编码后的遍历/分隔符；
- 不跟随目录或文件符号链接，不打开 FIFO 等特殊节点；
- 在扫描前后识别媒体根或媒体库根被删除、改名或替换，避免把失联挂载误判为空库；
- 不跨文件系统边界，并只向上层返回脱敏后的相对路径错误；
- 以有界批次遍历，并响应 `context.Context` 取消。
- 以真实 `net/http` 连接验证代表性的内容读取语义和路径攻击矩阵，但不把测试 harness
  当作现有权威 OpenAPI 的生产实现。
- 在隔离 Linux 容器中验证普通非 root 行为，并用显式高权限探针验证 bind mount
  边界。

本轮不包含生产 HTTP handler、目录选择 API、发布 Docker 镜像、认证中间件、
最终路由器选择或性能/容量测试。

## 实现摘要

实现位于 `internal/files`：

- `internal/pathpolicy` 统一媒体库创建、scanner 和真实文件访问使用的纯相对路径词法策略；它不接触 I/O。原始合法文件名保持不变；百分号解码只用于逐层检查潜在的 NUL、分隔符和点组件，不使用解码后的字符串访问文件系统。
- `Root` 以构建平台对应的 `anchoredRoot` 锚定可信目录，并用打开后的信息以及
  设备号/inode 身份检查根目录是否被替换。
- Linux `anchoredRoot` 使用已有依赖 `golang.org/x/sys/unix` 的 `Openat2`。
  根本身可以是部署提供的 `/library` mount；根以下的每次实际打开都同时使用
  `RESOLVE_BENEATH`、`RESOLVE_NO_SYMLINKS` 和 `RESOLVE_NO_XDEV`。`EXDEV` 映射为
  `ErrCrossDevice`，解析竞态失败关闭为 `ErrChanged`。子目录适配器保留原始边界
  FD 和根相对前缀，每次都从原始边界一次性解析完整路径；即使已打开的子目录随后
  被移出媒体根，也不能继续以该子目录 FD 打开外部内容。
- 该策略允许 `/library` 本身是 Docker/Compose 提供的一个只读 mount，但有意拒绝
  `/library` 之下的任何嵌套 mount，包括管理员显式添加的 bind mount。部署模型
  因而必须映射一个共同大根，再在 UI 中选择普通子目录；不能同时承诺多个
  `/library/<name>` 子 volume。
- Linux 初始化时会以 `"."` 探测完整 `openat2` 策略；`ENOSYS`、不支持的 flags/
  struct 或被 seccomp/LSM 阻断时返回 `ErrKernelBoundaryUnavailable`，绝不静默降级
  到用户态检查。
- Darwin 与其他非 Linux 平台继续用 Go
  [`os.OpenRoot`](https://pkg.go.dev/os#OpenRoot) 加现有身份、symlink 和设备检查，
  每次子路径打开也重新从原始边界解析完整相对路径，避免沿已移出边界的旧子目录
  FD 继续读取；但代码和报告明确不把它记录为 Linux 同等级别的 mount 边界保证。
- `CaptureAt`/`VerifyAt` 对媒体库子根做扫描前后身份校验。`Open`、`OpenDir` 和
  `Walk` 仍执行类型和身份复核；Linux 下它们使用的真实句柄来自上述
  `openat2`，不是先检查后通过另一条路径打开。
- `Walk` 每批读取有限数量的目录项，不为每个条目创建 goroutine；调用方可跳过系统目录并可协作取消。
- `ScanWalker` 在 `internal/files` 内实现 scanner 拥有的 `Walker` 接口，只把媒体库相对路径、跳过事件和设备号/inode 身份交给扫描服务；遍历起始目录必须匹配扫描前捕获的身份，阻止 A → B → A 替换通过仅看首尾的校验。集成测试直接使用该生产适配器。
- 对外错误保留操作名和相对路径，不包含配置的绝对媒体根。
- `tests/integration/http_content_boundary_test.go` 是测试专用 HTTP harness。HTTP
  层只向 `mediaContentUseCase` 传不透明 asset ID；测试 capability 才把索引记录
  交给 `internal/files`。它同时构造绝对路径、遍历、多重编码、NUL 和符号链接形式
  的污染索引记录，验证即使派生数据库内容被污染，HTTP 响应也不会返回原文件或
  泄漏根路径、外部路径、攻击路径和外部文件内容。
- `tests/integration/linux_mount_boundary_test.go` 是带
  `linux && fsboundary` build tag 的高权限验收探针。默认测试不会请求
  `CAP_SYS_ADMIN`；必须在隔离容器或 mount namespace 中显式执行。同/跨设备两个
  主要用例覆盖 `Open`、`OpenDir`、`CaptureAt`、`VerifyAt`、定点 `Walk` 和根遍历；
  self-bind 用例专门隔离验证 device/inode 不变时的 `CaptureAt`/`VerifyAt`。
  显式启用该 tag 却缺少 root、`CAP_SYS_ADMIN`、`mount` 命令或所需设备拓扑时，
  测试会失败而不是跳过，避免 CI 把“没有运行到证据”登记为通过。

Go 的 `os.Root` 文档明确说明它不禁止文件系统边界、Linux bind mount 或
`/proc` 特殊文件。FolioPath 因此只在非 Linux fallback 使用它；Linux 部署目标
由内核级 `openat2` 解析约束承担该安全责任。

## 已执行验证

从仓库根目录执行：

```sh
go version
go test -count=1 ./internal/pathpolicy ./internal/files ./tests/integration
go test -race -count=1 ./internal/pathpolicy ./internal/files ./tests/integration
go test -count=1 -v ./tests/integration -run 'TestHTTPContentBoundary'
```

结果：

- `go version` 返回 `go1.26.4 darwin/arm64`。
- `internal/pathpolicy`、`internal/files`、跨组件集成测试与 race detector 均通过。
- 路径表覆盖绝对路径、UNC 风格开头、`.`/`..`、重复分隔符、反斜杠、NUL、无效 UTF-8、大小写混合及二/三重编码的点组件、斜杠、反斜杠和 NUL。
- 文件系统用例覆盖合法 Unicode/字面 `%` 文件名、内部/外部符号链接、最终根符号链接、根目录删除/替换、扫描中根替换、确定性的 A → B → A 根身份回归、FIFO 特殊节点、取消、错误不泄露绝对根路径，以及设备/inode 身份判断。
- HTTP harness 覆盖完整 `GET`、合法单 Range `206`、越界与多 Range `416`、
  `HEAD`、`Last-Modified`/`If-Modified-Since` `304` 和 `nosniff`。
- URL 输入矩阵覆盖转义绝对路径、转义遍历、双重编码遍历、NUL 和未定义的
  `path` query；均得到 `400`。
- 污染索引矩阵覆盖绝对路径、遍历、双重编码、NUL 和符号链接；均得到脱敏的
  `404`，没有返回外部内容。

### Linux 普通容器证据

镜像与平台：

- Docker Engine `29.5.2`，服务端 `linux/arm64`。
- `golang:1.26.4-bookworm`，digest
  `sha256:b305420a68d0f229d91eb3b3ed9e519fcf2cf5461da4bef997bf927e8c0bfd2b`。
- 容器内 `go1.26.4 linux/arm64`，内核
  `Linux 6.12.76-linuxkit aarch64`。
- 仓库只读挂载，module/build cache 使用容器临时目录；测试进程 UID/GID 为
  `65532:65532`。

下面以 `$REPOSITORY` 表示仓库的绝对路径。执行：

```sh
docker run --rm --platform linux/arm64 --user 65532:65532 \
  --mount type=bind,src="$REPOSITORY",dst=/src,readonly \
  -w /src \
  -e GOMODCACHE=/tmp/foliopath-modcache \
  -e GOCACHE=/tmp/foliopath-buildcache \
  golang:1.26.4-bookworm \
  go test -count=1 \
    ./internal/pathpolicy ./internal/files ./tests/integration
```

结果：完整的 `internal/pathpolicy`、`internal/files` 和 `tests/integration`
在 Linux/arm64 非 root 环境全部通过；另一次带 `-v` 的定向运行确认
`TestUnreadableRootBecomesOffline` 实际执行而不是跳过。这个命令没有请求 mount
权限，所以不计作 bind mount 证据。第一次完整运行在下载依赖时遇到一次
`proxy.golang.org` TLS handshake timeout；使用同一镜像重试后成功，未把失败运行
登记为通过。

修复完成后的最终非 root Linux 回归还执行并通过 `go vet ./...`、
`go test -count=1 ./...` 和 `go test -race -count=1 ./...`，覆盖当前所有 Go
package，而不只 FS-01 定向用例。

普通、无额外 capability 的 Linux 测试还包括：

- 以 `/` 为允许根，确认 `Open("proc/version")`、`OpenDir("proc")`、
  `CaptureAt("proc")` 和 `Walk("proc")` 均以 `ErrCrossDevice` 拒绝现有 `/proc`
  mount，且不读取挂载内容。
- 对 `EXDEV`、`ELOOP`、`EAGAIN`、`ESTALE`、`ENOSYS`、`EINVAL` 和 `E2BIG`
  的失败关闭映射做 Linux 单元测试。
- 打开子目录适配器后把该目录移出媒体根，再尝试通过旧适配器打开其中的文件；
  请求按原媒体根相对路径得到 `fs.ErrNotExist`，没有沿旧子目录 FD 读取外部内容。

### Linux mount namespace 探针

默认测试不编译 `fsboundary` probe，因此应用与普通 CI 容器不需要
`CAP_SYS_ADMIN`。显式启用 tag 时，探针要求可用的独立 mount namespace；缺少权限
会直接失败。验证时只在隔离测试容器中增加 `SYS_ADMIN` 并关闭该容器的默认
seccomp 过滤：

```sh
docker run --rm --platform linux/arm64 \
  --cap-add SYS_ADMIN --security-opt seccomp=unconfined \
  --mount type=bind,src="$REPOSITORY",dst=/src,readonly \
  -w /src golang:1.26.4-bookworm \
  go test -count=1 -v -tags fsboundary ./tests/integration \
    -run '^TestFS01Rejects.*BindMount$'
```

修复前基线结果：

- 跨设备 bind mount 被拒绝；
- 同一 overlay 文件系统内的 bind mount 可被 `Root.Open` 成功读取，命令退出码
  为 `1`。该失败证明仅比较设备号不足以实现产品约束。

接入 Linux `openat2` 适配器后，以完全相同的 mount 构造重跑，结果为：

- `TestFS01RejectsCrossDeviceBindMount` 通过；
- `TestFS01RejectsSameDeviceBindMount` 通过；
- `TestFS01RejectsSelfBindMount` 通过：把媒体库子目录 bind 到它自己，挂载前后
  device/inode 完全相同；`CaptureAt` 仍以 `ErrCrossDevice` 拒绝，`VerifyAt`
  仍转换为离线/根变化。该用例证明结果来自 `RESOLVE_NO_XDEV`，不是身份不匹配；
- 前两个用例都先在挂载前捕获原目录身份，再挂载测试源。挂载后 `Open`、
  `OpenDir`、`CaptureAt`、`VerifyAt`、定点 `Walk` 和根遍历全部按各自公开语义
  失败关闭，且没有观察到 `mounted/secret.jpg`。
- 同一 tagged suite 的 race detector 运行也通过。

所有测试均使用 `t.TempDir()` 动态创建合成目录，没有读取开发者真实媒体库。未运行 benchmark，因此本报告不提供吞吐、延迟或内存数字。

## 未验证与剩余风险

1. **Linux 运行条件**：Linux 必须提供允许上述 resolve flags 的 `openat2`；
   不支持的内核或阻断 syscall 的 seccomp/LSM 配置会使媒体根初始化失败。尚需把
   最低内核与容器 profile 固定为发布前置条件并加入健康检查。
2. **高频动态竞态**：静态同设备/跨设备 mount、根替换、A → B → A 和已打开子目录
   被移出边界均已覆盖，实际句柄也由内核从原始边界原子解析；尚未运行持续
   rename/mount churn 的长时间压力测试。
3. **生产 HTTP 端到端**：测试 harness 使用标准库 `ServeMux` 和测试 capability，
   证明的是架构组合方向与 `net/http` 基础语义；它没有验证生产 router/handler 对权威
   `api/openapi.yaml` 的实现，也没有验证认证/CSRF 中间件、API 错误 envelope 或反向代理。
4. **容器权限与挂载消失**：已验证普通非 root Linux 执行，但尚未以真实
   `/library:ro` volume 模拟运行期 unmount、权限变化、`EIO` 或 `ESTALE`。
5. **平台范围**：已执行 Darwin/arm64 与 linux/arm64；非 Linux fallback 不具备
   已证明的 no-mount-crossing 保证，也不能据此声明 linux/amd64、NAS 文件系统或
   任意内核版本已通过。

## 决策与下一步

- 保留 `internal/files` 作为唯一文件系统访问边界。Linux 同设备 bind mount 缺陷
  已修复并有内核级验收证据；FS-01 仍因生产 HTTP 与发布容器证据缺失而保持
  Conditional，不能以本 spike 代替完整发布门槛。
- 保留 `fsboundary` tagged acceptance probe，并在具备独立 mount namespace 和
  `CAP_SYS_ADMIN` 的安全 CI job 中运行；普通应用容器本身不获得该 capability。
- 将 Linux `openat2` 支持与 seccomp compatibility 纳入发布环境检查；不允许在
  Linux 上回退到 `os.Root`。
- 权威 OpenAPI 已建立；下一步创建 media capability 与生产 handler 时，把当前 HTTP 恶意
  矩阵迁移到生产契约测试，并补认证、错误 envelope、反向代理和客户端取消。
- 完成上述验证前，可继续开发不扩大信任边界的代码，但不得宣称路径安全验证已完整通过。
