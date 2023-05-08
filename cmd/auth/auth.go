package main

import (
	//"context"
	"flag"
	"fmt"
	"net/http"

	//"os"

	//"github.com/bloqs-sites/bloqsenjin/internal/auth"
	auth_http "github.com/bloqs-sites/bloqsenjin/pkg/auth/http"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/santhosh-tekuri/jsonschema/v5"
	//dbh "github.com/bloqs-sites/bloqsenjin/internal/db"
	//auth_server "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	//"github.com/bloqs-sites/bloqsenjin/proto"
	//"google.golang.org/grpc"
)

var (
	httpPort = flag.Int("HTTPPort", 8080, "The HTTP server port")
	//gRPCPort = flag.Int("gRPCPort", 50051, "The gRPC server port")

	//s *grpc.Server
)

func main() {
	flag.Parse()
	if err := conf.Compile(); err != nil {
		switch err := err.(type) {
		case jsonschema.InvalidJSONTypeError:
			panic(err)
		}
	}

	ch := make(chan error)

	// TODO: How can I make it that you can specify which implementation of the interfaces you want to use?
	//creds := dbh.NewMySQL(os.Getenv("DSN"))
	//secrets := dbh.NewKeyDB(dbh.NewRedisCreds("localhost", 6379, "", 0))

	//auther := auth.NewBloqsAuther(context.Background(), &creds)
	//tokener := auth.NewBloqsTokener(secrets)

	//go startGRPCServer(ch, auther, tokener)
	go startHTTPServer(ch)

	for i := range ch {
		if i != nil {
			panic(i)
		}
	}
}

//func startGRPCServer(ch chan error, a auth_server.Auther, t auth_server.Tokener) {
//	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *gRPCPort))
//	if err != nil {
//		ch <- err
//	}
//
//	s = grpc.NewServer()
//
//	as := auth_server.NewAuthServer(a, t)
//
//	proto.RegisterAuthServer(s, as)
//	log.Printf("server listening at %v", lis.Addr())
//	if err := s.Serve(lis); err != nil {
//		ch <- err
//	}
//}

//func createGRPCClient(ch chan error) (proto.AuthClient, func()) {
//	conn, err := grpc.Dial(net.JoinHostPort("localhost", fmt.Sprint(*gRPCPort)), grpc.WithTransportCredentials(insecure.NewCredentials()))
//	if err != nil {
//		ch <- err
//	}
//	return proto.NewAuthClient(conn), func() {
//		conn.Close()
//	}
//}

func startHTTPServer(ch chan error) {
	fmt.Printf("Auth HTTP server port:\t %d\n", *httpPort)
	ch <- http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), auth_http.Server())
}
