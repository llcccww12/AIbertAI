# 目标模式与 DSH 适配修复清单

> 状态：已确认，按顺序实施。
>
> 日期：2026-11-26
>
> 范围：员工端桌面（`AIbertAI/native`、`AIbertAI/code/app`、`AIbertAI/code/web`）。服务端管理台不在本期；服务端只复用现有回执接口 `/sessions/{id}/model-invocations/{invocation}/receipt`。

## 1. 背景

员工端跑目标时出现四类症状，经查都落在“模型调用没有结算”与“目标状态没有和 DSH 对齐”两件事上：

- 会话直接冻结：目标暂停后仍占着 `GoalRun`，用户消息被拒，队列永远等不到唤醒。
- 目标在后台继续，前台却弹红色“本轮执行未完成”，用户以为任务失败了。
- 模型自己开的 DSH 目标在会话窗口没有任何显示，删除也没有反应。
- 提示词里要求的“补记”在管理台没有入口，实际只能自动补。

## 2. 修复项

| # | 问题 | 根因 | 改法 | 验证 |
| --- | --- | --- | --- | --- |
| 1 | 老版本把调用推给上游却没记投递，记录停在 `dispatched` | 只有上游关连接才记 `delivered` | `recoverCommittedDelivery`：SDK 报 `committed` 且账本 `succeeded` 时补记；补测试 | 新增用例在修复前失败，`go test ./` 全绿 |
| 2 | 结算说不清时无限期阻塞 | `Uncertain` 与 `hasPendingDSHInvocation` 永久挡调度 | 3 次对账或 60 秒后按 0 释放，记录标 `uncertain` 并留档；账本可达时自动补记实际用量；会话留一行可见记录 | 模拟账本不可达，目标在 60 秒内恢复为可继续 |
| 3 | 用户动作被结算挡住 | 同一闸门同时管用户消息和自动续跑 | 用户动作（发消息、暂停、删除、切模型）不等结算；只有自动续跑等对账 | 待结算状态下发消息、暂停、删除都能执行 |
| 4 | 会话冻结 | `acpturns.go` 只在非暂停分支清 `GoalRun`；`reserveTurn` 只看 `GoalRun != ""` 就拒绝 | 后台轮结束且目标不再 `active/reconciling/clearing` 时清 `GoalRun`；`reserveTurn` 只在目标确实活跃时拒绝 | 后台轮失败后能立刻发消息 |
| 5 | 后台继续却报红 | 目标轮失败后 `acpTurnStatePayload` 直接回传技术错误码，前端按码取红文案 | 目标仍 `is_current`（可恢复）时 `lastError` 回传前端已有的中性码 `session_turn_failed`；技术码保留在 turn 上供恢复使用 | 目标存活时不再出现红色“执行出错” |
| 6 | 模型自建的 DSH 目标不显示 | `dshgoalevents.go` 要求本地已有目标才接管；普通轮的 `goal/change` 事件被丢弃 | `adoptDSHGoalEvent` 把模型自建目标镜像成 `dsh_observed` 记录，随 `_monkeyai/session/state` 推送；该记录不拿 `GoalRun`，所有“目标是否占用会话”的判断改走 `platformGoal()` | 模型自建目标后窗口出现状态条，且仍能正常发消息 |
| 7 | 删除目标没反应 | 窗口侧的 DSH 目标动作一律返回 `dsh_goal_control_unavailable`，前端静默 | 镜像目标允许 `clear`（只清记录，不动 DSH），其余动作仍拒绝 | 删除后状态条消失；暂停仍拒绝 |
| 8 | 其余适配问题 | `goal_session_busy` 静默、轮次/预算展示不一致 | 逐个回显并统一字段 | 对应测试 |
| 9 | 看不出目标活着 | 状态条不读 `rounds_started`（前端 0 处引用） | 前端源码已丢失，直接在 `code/web/assets/index-zDHqw1Zn.js` 的目标状态条补“第 N 轮”，并补 `goal.rounds` / `goal.roundsCount` 中英文案与 `.composer-goal-rounds` 样式 | 目标推进中状态条显示当前轮次 |
| 10 | 完成/暂停不提醒 | 前端判定完成用 `status === "complete"`，原生一直发 `"completed"`，完成分支永远不成立，通知不触发 | `goalView` 出口把 `completed` 归一为 `complete`，前端既有的完成通知与未读逻辑随之生效 | 目标完成后收到一次完成通知 |

## 3. 结算降级的记账口径

| 情形 | 处置 |
| --- | --- |
| 账本证明确实成功 | 照实记账 |
| 账本证明没派发/没扣费 | 记 0，标 `not_charged` |
| 账本也说不清 | 先按 0 释放，标 `uncertain` 并留档；账本可达时自动补记实际值 |

释放时把该轮标为“结果未知”，让模型先核对工作区再继续（复用 `dshPendingWrite` 检查），避免重复执行工具动作。

## 4. 实施顺序

4 → 3 → 1 → 2 → 5 → 6 → 7 → 8 → 9 → 10。每项单独提交，`cd AIbertAI/native && go test ./` 必须全绿。

## 5. 不做的部分

- 管理台的未结算调用列表与人工补记入口（需新增 admin 接口 + 页面）。
- 自动级联重跑、目标模板等产品能力。
