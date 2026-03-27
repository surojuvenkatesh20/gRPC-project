package interceptors

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type rateLimiter struct {
	visitors  map[string]int
	limit     int
	resetTime time.Duration
	mu        sync.Mutex
}

func NewRateLimiter(limit int, resetTime time.Duration) *rateLimiter {
	rateLimiter := rateLimiter{
		limit:     limit,
		resetTime: resetTime,
		visitors:  make(map[string]int),
	}
	go rateLimiter.resetQuota()
	return &rateLimiter
}

func (rl *rateLimiter) resetQuota() {
	for {
		time.Sleep(rl.resetTime)
		rl.mu.Lock()
		rl.visitors = make(map[string]int)
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) RateLimiterInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	fmt.Println("Rate limiter middleware ran.")

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "Internal server error.")
	}

	visitorIp := p.Addr.String()
	rl.visitors[visitorIp]++
	fmt.Printf("+++++++++Visitor: %s, VisiterCount: %d\n", visitorIp, rl.visitors[visitorIp])
	if rl.visitors[visitorIp] > rl.limit {
		return nil, status.Error(codes.ResourceExhausted, "Rate limit exceeded.")
	}
	return handler(ctx, req)
}
