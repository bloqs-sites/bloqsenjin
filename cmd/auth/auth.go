package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/proto"

	pb "github.com/bloqs-sites/bloqsenjin/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	//"github.com/redis/go-redis/v9"
)

var (
	httpPort = flag.Int("HTTPPort", 8080, "The HTTP server port")
	gRPCPort = flag.Int("gRPCPort", 50051, "The gRPC server port")

	auther = new(auth.Auther)

	s *grpc.Server
)

type server struct {
	pb.UnimplementedAuthServer
}

func (s *server) SignIn(ctx context.Context, in *pb.Credentials) (*pb.Validation, error) {
	switch x := in.Creds.(type) {
	case *proto.Credentials_Basic:
		if err := auther.SignInBasic(x); err != nil {
			msg := err.Error()
			return &pb.Validation{
				Valid:   false,
				Message: &msg,
			}, err
		}
	case nil:
		msg := ""
		return &pb.Validation{
			Valid:   false,
			Message: &msg,
		}, fmt.Errorf("")
	default:
		msg := ""
		return &pb.Validation{
			Valid:   false,
			Message: &msg,
		}, fmt.Errorf("Profile.Avatar has unexpected type %T", x)
	}

	return &pb.Validation{
		Valid: true,
	}, nil
}

func (s *server) SignOut(ctx context.Context, in *pb.Credentials) (*pb.Validation, error) {
	return &pb.Validation{
		Valid: true,
	}, nil
}

func (s *server) LogIn(ctx context.Context, in *pb.Credentials) (*pb.Token, error) {
	var x uint64 = 4
	return &pb.Token{
		Jwt:         []byte(""),
		Permissions: &x,
	}, nil
}

func (s *server) LogOut(ctx context.Context, in *pb.Credentials) (*pb.Validation, error) {
	return &pb.Validation{
		Valid: true,
	}, nil
}

func (s *server) Validate(ctx context.Context, in *pb.Token) (*pb.Validation, error) {
	return &pb.Validation{
		Valid: auther.VerifyToken(string(in.GetJwt()), uint(*in.Permissions)),
	}, nil
}

func main() {
	flag.Parse()

    ch := make(chan error)

	go startGRPCServer(ch)
    go startHTTPServer(ch)

    for {
        select {
        case err := <- ch:
                if err != nil {
                    panic(err);
                }
        }
    }

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

func startGRPCServer(ch chan error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *gRPCPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s = grpc.NewServer()
	pb.RegisterAuthServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		ch <- err
	}
}

func createGRPCClient() pb.AuthClient {
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", *gRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	return pb.NewAuthClient(conn)
}

func startHTTPServer(ch chan error) {
	route := conf.MustGetConfOrDefault("/", "auth", "signInPath")
	query := conf.MustGetConfOrDefault("type", "auth", "signInTypeQueryParam")

	fmt.Printf("Auth path:\t %s\n", route)
	fmt.Printf("Auth type query parameter:\t %s\n", query)

	c := createGRPCClient()

	r := mux.NewRouter()
	r.Route(route, func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			s.ServeHTTP(w, r)
		}

		var err error

		if err = r.ParseForm(); err != nil {
			return
		}

		var v *pb.Validation

		switch r.URL.Query().Get(query) {
		case "basic":
			v, err = c.SignIn(context.Background(), &pb.Credentials{
				Creds: &proto.Credentials_Basic{
					Basic: &pb.BasicCredentials{
						Email:    r.Form["email"][0],
						Password: r.Form["pass"][0],
					},
				},
			})
		}

		if err != nil {
			http.NotFound(w, r)
		}

		if v != nil {
			http.NotFound(w, r)
		}

		w.Write([]byte(*v.Message))

		if v.Valid {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(400)
		}
	})

	fmt.Printf("Auth HTTP server port:\t %d\n", *httpPort)
	ch <- http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), r)
}
