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
)

var (
	port = flag.Int("port", 50051, "The server port")
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
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
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
