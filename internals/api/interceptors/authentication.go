package interceptors

import (
	"context"
	"fmt"
	"grpcmongoproject/pkg/utils"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthenticationInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	fmt.Println("Authentication middleware ran.")

	fmt.Println(info.FullMethod)
	skipMethods := make(map[string]bool)
	skipMethods["/main.ExecsService/ExecLogin"] = true
	skipMethods["/main.ExecsService/ExecForgotPassword"] = true
	skipMethods["/main.ExecsService/ExecResetPassword"] = true

	if skipMethods[info.FullMethod] {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "No metadata found.")
	}

	authHeader, ok := md["authorization"]
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "Authorization header not present")
	}

	tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	//Check if the token is already logged out
	isTokenLoggedOut := utils.JwtStore.IsLoggedOut(tokenString)
	if isTokenLoggedOut {
		return nil, status.Error(codes.Unauthenticated, "Invalid token.")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "Unauthorized access.")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		fmt.Println("Error in token parsing", err)
		return nil, status.Error(codes.Unauthenticated, "Unauthorized access.")
	}

	if !token.Valid {
		fmt.Println("invalid token")
		return nil, status.Error(codes.Unauthenticated, "Unauthorized access.")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		fmt.Println("Error in getting claims.")
		return nil, status.Error(codes.Unauthenticated, "Unauthorized access.")
	}

	role, ok := claims["role"].(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "Unauthorized access.")
	}

	userId := claims["uid"].(string)
	username := claims["uname"].(string)
	expiresAtF64 := claims["exp"].(float64)
	expiresAtI64 := int64(expiresAtF64)
	expiresAtStr := fmt.Sprintf("%d", expiresAtI64)

	newCtx := context.WithValue(ctx, "uid", userId)
	newCtx = context.WithValue(newCtx, "uname", username)
	newCtx = context.WithValue(newCtx, "role", role)
	newCtx = context.WithValue(newCtx, "exp", expiresAtStr)

	return handler(newCtx, req)
}
