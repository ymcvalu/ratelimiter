# ratelimiter
a distributed ratelimiting implemention base on redis, using token bucket algorithm

usage:
```go
r := New(redisClient, "ratelimit:test", capacity, rate)
// try to take a token
if r.IsAllow(){
    // --snip--
}
```