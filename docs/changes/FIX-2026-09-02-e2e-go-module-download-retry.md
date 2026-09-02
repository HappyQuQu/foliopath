# FIX-2026-09-02：端到端镜像依赖下载抗瞬断

- Slice：POST-MVP-5 revision 2 / 全 S2 验证面
- Gate：S2 backend evidence verification
- 不变式：不改变生产依赖、版本或镜像内容；测试失败必须区分外部下载故障与产品断言失败

## 变更

- 测试专用应用镜像的 `go mod download` 使用 BuildKit module cache，并在连接提前关闭时进行最多三次
  有界重试。
- 后续 `go build` 复用同一个只读语义的 module cache，避免下载步骤成功后在新层重新获取相同依赖。
- 没有修改 `go.mod`、`go.sum`、生产 Dockerfile、模型来源或任何发布 Gate。

## 验证

- 首次与直接重试均在 `proxy.golang.org` 返回 `unexpected EOF`，尚未进入产品断言。
- 加入有界重试后，`make test-e2e` 成功输出 `application container smoke passed`；真实应用启动、健康检查、
  未认证 API、SQLite 初始化、优雅停机、同一数据卷重启和只读媒体 sentinel 均通过。
