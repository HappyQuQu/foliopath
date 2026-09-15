# 视频移除主进程闭环与设置提示（2026-09-15）

CR-2026-030 / POST-MVP-5 / FR-INT-006/007/015/016/017，Owner app/API/前端 ai。

## 主进程实测

本地 Docker 实际 distroless 应用 + 专用离线 AI，合成媒体和独立数据卷。
app-chain.mjs 新增 FOLIOPATH_TEST_VIDEO_REMOVAL=1：只允许指定临时夹具目录，
将 3 秒合成视频移至新建临时保留目录，finally 放回；不永久删除媒体。

- 初始 2 视频：1 个完整 4 帧结果、1 个 1 秒降级样例。
- 扫描确认移出 3 秒视频后：eligible=1、completed=0、degraded=0、failed=0、
  complete=false，搜索无结果，剩余视频为待处理。
- 新 missing 回填后：degraded=1、completed=0、complete=false；历史作业状态和
  revision 与删除前一致。闭环 PASS。
- 合成视频已恢复并请求扫描；再运行基础图片/视频链路 PASS，恢复为 1 个有效视频结果。
- 脚本每轮使用新任务幂等 key，避免把旧成功作业重放冒充本轮回填。

## 界面发现与修复

实际浏览器 /settings/ai 显示图片语义模型可用、图片覆盖率 1/1，但顶部把旧内置
运行时未就绪描述成“当前推理组件未就绪”，且详情仍写独立服务尚未接入。
修正中英文文案：明确内置状态只适用于兼容维护，独立服务以当前模型的服务端状态为准。
重新构建实际镜像并在浏览器登录后确认新提示及展开的部署说明。

人物区仍显示载入错误；API GET /libraries/lib_1/ai/faces 返回 404。
源码明确 face read/control 路由因发布 Gate 未通过而未接入主进程，不是网络故障。
本轮未绕过 Gate，也未假称人脸已实现或把 404 隐藏成正常状态。
概览 1/1 是图片覆盖率，不是视频全部完成；完整视频 UI 展示仍待验收。

## 检查与边界

- 本地实际应用镜像、web 测试镜像构建成功。
- Docker 前端 AISettingsPage.test.tsx：14/14 PASS。
- make arch-check、git diff --check 通过。
- 浏览器真实登录/设置页/展开部署说明复查完成；不等于全部浏览器自动化 E2E。
- 未访问 server222，未使用真实照片；没有执行完整七项检查或双架构验收。
- 测试后服务停止；保留测试卷、已恢复的临时合成文件和空保留目录。
