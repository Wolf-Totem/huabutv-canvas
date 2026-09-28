# 画布连线：框选、多选删除、双击删除

| 字段 | 内容 |
| --- | --- |
| 日期 | 2026-09-28 |
| 状态 | 待开发（产品口径已定，本文冻结范围） |
| 产品 | 画布TV |
| 现网 | v1.5.59，schema 45 |
| 前置 | 单击连线可选中一条；Delete/Backspace 删这一条；右键菜单可删；左键框选只命中节点 |

---

## 1. 问题

用户希望连线与节点一样能被左键框选，选中后 Delete 批量删掉；双击一条连线也能删。

现状缺口：

- 选中态是单个 `selectedConnectionId`，不能多选。
- 框选手势只查节点 AABB（`canvas-selection.ts` 空间索引），完全不测连线。
- Delete 优先删节点；没有节点时才删那一条连线。
- `ConnectionPath` 有透明 16px 描边可点选、可右键，没有 `onDoubleClick`。
- Leafer 层只负责画线，`pointerEvents` 不接交互。

## 2. 目标

- 空白处按住左键拖框：框到的**节点和连线**都进入选中（与现有节点框选同一套手势）。
- 选中后按 Delete / Backspace：删掉选中的节点和选中的连线。
- 双击一条连线：直接删除这一条（不经过确认）。
- 单击连线仍只选这一条，并清掉节点选中（保持现有互斥）。
- 只读画布不能删。

## 3. 非目标

- 不改连线几何算法、不重做 Leafer 命中。
- 不做连线拖拽改锚点、不合并/拆分连线。
- 不改「隐藏未选择连线」的显示规则；框选命中以**当前可见连线**为准。
- 不自动改分镜 `assetBindings`（见风险）；不迁库；不改 `handler/agent.go`。

## 4. 交互合同

### 4.1 框选

沿用现有 `resolveCanvasPointerIntent`：桌面左键在背景拖 = 框选；触控/中键/空格仍是平移。

命中规则与节点一致：

| 拖动方向 | `hitMode` | 连线 |
| --- | --- | --- |
| 左 → 右 | `contain` | 整条折线/曲线的包围盒完全在框内 |
| 右 → 左 | `intersect` | 线段与框相交（或包围盒相交，见实现取舍） |

修饰键与节点共用 `replace / add / toggle / subtract`。

框里同时有节点和连线时：**两边都选中**。预览高亮两者。

隐藏未选择连线打开时，框选只测 `visibleDisplayConnections`，避免点到看不见的线。

### 4.2 键盘删除

```
Delete / Backspace：
  删除 selectedNodeIds 对应节点（现有级联：节点删掉则端点连线一并去掉）
  再删除仍存在的 selectedConnectionIds
```

节点已删时，连在它上面的线不必再单独删。只选线、不选节点时，只删线，两端节点保留。

### 4.3 双击删除

在命中描边（现有 16px `pointerEvents: stroke`）上 `dblclick`：删这一条，清空该条选中。单击仍只选中。双击间隔走浏览器默认，不另做计时器。

误触风险：接受。Esc 取消选中不变。

### 4.4 与单击/右键的关系

- 单击：`selectedConnectionIds = {id}`，清空节点选中。
- 右键：选中该条后打开现有连接菜单。
- Shift+单击（可选，第二期）：在连线多选里加减，不在本期必做；本期框选已能多选。

## 5. 数据与状态

把 `selectedConnectionId: string | null` 升级为 `selectedConnectionIds: Set<string>`。

兼容点：

- 强调高亮：`selectedConnectionIds.has(id)` 替代 `===`。
- 键盘、右键删除、隐藏未选连线的「当前选中」都改读这个 Set。
- `deleteConnection(id)` 保留；新增 `deleteConnections(ids: Set<string>)` 一次 `commitConnections`，避免 N 次历史记录。
- 框选 `replace` 时清空「另一类」选中：若本次框只中线、不中节点，节点选中清空；只中节点则连线选中清空；都中则都保留。单击节点仍清连线（现有 `setSelectedConnectionId(null)` 改成 `setSelectedConnectionIds(new Set())`）。

