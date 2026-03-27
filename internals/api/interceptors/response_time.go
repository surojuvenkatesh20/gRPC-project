package interceptors

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func ResponseTimeInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	fmt.Println("Response time interceptor ran.")

	start := time.Now()
	response, err := handler(ctx, req)
	reqDuration := time.Since(start)

	st, _ := status.FromError(err)
	fmt.Printf("Method %s, status: %d, duration: %v", info.FullMethod, st.Code(), reqDuration)

	md := metadata.Pairs("X-Response-Time", reqDuration.String())
	grpc.SetHeader(ctx, md)
	fmt.Println("Response time middleware exists")
	return response, err
}
