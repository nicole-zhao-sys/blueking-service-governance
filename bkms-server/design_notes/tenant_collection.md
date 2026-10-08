# Collection 租户隔离

业务集合的读写统一从 `ctx` 注入 `tenant_id`，避免每个 Store 手写过滤。本轮只接入 `workspaces`，文档只描述当前这份实现已经做到的行为，以及还没补上的边界。

## 要点

- 租户只认 `ctx`，不认请求参数、不认调用方传入的 `tenant_id`（防伪造，注入时覆盖）。
- HTTP 入口由 `tenant.Required` 写入 ctx：单租统一为 `default`，多租校验 header 与用户归属。
- 异步任务由 taskq envelope 携带 `_tenantId`，worker 恢复到 ctx。
- 缺租户直接失败（`ErrTenantIDRequired`），不静默回落，避免漏中间件变成错数据。
- 字段名统一 `tenant_id`（`tenant.FieldTenantID`）。本轮不改 `workspaces.id` 全局唯一索引。

## 当前实现

```
HTTP / taskq / CLI
    → ctx 带 tenant_id
    → Store 调 dbtenant.Wrap(coll)
    → filter / 写入文档注入 tenant_id
    → mongo
```

| 路径 | 行为 |
|---|---|
| Find / Delete / Count | 只改 filter |
| Insert / Replace | 文档写入当前租户 |
| Update | 只改 filter，不改 update payload |
| Aggregate | 管道首部加 `$match: {tenant_id}` |

接入方式：`NewXxxStore` 里 `dbtenant.Wrap(coll)`，Store 方法签名不变。

存量 `workspaces` 由迁移 `000018` 回填 `tenant_id=default`，并建普通索引。

当前接入点只有 `WorkspaceStoreMongo`。因此本轮实际生效的是：

- workspace 的 `List / ListWithPagination / Get / Create / Update / Delete / CountByState`
- taskq 恢复出来、HTTP 中间件写入、命令行显式写入的 `ctx tenant`

还没有把所有业务集合都切到这套包装层。

## 为什么放 Collection，而不是 Store 小函数

- 规则只有一份：漏写 filter 不会静默跨租户读。
- Store 仍是普通 `Find(ctx, filter)`，不把租户做成查询参数。
- 单租 / 多租差异停在入口中间件，数据层始终「ctx 必有 tenant」。

不把 Wrapper 做成完整 `mongo.Collection` 替代物；只包 Store 实际用到的方法。当前 `workspaces` 的平台管理查询也走同一个 Store，因此跨租户边界要单独说明。

## 跨租户怎么处理

分两种情况看：两类当前都已经有实现，但做法不一样。

**1. 表本身没有 `tenant_id`**（全平台一份）  
这个现在已经有实现：继续用 `platformTables`。集合名加进去，`Wrap` 后 `isPlatform` 为 true，读写都不注入。现有三个：`cluster_addon_defs`、`depservice_services`、`plat_admin_role_bindings`。`workspaces` 不要加进去。

**2. 表有 `tenant_id`，但这次调用要看全部租户**  
这个当前也已经实现，但实现方式不是改 ctx，而是走同一个 Store 的跨租户视图：

- `WorkspaceStore` 暴露 `CrossTenant()`，返回同一底层集合的跨租户访问视图
- `WorkspaceStoreMongo.CrossTenant()` 通过 `dbtenant.CrossTenantValue(s.collection, newWorkspaceStoreMongo)` 构造跨租户 store
- `Collection` 内部用 `scopeCrossTenant` 标记这个视图；只有默认 `scopeTenant` 才会注入 `tenant_id`
- platmgt workspace / platmgt workspace admin 已经显式使用 `h.registry.WorkspaceStore.CrossTenant()`

也就是说，当前 `workspaces` 已经支持“同一张表，普通业务按租户隔离，平台管理走跨租户视图”。

平台管理侧现在的写法类似：

```go
func (h *GinHandler) service() *platmgtworkspace.Service {
	return platmgtworkspace.NewService(
		h.registry.WorkspaceStore.CrossTenant(),
		h.registry.AppStore,
		h.registry.EnvStore,
	)
}
```

如果后续有命令行或其他平台侧调用也需要跨租户访问 `workspaces`，同样应复用 `WorkspaceStore.CrossTenant()`，而不是给 `List/Get` 增加 `SkipTenant` 之类的参数。
这样做的好处是：业务侧和平台侧都还是普通的 `store.List/Get/CountByState`，差异只体现在拿到的是默认 store 还是 `CrossTenant()` store。

## 当前边界

- 当前只有 `workspaces` 接入了 Wrapper，其它集合还没有统一切换。
- 平台共享表已经支持按集合名跳过租户注入。
- `workspaces` 已经支持 `CrossTenant()` 视图；但这套模式目前还没有推广到其它租户表。