历史：走现有 `commitConnections` / `commitNodes`，可撤销。

## 6. 实现边界

### 6.1 框选命中（核心）

不要把每条贝塞尔拿去像素扫描。建议两级：

1. **粗测**：连线控制点/端点 AABB 与选框 `contain` / `intersect`（与节点同一套 `canvasSelectionHitsBounds`）。
2. **精测（intersect）**：对 `canvasConnectionPath` 采样折线（例如 12～24 段）做线段-矩形相交；`contain` 可只要求全部采样点在框内，或要求 AABB contain（更严、实现更简单）。

第一期推荐：

- `contain`：连线 AABB 完全在框内（可能漏掉「框住弯曲中段但端点在框外」的线，可接受）。
- `intersect`：AABB 相交即可（可能多选擦边线，可用采样收紧）。

几何函数做成纯函数，放 `canvas-selection.ts` 或新建 `canvas-connection-selection.ts`，框选预览每帧调用，必须廉价。

采样输入用当前 `displayConnections` 的 from/to 节点（含分镜滚动偏移），与绘制路径一致。

### 6.2 框选控制器

`use-canvas-selection-controller.ts` 的 `updateSelectionPreview`：

- 现有节点 `hitNodeIds` 保留。
- 增加 `hitConnectionIds`。
- `applyCanvasSelectionStrategy` 对节点、连线各算一份。
- 预览：节点继续 `applyCanvasNodeSelectionPreview`；连线用 CSS/`data-connection-id` 或 Leafer 选中态（Leafer 已有 `selectedConnectionId`，改为 Set 后按 id 加粗）。

手势 `initialSelection` 扩展为 `{ nodes, connections }`，Cancel/Esc 能还原。

### 6.3 双击

`ConnectionPath` 增加 `onDoubleClick`。`canvas-project-world-layers` 传到 `deleteConnection`。`stopPropagation`，避免触发画布双击（若有）。只读层不传回调。

### 6.4 键盘

`use-canvas-keyboard.ts`：有节点或连线选中时 preventDefault，避免浏览器后退；先 `deleteNodes` 再 `deleteConnections` 剩余 id。

### 6.5 分镜素材连线（风险）

`relation === "storyboard-asset-reference"` 的线由 `assetBindings` 同步回来。只删线、不改绑定，下次改分镜会把线画回来。

第一期：**允许删**，与普通线相同；文档/changelog 不单独承诺「卸绑定」。若验收发现立刻弹回，再在 `deleteConnections` 里对这类 relation 同步清 `assetBindings`（复用格子卸绑定那套）。

## 7. 测试

纯函数：

- AABB contain / intersect 命中与否。
- 采样折线与矩形相交（横线穿过框、框在弧外不相交）。
- `applyCanvasSelectionStrategy` 分别作用于节点 id 与连线 id。

键盘/删除：

- 只选两条线，Delete 后两条消失、节点还在。
- 框选节点+线，Delete 后节点及所有相关线都没了。
- 双击一条线，该线消失。

回归：

- 单击线仍只选一条；框选节点行为与现在一致（含 Shift/Alt）。
- 隐藏未选择连线时，框选选不中已隐藏的线。
- 撤销能恢复批量删线。
- 只读画布双击/Delete 无效。

## 8. 验收

画布上多节点、多条连线（含交叉）：

1. 左→右框住两条完整连线 → 两条高亮；Delete → 两条消失，节点还在。
2. 右→左框擦过一条线 → 该线入选。
3. 框同时罩住一个节点和一条不相干的线 → 两者都选中；Delete 删该节点（及其端点线）以及那条不相干的线。
4. 双击一条线 → 立刻删除。
5. 撤销两次能回到框选删除前。

## 9. 发布

无 schema。随下一小版发布。不改广场 ingest、`agent.go`、默认首页。

## 10. 落地顺序

1. `selectedConnectionIds` + `deleteConnections`，键盘走 Set。
2. 连线几何命中纯函数 + 框选接入（可见连线）。
3. 双击删除。
4. 高亮/Leafer/隐藏未选连线读 Set。
5. 测框选、批量删、双击、撤销、只读。
6. 发版。
