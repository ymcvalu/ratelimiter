package ratelimiter

import (
	"github.com/go-redis/redis"
)

var ratelimitScript = redis.NewScript(`redis.replicate_commands()
local ratelimit_info = redis.pcall('HMGET',KEYS[1],'last_time','current_token')
local last_time = ratelimit_info[1]
local current_token = tonumber(ratelimit_info[2])
local now = redis.call('time')
local now_ms = now[1]*1000 + now[2]/1000
local max_token = tonumber(ARGV[1])
local token_rate = tonumber(ARGV[2])
local token_rate_ms = token_rate/1000
if current_token == nil then
  current_token = max_token
  last_time = now_ms
end
if current_token == 0 then 
    local pass = now_ms - last_time
    local to_add = math.floor(pass * token_rate_ms)
    current_token = current_token + to_add
    last_time = to_add / token_rate_ms + last_time
    if current_token > max_token then 
        current_token = max_token
    end
end
local result = 0
if current_token > 0 then 
    current_token = current_token - 1
    result = 1
end
redis.call('HMSET',KEYS[1],'last_time',last_time,'current_token',current_token)
redis.call('pexpire', KEYS[1], 1000)
return result
`)

type Scripter interface {
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(hashes ...string) *redis.BoolSliceCmd
	ScriptLoad(script string) *redis.StringCmd
}

type Ratelimiter struct {
	r    Scripter
	key  string
	cap  int64 // bucket capacity
	rate int64 // token generated rate, per second
}

func New(r Scripter, key string, cap, rate int64) *Ratelimiter {
	return &Ratelimiter{
		r:    r,
		key:  key,
		cap:  cap,
		rate: rate,
	}
}

func (r *Ratelimiter) IsAllow() (bool, error) {
	resp := ratelimitScript.Run(r.r, []string{r.key}, r.cap, r.rate)
	return resp.Bool()
}