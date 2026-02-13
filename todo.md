# TODO

## Frontend

- [ ] Disabled proxy 在前端不显示的问题
  - 当前行为：`enabled: false` 的代理在 `pkg/config/load.go` 中被过滤，不会加载到 proxy manager，前端无法看到
  - 需要考虑：是否应该在前端显示 disabled 的代理（以灰色或其他方式标识），并允许用户启用/禁用

- [ ] Store proxy 删除后前端列表没有及时刷新
  - 原因：`RemoveProxy` 通过 `notifyChangeUnlocked()` 异步通知变更，前端立即调用 `fetchData()` 时 proxy manager 可能还没处理完
  - 可能的解决方案：
    1. 后端删除 API 等待 proxy manager 更新完成后再返回
    2. 前端乐观更新，先从列表移除再后台刷新
    3. 前端适当延迟后再刷新（不优雅）
