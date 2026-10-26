# CR-2026-043 同图人物防误合并

- 授权：CR-2026-042后继续开发；POST-MVP-5既有FR-INT-010/012/016、NFR-INT-006；FR-PEOPLE-022～024。Owner：face审核/SQLite事务、AssetFace只读投影、API生成边界及单图整理。
- 后端先行：cannot_link原来只核对检测revision，不校验之后改变的人工归属；新增成对sourcePersonId/targetPersonId及expectedSourceRevision/expectedTargetRevision。新写入必须是两位不同active人物的当前bound人脸及确切版本；无归属guard的新请求冲突，旧已完成同键回执仍可重放。原语义检测修订不替代人工版本。
- 持久化：沿用face_cannot_links及现有undo JSON快照，记录本次两个人物和anchor版本；新快照的撤销拒绝后续归属/人物/anchor变化。旧快照仅按其实际已记录的link版本撤销，不伪造历史guard。无数据库迁移/模型身份变更或新事务owner。
- 读取：AssetFace增加可选cannotLinkFaceIds，只包含同一照片当前投影内的已标记关系、排序去重及最多63个opaque ID。新服务始终返回数组，缺字段表示旧服务/未知，不能当作无关系；同一有界只读事务读取人脸/关系，不暴露其他照片/库的人脸ID。
- UI：单图明确选一张已归类人脸，再选同图另一位不同人物的已归类人脸，核对两脸和姓名后确认禁止合并。不是自动识别身份，不移动/拆分/命名照片。相同人物需先明确移出/移动；已有关系显示并禁用重复创建，未知状态/图片失败暂停。仅原确认框内显式同key重试，成功回执进入统一撤销。
- 范围：本轮只提供同图入口，后端既有跨图关系合同仍保留。无全局关系历史或强制合并；不重算模型。原6010、原件、人工整理不写入，不连接server222，10源问题暂留，不推送或发行。
- 验证：先红测证明旧无guard请求会写入，再覆盖guard/无写入冲突、重复/同键/历史重放、撤销后置保护、merge阻断及同图投影边界；正式组件原型/单测/三浏览器/真实模型独立Docker后记录实际结果。

状态：本切片开发及本地Docker验收完成；原6010未部署此候选，不代表其他AI/质量/发行完成。

## 验证过程与首轮问题

- 红测实际失败：旧实现的无人物guard请求成功写入关系；修复后同一测试预期409/无写入通过。
- 全量首轮被模型下载合成测试的宿主空间门槛拦住（insufficient_space）；转至专用扩展盘TMPDIR后，该项通过。并发负载下既有视频独立超时测试出现一次deadline；未放宽生产或断言预算，按原预算再次执行通过；保留该次超时，不将其说成已证明的生产缺陷。
- 既有TestFaceDetachedReassignment/linked夹具遗漏新增guard，已按其两位人物实际revision=2适配，重新执行全量通过；不能把旧fixture失败当生产成功。
- 真实模型首轮前4阶段通过，新增第5阶段错误地假定原source归属和excluded人脸同图；根据实际person assets选择同图另一张脸，保存其原状态并在末尾恢复；第二次全5阶段通过。最终英文说明更正后的e6639025再次全5阶段通过。
- 首轮Story播放未等待图片load，读取到disabled radio，不能打开确认框。改为等待正式组件可编辑状态；隔离静态Docker中英390/1920四项通过，不放宽生产图片检查。
- 第一候选91582bf9的浏览器矩阵发现测试“已有关系”断言与实际文案不同，且英文介绍仍声称每次只影响一脸；修正断言和英文介绍。旧矩阵在已确认失败后中止并清理，不能称通过。最终固定e6639025矩阵独立重跑，不混用前一镜像的结果。

## 当前已验证

- `make fmt-check arch-check generate-check lint`通过；Go全量、integration及`make test-e2e`容器runtime smoke在专用扩展盘TMPDIR通过。
- 定向`go test -race ./internal/face ./internal/api ./internal/store/sqlite -run 'TestCannotLink|TestGuardedUndo|TestFaceProjections|TestReviewService' -count=1`通过；覆盖逆序pair、旧回执/快照、source/target/anchor变更、merge阻断与跨图投影边界。
- `npm --prefix web run check`通过：76文件504项、audit:high、生成/架构/visual inventory/types及Storybook；最后Story等待状态更正后另行build和types通过。
- OpenAPI lint有效，保留两个health endpoint缺少4xx的既有warning，未跳过。
- 正式Story四项中英深浅/390与1920通过并检查截图；768/1440两档中间宽度另由最终矩阵覆盖。
- `node tests/release/face_cannot_link_smoke.mjs`使用固定e6639025、既有真实buffalo_l候选与合成双图通过：分组确认、单图归类、统一撤销、人物恢复、防误合并共5阶段。新关系从正式API读回且双向，只在同图显示；新无guard409、真实提交丢响应→同键false/true、合并拒绝、解除后合并及undo通过。最后1assigned/1excluded/2anonymous，其他脸、非空人工标签、源hash、重启和原测试容器内database-check通过。
- 旧业务探针face-business-probe已适配新guard（包含已有assigned exclusion的source保护）；本轮未执行该历史多阶段探针，不沿用其旧通过结果。

## 最终结果

- 最终本地arm64候选：`sha256:e663902599f37604ca066e4d42e399811d0fe35148dba67d0ae15c6c34f70085`（`foliopath:local-cannot-link-20261025`）。该固定镜像三浏览器324/324通过：新增同图关系36项，加原人物恢复/撤销/单图/目录/分组/人物详情/上下文侧栏兼容回归；没有新失败。四档宽度、中英深浅、axe、取消焦点、无横向溢出及同键重试/冲突/撤销覆盖通过。
- 同固定候选真实模型5阶段、独立静态正式Story4项、76文件504项前端以及Go全量/集成/race与runtime均按上文边界通过。generic runtime smoke是单独构建的d4449685，不冒充最终SPA镜像；最终镜像由真实模型和324项浏览器验证。
- 目检1920中文工作区和390英文确认框截图；测试容器、临时数据卷及私网已清理，保留可复核日志/截图hash。
- [机器证据](../evidence/people/2026-10-25-cannot-link-local.json)。原6010未部署、原件/原库人工整理/任务未修改，未连接server222/推送。10项源问题继续暂留。OCR、全局/跨图关系管理与历史、质量/容量/跨架构和发行仍独立保留。
