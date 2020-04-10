package ratelimiter

import (
	"github.com/go-redis/redis"
	"runtime"
	"sync"
	"testing"
	"time"
)

var redisClient *redis.Client

func TestMain(m *testing.M) {
	redisClient = redis.NewClient(&redis.Options{
		DB:       0,
		Password: "",
		Network:  "tcp",
		Addr:     "127.0.0.1:6379",
	})
	m.Run()
}

func TestIsAllow(t *testing.T) {
	const (
		bucketCapacity = 50
		tokenRate      = 100
		N              = 100
	)

	allowCount := 0
	r := New(redisClient, "ratelimit:test", bucketCapacity, tokenRate)
	now := time.Now()
	for i := 0; i < N; i++ {
		allow, err := r.IsAllow()
		if err != nil {
			t.Errorf("error raised when trying to take tokens: %s", err.Error())
			return
		}
		if allow {
			allowCount++
		}
		t.Logf("#%04d %v", i+1, allow)
	}

	delta := time.Now().Sub(now)
	t.Logf("total: %d  allow: %d  disallow: %d  cost time: %v", N, allowCount, N-allowCount, delta)

	ms := delta.Milliseconds()
	t.Logf("the estimation allow count is %d", bucketCapacity+ms*100/1000)
}

func TestMultiTask(t *testing.T) {
	const (
		bucketCapacity = 200
		tokenRate      = 200
		N              = 100
	)
	r := New(redisClient, "ratelimit:test", bucketCapacity, tokenRate)
	taskNum := runtime.GOMAXPROCS(0)
	allows := make([]int, taskNum)
	now := time.Now()
	wg := sync.WaitGroup{}
	for i := 0; i < taskNum; i++ {
		me := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < N; i++ {
				allow, err := r.IsAllow()
				if err != nil {
					t.Errorf("error raised: %s", err.Error())
					return
				}
				if allow {
					allows[me]++
				}
			}
		}()
	}
	wg.Wait()
	timeCost := time.Now().Sub(now).Milliseconds()
	genNum := timeCost * tokenRate / 1000
	totalAllow := 0
	for i := 0; i < taskNum; i++ {
		totalAllow += allows[i]
	}

	t.Logf("tokens per task taken: %v", allows)
	t.Logf("total: %d  allow: %d  disallow: %d  estimation: %d  time cost: %vms",
		taskNum*N,
		totalAllow,
		taskNum*N-totalAllow,
		bucketCapacity+genNum,
		timeCost,
	)
}

func BenchmarkIsAllow(b *testing.B) {
	b.Run("100", func(b *testing.B) {
		r := New(redisClient, "ratelimit:test", 100, 100)
		for i := 0; i < b.N; i++ {
			_, err := r.IsAllow()
			if err != nil {
				b.Errorf("error raised: %s", err.Error())
				return
			}
		}
	})

	b.Run("500", func(b *testing.B) {
		r := New(redisClient, "ratelimit:test", 500, 500)
		for i := 0; i < b.N; i++ {
			_, err := r.IsAllow()
			if err != nil {
				b.Errorf("error raised: %s", err.Error())
				return
			}
		}
	})

	b.Run("1000", func(b *testing.B) {
		r := New(redisClient, "ratelimit:test", 1000, 1000)
		for i := 0; i < b.N; i++ {
			_, err := r.IsAllow()
			if err != nil {
				b.Errorf("error raised: %s", err.Error())
				return
			}
		}
	})
}