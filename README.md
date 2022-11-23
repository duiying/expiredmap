# ExpiredMap

Golang 实现的有过期策略且线程安全的 map（支持泛型）。  

适用场景：内存级缓存，减少 Redis 的 I/O 次数，减轻 Redis 的连接数压力。  

Key 的删除策略：  

- 主动删除（定时器）
- 惰性删除（访问 Key 时检查过期时间）

使用：  

```go
// 生成一个 key 是 int64 类型、value 是 *cacheItem 类型的 map
// 最大容量为 10，每隔 5s 定时检查 Key 的过期时间（根据 Key 的数量适当调整定时检查时间，防止给 CPU 造成压力）
em := NewExpiredMap[int64, *cacheItem](10, time.Second*5)
```






