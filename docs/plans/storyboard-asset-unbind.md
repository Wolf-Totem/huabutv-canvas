# 分镜关联资产：格子内卸绑定

| 字段 | 内容 |
| --- | --- |
| 日期 | 2026-09-28 |
| 状态 | 待开发（产品口径已定，本文冻结范围） |
| 产品 | 画布TV |
| 现网 | v1.5.55，schema 45 |
| 前置 | v1.5.54 已能把画布素材挂到每一镜，并按文案匹配；关联列显示节点标题 |

---

## 1. 问题

分镜「关联资产」列能挂、能看、不能卸。

芯片点击只打开预览。检查关联默认只增不删。用户要拿掉某一镜上的「张三」，现在没有正规入口：删画布连线绑还在，删张三节点会把图一起丢掉。

这不是绑定写死，是单元格缺卸绑定入口。数据层已经能改 `assetBindings`。

## 2. 目标

- 在关联芯片上卸掉**这一镜**的这一条绑定。
- 画布上的素材节点不动；其他镜的绑定不动。
- 失效绑定（节点已删、芯片显示「已失效」）同样能卸。
- 紧凑分镜表和全屏编辑共用同一行为。
- 卸完后行连线与生成参考图列表与绑定一致，不再把已卸素材送进该镜生成。

## 3. 非目标

- 不改绑定数据结构（仍是 `{ nodeId, role, priority }`）。
- 不在本文重做 Agent 写口；`unbind_assets` 已有，人手走格子即可。
- 不把检查关联改成会删已有绑定。
- 不靠删除画布连线或删除素材节点来卸关联。
- 不做拖出格子删除、不做一次清空全表（除非日后单独开「清空本表关联」）。
- 不改计费、迁库、默认首页、广场 ingest；不改 `handler/agent.go`。

## 4. 语义

| 操作 | 含义 |
| --- | --- |
| 点芯片（非 ×） | 预览素材，与现在相同 |
| 点芯片上的 × | 从**当前行** `assetBindings` 去掉该 `nodeId` |
| 卸绑定 | 不删除画布节点、不改素材库、不影响其他行 |
| 卸失效芯片 | 去掉悬空 `nodeId`，格子恢复「未关联」或只留其余绑定 |

绑定仍是真相；`relation = storyboard-asset-reference` 且 `toHandleId = row:{rowId}` 的连线跟着绑定重建。生成继续读 `storyboardRowReferenceNodeIds`（含该行 `assetBindings`），卸掉后该镜参考图不再含此节点。

检查关联继续只增不删，避免一点检查把人手卸掉的又挂回去。

## 5. 交互

放在 `StoryboardAssetsCell`，不要藏进行菜单。

- hover / 聚焦芯片时右上角出现 ×；触控下 × 常显或易于点到（命中区域不小于 16px）。
- `aria-label`：`从本镜移除「张三」`；失效则为 `移除失效资产`。
- × 与预览点击分区：点 × 不打开预览；`stopPropagation`，避免画布拖拽/选中节点。
- 不必二次确认：卸的是引用，素材还在画布上，可再检查关联或 Agent 绑回。
- 「未关联」空态不变，不出现删除按钮。

紧凑节点与全屏编辑都传入同一 `onRemove(nodeId)`。

## 6. 实现边界

### 前端（主路径）

1. `StoryboardAssetsCell` 增加可选 `onRemove?: (nodeId: string) => void`。有回调才渲染 ×（只读画布不传）。
2. `canvas-script-node.tsx` 紧凑表与全屏表：

   `onRemove={(nodeId) => onUpdateRow(row.id, { assetBindings: (row.assetBindings || []).filter((b) => b.nodeId !== nodeId) })}`

   全屏编辑走现有 `updateRow`。
3. 复用 `updateScriptRow`：已有 `if (patch.assetBindings)` 时调用 `replaceStoryboardAssetReferenceConnections`。不要另写一套清线逻辑。
4. `normalizeStoryboardAssetBindings` 继续去重、丢非法 role；过滤后空数组合法，显示「未关联」。

### 不要做的实现

- 不要只删 `connections` 而留下 `assetBindings`（线会按绑定画回来）。
- 不要在检查关联成功时 `replace` 整表绑定。
- 不要调用删除节点 API。

### Agent（只对齐，不作为人手入口）

`unbind_assets` 已存在。人手不依赖 Agent。若有余量，确认卸绑定后 `canvas_read_storyboard` 返回的该行 `assetBindings` 与标题一致即可，不必新 action。

## 7. 测试

- 单元格：有 `onRemove` 时芯片可卸；无回调时无 ×；点 × 不打开预览。
- `updateScriptRow`：某行去掉一个 `nodeId` 后，该行绑定少一条；其他行不变；对应 `storyboard-asset-reference` 消失；画布上该素材节点仍在。
- 失效 `nodeId`（nodes 里没有）仍能卸。
- 卸到空：该行「未关联」。
- 检查关联：已卸的名字若仍在文案里，再次检查可以挂回；不会在检查时自动清掉未出现在文案里的绑定。
- 生成参考：卸掉后 `storyboardRowReferenceNodeIds` 不再包含该节点。
- 回归：点芯片仍预览；绑定芯片仍显示节点标题。

不强制新的后端测；人手路径不经过 `unbind_assets`。若顺手补一条：`unbind_assets` 后该行绑定为空且行连线清除。

## 8. 验收

画布上有「张三」，第 1、2 镜都已关联：

1. 第 2 镜点 × → 第 2 镜变未关联（或只剩别的资产），第 1 镜仍是张三，左侧张三图还在。
2. 第 2 镜再点检查关联（文案含「张三在跳舞」）→ 可重新挂上。
3. 只读分享/只读画布没有 ×。

## 9. 发布

无 schema 变更，无需迁库。随下一小版（建议 v1.5.56）与现网同样的 compose 发布。广场 ingest、`handler/agent.go`、默认首页不在本版。

## 10. 落地顺序

1. 单元格 × + `onRemove`。
2. 接到 `onUpdateRow` / `updateRow`（连线同步已有）。
3. 测卸一条、卸失效、卸空、检查关联不误删。
4. 发版。
