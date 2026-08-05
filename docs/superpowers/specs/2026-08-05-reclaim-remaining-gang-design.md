# Reclaim 剩余 Gang 需求设计

## 目标

Reclaim 只为 Job 尚未达到 `MinAvailable` 的部分回收资源，避免重复计算已经成功调度的 replica。

## 规则

`remainingMin = max(0, MinAvailable - ScheduledReplicas)`，需求量为 `remainingMin * Request`。

如果 `remainingMin == 0`，该 Job 不需要因为 Gang 门槛进行 reclaim。

## 非目标

- 不改变 `Replicas`、`BatchSize` 或普通调度逻辑。
- 不改变 victim 排序、队列资格和回滚语义。

## 验收

- 已调度部分 replica 的 Job 只回收剩余缺口。
- 已达到 `MinAvailable` 的 Job 不会额外驱逐 victim。
- 全量测试通过。
