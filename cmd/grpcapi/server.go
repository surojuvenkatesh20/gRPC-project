package main

import (
	"grpcmongoproject"
	"grpcmongoproject/internals/api/handlers"
	"grpcmongoproject/internals/api/interceptors"
	"grpcmongoproject/pkg/utils"
	pb "grpcmongoproject/proto/gen"
	"log"
	"net"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func LoadEnvFileContents() {
	contents, err := grpcmongoproject.EnvFile.ReadFile(".env")
	if err != nil {
		log.Fatalln("Error in reading .env file contents: ", err)
	}
	tempFile, err := os.CreateTemp("", ".env")
	if err != nil {
		log.Fatalln("Error in creating env file: ", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(contents)
	if err != nil {
		log.Fatalln("Error in writing contents into temp file: ", err)
	}
	err = tempFile.Close()
	if err != nil {
		log.Fatalln("Error in closing temp file")
	}
	err = godotenv.Load(tempFile.Name())
	if err != nil {
		log.Fatalln("Error in loading temp .env file: ", err)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Error loading .env file: ", err)
	}

	// cert := os.Getenv("CERT_FILE")
	// key := os.Getenv("KEY_FILE")

	// creds, err := credentials.NewServerTLSFromFile(cert, key)
	// if err != nil {
	// 	log.Fatalln("Error in generating certificate credentials: ", err)
	// }

	//Without TLS
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors.ResponseTimeInterceptor, interceptors.AuthenticationInterceptor))

	//with TLS
	// r = interceptors.NewRateLimiter(5, time.Minute)
	// grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors.ResponseTimeInterceptor, interceptors.AuthenticationInterceptor), grpc.Creds(creds))
	// grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors.ResponseTimeInterceptor), grpc.Creds(creds))
	// grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(r.RateLimiterInterceptor, interceptors.ResponseTimeInterceptor, interceptors.AuthenticationInterceptor), grpc.Creds(creds))
	pb.RegisterExecsServiceServer(grpcServer, &handlers.Server{})
	pb.RegisterStudentsServiceServer(grpcServer, &handlers.Server{})
	pb.RegisterTeachersServiceServer(grpcServer, &handlers.Server{})

	go utils.JwtStore.DeleteExpiredTokensBG()
	// reflection.Register(grpcServer)

	port := os.Getenv("SERVER_PORT")
	log.Println("gRPC Server is listening to port: ", port)
	listen, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalln("Error creating a TCP Web Socket.")
	}
	err = grpcServer.Serve(listen)
	if err != nil {
		log.Fatalln("Error in Starting gRPC server: ", err)
	}
}
