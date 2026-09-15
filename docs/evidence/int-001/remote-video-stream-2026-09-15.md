# 视频流式编码与真实模型链路

2026-09-15，POST-MVP-5 / CR-2026-030，Owner semantic；本地 Docker Linux/arm64。

## 实现

VideoProcessor 的完整计划循环改为消费 StreamImageEncoder；新增 NewStreamVideoProcessor，
旧 NewVideoProcessor 通过既有 legacyStreamEncoder 保留旧预处理/张量语义。
复用同一计划循环、EncodeEmbedding、帧关闭和仓储接口，无第二个 worker。
不修改 app/run.go、模型激活、迁移或 API。主应用配置/服务模型记录仍未完成。

## 已执行与结果

构建 `docker build --target remote-ml-regression -t foliopath-remote-ml-regression:local .` 成功。

```sh
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache \
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm regression \
  '/out/semantic.test -test.v -test.timeout=3m && /out/sqlite.test -test.run "TestVideoEmbeddingRepository|TestVideoJob" -test.v -test.timeout=3m'
```

semantic 全包 PASS；SQLite 7 项视频仓储/任务测试 PASS，联合命令 exit 0；二进制启用 race。
新增单元覆盖成功、推理前/中退役、异常向量、取消、缺依赖；成功时四帧 512 维，
失败时零提交，所有帧流（包括未处理的帧）恰好关闭一次。

随后按 [专用镜像说明](../../../services/inference/README.md)从镜像独立读取身份，
启动候选服务，以 Compose 覆盖文件运行 `persisted-probe`。
首轮测试夹具使用 JPEG，被原有 storyboard WebP 合同拒绝；未改合同，转换合成夹具为 WebP 后
重建重跑 PASS（4.30 秒，exit 0）。

- 原有三张合成图片任务、数据库重开和文本检索仍通过。
- 四张合成 WebP 帧经 imagevips、身份绑定 HTTP、真实离线 AI 返回向量，完整计划成功存入 SQLite。
- 真实文本向量检索返回一个四帧视频计划；删除临时测试数据库的一帧后，该不完整计划不再返回。
- `make arch-check` 宿主静态检查 PASS；`git diff --check` PASS。

## 限制

这是合成 storyboard，不是 ffmpeg 解码或真实视频测试。未测试生产启动、浏览器视频搜索，
不证明模型质量；代次仍通过测试夹具创建，不是生产激活。未跑全仓 lint/生成/E2E/完整 SQLite。
没有读取原媒体、没有 server222 操作、没有发布或提交。服务模型不能伪装为旧 direct 模型目录，
后续仍需定义真实服务模型持久化与激活合同。
