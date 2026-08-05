# Session Event Controller 设计

## 目标

在内存中维护跨 Session 的输入快照，让 Job、Node 和 RunningTask 变化可以在下一次 Session 生效。

## 设计

- `SessionController` 持有 NodeSpec、Job、Queue 和 RunningTask 快照。
- `Apply(Event)` 更新快照：AddJob、RemoveJob、AddNode、RemoveNode、UpdateRunningTasks。
- `Run()` 从当前 NodeSpec 创建新的 Scheduler，再调度当前 Job/Queue/victim 快照。
- 事件只更新输入，不直接执行调度。

## 非目标

- 不接数据库或 Kubernetes watch。
- 不在事件 Apply 时自动触发调度。
- 不改变单次 Scheduler 的调度算法。

## 验收

- AddJob 后下一次 Run 能看到新 Job。
- RemoveNode 后下一次 Run 不使用该 Node。
- 重复 Add 或删除不存在对象返回错误。
- UpdateRunningTasks 在下一次 Run 中生效。
