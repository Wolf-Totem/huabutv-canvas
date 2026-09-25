# 画布TV 下一版开发文档

| 字段 | 内容 |
|---|---|
| 状态 | 已定方案，实施中 |
| 日期 | 2026-09-25 |
| 当前版本 | 本地 / 现网 `v1.5.33`，仓库 `Wolf-Totem/huabutv-canvas` |
| 上游对照 | [ddcat-ai/open-ai-canvas](https://github.com/ddcat-ai/open-ai-canvas) 正式 `v1.5.7`，预览 `v1.5.7.1` |
| 现网 schema | **41** |
| 原则 | 用户可见文案与 CHANGELOG 不出现第三方站名；密钥不入库；发完推 GitHub |

已确认：首页海报后台可换图/换文案/换链接；广场只清导入记录（`ext:`）；上游插件与大功能一起做、禁止整仓 merge。

---

## 0. 范围

| # | 目标 | 不做什么 |
|---|---|---|
| A | 画布连线常驻流动（含拖线） | 不改连线数据模型、不改命中热区 |
| B | 清导入广场作品，导 80 张带流程图的展示卡，挂管理员账号 | 不删分类、系统用户、站点预设素材；不清用户申请作品 |
| C | 跟上游影策可隔离的插件 + 大功能 | 禁止整仓 merge（会冲掉门户/广场/代理） |
| D | 首页红框换成「海报轮播 + 开始创作 + 四格」；后台管图、文、链接 | 不改左侧「绘无限造未来」与底部弧形精选的产品定位 |
| E | README / About 按画布TV 写，保留上游声明，去掉二维码 | 不删 `NOTICE` / `LICENSE` / 贡献者与赞助商表 |

---

## 1. 里程碑

| 里程碑 | 产出 | 验收 |
|---|---|---|
| **M0 文档与基线** | 本文进仓库 + README 署名口径 | 文案口径冻结；`NOTICE` 仍在 |
| **M1 连线流动** | 已连线与拖线草稿都从上游节点流向下游 | 空闲可见流动；平移/减弱动效时停；点线、右键正常 |
| **M2 导入拓扑** | 文本/分组节点可进快照，连线不再被丢 | 测试画布导入后 `connections.length >= 1` |
| **M3 广场重置** | 只删 `ext:` 作品；导 80 条到管理员；精选 JSON 同步 | listed 导入=80；查看过程是真流程图；用户申请作品仍在 |
| **M4 首页模块** | 红框 1:1 布局；后台换海报、换字、换链接 | 改 href 立即生效；海报图可上传替换 |
| **M5 上游能力** | 插件包 + 大功能按 §5 清单移植 | 后台能启用新插件；表格/成本/历史等按清单可点 |
| **M6 发版** | `v1.5.33`，现网 + `git push origin main` | 健康检查通过；CHANGELOG 无第三方站名 |

每步一个可验证 diff。M5 内部再按功能切片。

---

## 2. M1 连线常驻流动

**现状：** `web/src/components/canvas/canvas-connections.tsx` 已有虚线流动 + 流光，方向已是 from → to，但只在悬停/选中时画。可编辑画布常态线是 Leafer 实线。`.canvas-connection-draft` 有 CSS，拖线没用上。

**改动：**

1. 只读/参观：`showVisual` 为真就画 flow + comet，不要求 `emphasized`。
2. 可编辑：不要把 SVG 改成 `visualMode="full"`（会和 Leafer 双线）。给 Leafer 线加 dashOffset，或加一条 `pointer-events:none` 的 SVG 流光。
3. 鼠标拖线：草稿套 `.canvas-connection-draft`，光从头节点流向指针。
4. 透明 16px path 仍是唯一命中层；平移与 `prefers-reduced-motion` 停动画。

---

## 3. M2–M3 广场：只清导入 + 80 张有流程图的卡

**根因：** `libtv_adapter.go` 只留 image/video/material-style。文本被丢后连线两端对不上，快照 `connections: []`。

**适配器：** 未知类型降级为 `text` 节点，保留坐标；两端都进 mapping 才写 connection。种子过滤：`节点 >= 4` 且 `连线 >= 1` 才入库，凑满 80。

**只清导入：**

- 删 `source_project_id LIKE 'ext:%'` 及其 likes/events/tags/snapshots。
- 不动：分类、`plaza-system`、广场开关、用户申请作品、`web/public` 预设图视频文。

**挂管理员：** 每条先写入管理员 `canvas_projects`（以后可在 `/canvas/:id` 改），再 listed 到广场，`author_id = 管理员`。封面=第一张图，成片=第一条视频。刷新 `plaza-featured.json`，弧形轨与 `/create` 广场优先读 API。

现网先 `SELECT` 导入 vs 用户作品数量，再走 `--reset-imported` 或管理员清理接口。

---

## 4. M4 首页红框 + 后台换海报/换链接

红框替换居中「开始创作」浮钮。英雄视频仍铺底，左侧文案、底部弧形精选保留。

```
[ 海报轮播：中卡 + 左右露出 / 箭头 / 圆点 ]
[ 开始创作大卡(+)     ] [ 磁贴1 ] [ 磁贴2 ]
                       [ 磁贴3 ] [ 磁贴4 ]
```

扩 `appearance.landing.heroShowcase`（不另起表）：

```ts
banners: [{ id, title, imageUrl, href, openInNewTab?, workId? }]
create:  { title, subtitle, href }
tiles:   [{ id, title, subtitle, badge?, href, icon? }]
```

后台「站点外观」新分组 **首页海报与创作模块**：

- 海报：增删排序；上传/替换图片；标题；超链接；是否新窗口；可选绑定管理员画布 `workId`
- 开始创作：标题、副标题、链接（默认 `/create`）
- 四格：每格标题、副标题、角标、链接
- `href` 用现有 `validHomeHref`（`#` / 站内路径 / `https`）

海报图走管理员资源上传，公开 URL 用 appearance 资源接口。默认 banners 的 `workId` 指向 M3 导入的管理员画布。

文件：`manchuang-landing.ts`、`manchuang-home.tsx/.css`、`appearance-settings-page.tsx`。

---

## 5. M5 上游影策：插件 + 大功能

本仓 schema **41**（36–41 是代理/短信/广场）。上游快照到 **35**。本仓 24–35 已用同名 checksum 登记为现网已有（`acknowledgeExistingSchema`），缺的是 **代码** 不是再跑迁移。上游若还有新表，**只能挂本仓 42+**。

**禁止整仓 merge。** 对照 `.local/upstream-157` + 上游 `main`。冲突时保留本仓门户、广场、代理、登录弹层。

### 5.2 插件（全部进）

`dashscope-wan3-video`、`official-payment-epay`、`official-payment-zhifufm`、`volcengine-ark-agent-plan-seedream/seedance`、`image-tools`、`a6api`、`metaso-h3`、`antigravity-proxy`、`lxmone-video-suite`。

本地已有佳速三件套、虎皮椒等，不覆盖。

### 5.3 大功能切片

| 切片 | 上游 | 注意 |
|---|---|---|
| 多维表格 / 分镜关键帧 / 批量生成 | v1.5.7 PR #549 | 本仓已有 BatchTable，按上游补齐；参考图上限 10 |
| 模型分组 + 积分成本价 | v1.5.4 | 列可能已在现网；补后台与 CSV；用户接口不返回成本 |
| 收入/成本/利润汇总 | v1.5.6 | 与本仓支付/会员并存 |
| 隐藏连线、工作条、远程媒体稳定 URL | v1.5.5 | 与 M1 同时存在：隐藏管显示，流动管已显示的线 |
| 画布版本历史 / banner / 支付对账导出 | v1.5.1 | schema 名已登记，补 UI |
| Live2D 配置导入 | v1.5.7.1 | 跟上游实现 |
| 素材批量删除、Agent 稳定性、声明式文本流、模型选择 | v1.5.7.1 | 逐文件 diff |
| `main` 未发版：产出/选用模型分离、注册协议可配、角色卡封面、大画布性能 | main | 新表用 schema 42+；与短信/邀请冲突时本仓优先 |

不整段替换：`main.tsx` 分流、`public-application`、广场路由、`manchuang-home`、streamer 后台。

---

## 6. README / About

**写上：** 画布TV 基于 [ddcat-ai/open-ai-canvas](https://github.com/ddcat-ai/open-ai-canvas)（影策）二次开发；官网 canvas.j11.net。

**保留：** `NOTICE`、`LICENSE`、Infinite Canvas 声明、赞助商表、贡献者表。

**去掉：** `assets/wx.jpg` 二维码、上游演示站和测试账密。

---

## 7. 写代码顺序

```
M0 README 署名
M1 连线流动
M2 导入适配器
M3 现网清 ext: + 导 80 + 刷新精选
M4 首页模块 + 外观后台
M5.2 插件包
M5.3.x 大功能按表逐片
M6 v1.5.33 + CHANGELOG + 现网 + git push origin main
```

M3 依赖 M2。M4 默认海报 `workId` 依赖 M3。M0 / M1 / M5.2 可并行。

---

## 8. 风险

- 带连线的可复制源不够 80：放宽到 node>=3 且 conn>=1；仍不够如实报告，不拿无流程图的凑数。
- 外链媒体：封面/成片尽量深拷到管理员资源。
- 上游冲突：按文件白名单合。
- 新迁移只加 42+。
- 流动线多：平移停动画；减弱动效关闭。

CHANGELOG 允许写：连线流动、精选重导、首页海报可后台配置、同步上游协议与表格/成本等。禁止写第三方站名。
