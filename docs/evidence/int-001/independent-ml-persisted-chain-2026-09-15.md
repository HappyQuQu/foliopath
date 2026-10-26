# 真实独立 AI：回填入库、数据库重开与文本检索

2026-09-15；POST-MVP-5 / CR-2026-030 / ADR-0019；接入前隔离验证。

## 已执行

本地 Docker Desktop Linux/arm64。测试镜像 config digest：
`sha256:d934ec4e50b489c91ff3b570da799599af13e91fae401d28d3b7e294af897e2d`。
构建命令 `docker build --target remote-ml-regression -t foliopath-remote-ml-regression:local .` 成功。
初次构建遇到测试代码 float32/float64 类型错误，显式转换后重建成功。

```sh
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache \
docker compose -f spikes/int001-remote-ml/compose.yaml up -d machine-learning
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache \
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm persisted-probe
```

`TestRemoteSemanticPersistedProbe` PASS，3.14 秒，exit 0。
测试二进制以 `-race -tags=libvips,remotemlprobe` 编译。

1. 临时 SQLite 创建合成目录、图片、代次和任务，绑定明确的测试空间身份。
2. 既有任务领取/分页处理器读取内存生成 JPEG；imagevips 编码安全 PNG。
3. BoundEncoder 检查代次后调用真实离线 ML；3 张图片向量由既有事务提交，任务 succeeded。
4. 关闭并重开临时数据库，重新组合 BoundEncoder；真实文本推理后执行存储向量检索。
5. 返回 3 条持久化匹配，检查分数有限且在余弦范围内。

ML 容器固定镜像：
`ghcr.io/immich-app/immich-machine-learning:v3.1.0@sha256:5a0839dc5303cd7215bcd2180a26aed3af41675aefb3e75e5157e9f10ad16e6e`。
运行检查端口绑定为空，仅 `/cache` 挂载且 RW=false。probe 不挂载宿主媒体或数据库，
仅加入 internal 网络并使用限额 tmpfs；未连接 server222，未使用 coser。

同一镜像复跑 `docker compose -f spikes/int001-remote-ml/compose.yaml run --rm encoded-probe`
（相同缓存环境变量）通过：PNG/JPEG/GIF/WebP 与文本真实离线推理，0.47 秒，exit 0。
静态宿主检查 `make arch-check` 通过。没有执行本轮全仓 lint/生成/E2E/SQLite 全包回归。

## 证据边界

模型身份是合成夹具，代次通过测试 SQL 建立，不是生产激活。
所有图片内容相同，只验证向量链路和持久化，不证明检索质量、人物识别或模型准确率。
没有扫描真实原图，没有经过主应用 HTTP API/UI，也不是完整应用重启验证。
主应用配置、部署事实核验与生产激活仍未完成。本轮未修改生产组合或发布镜像。
