package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/bloqs-sites/bloqsenjin/internal/auth"

	pb "github.com/bloqs-sites/bloqsenjin/proto"
	"google.golang.org/grpc"
	//"github.com/redis/go-redis/v9"
)

var (
	httpPort = flag.Int("HTTPPort", 8080, "The HTTP server port")
	gRPCPort = flag.Int("gRPCPort", 50051, "The gRPC server port")

	auther = new(auth.Auther)
)

type server struct {
	pb.UnimplementedAuthServer
}

func (s *server) Validate(ctx context.Context, in *pb.Token) (*pb.Validation, error) {
	return &pb.Validation{
		Valid: auther.VerifyToken(string(in.GetJwt()), uint(*in.Permissions)),
	}, nil
}

func main() {
	flag.Parse()

    startgRPCServer()

	//rdb := redis.NewClient(&redis.Options{
	//	Addr:     "localhost:6379",
	//	Password: "",
	//	DB:       0,
	//})

	//err = rdb.Set(context.Background(), "mykey", "myvalue", 0).Err()

	//if err != nil {
	//    panic(err);
	//}
}

func startgRPCServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *gRPCPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterAuthServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
