# Reclaim Priority Gate Plan

**目标：** 只把可回收资源交给比 victim 更高 priority 的等待 Job。

1. 为 Job 加入 priority 字段。
2. 写 priority 10 等待 Job 不能驱逐 priority 100 victim 的失败测试。
3. 在 RunWithReclaim victim 过滤中要求 `waiting.Priority > victim.Priority`。
4. 加入正向测试，运行全量测试，提交并合并。
