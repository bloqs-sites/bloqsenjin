package main

import (
	//"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	//"os"

	//"github.com/bloqs-sites/bloqsenjin/internal/auth"
	auth_http "github.com/bloqs-sites/bloqsenjin/pkg/auth/http"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/santhosh-tekuri/jsonschema/v5"

	//dbh "github.com/bloqs-sites/bloqsenjin/internal/db"
	auth_server "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	httpPort = flag.Int("HTTPPort", 8080, "The HTTP server port")
	gRPCPort = flag.Int("gRPCPort", 50051, "The gRPC server port")

	s *grpc.Server
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

	for {
		select {
		case err := <-ch:
			if err != nil {
				panic(err)
			}
		}
	}
}

func startGRPCServer(ch chan error, a auth_server.Auther, t auth_server.Tokener) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *gRPCPort))
	if err != nil {
		ch <- err
	}

	s = grpc.NewServer()

	as := auth_server.NewAuthServer(a, t)

	proto.RegisterAuthServer(s, &as)
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		ch <- err
	}
}

func createGRPCClient(ch chan error) (proto.AuthClient, func()) {
	conn, err := grpc.Dial(net.JoinHostPort("localhost", fmt.Sprint(*gRPCPort)), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		ch <- err
	}
	return proto.NewAuthClient(conn), func() {
		conn.Close()
	}
}

func startHTTPServer(ch chan error) {
	//	r.Route(log_in_route, func(w http.ResponseWriter, r *http.Request) {
	//		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
	//			s.ServeHTTP(w, r)
	//			return
	//		}
	//
	//		var err error
	//
	//		if err = r.ParseMultipartForm(64 << 20); err != nil {
	//			return
	//		}
	//
	//		var v *proto.Validation
	//
	//		c, cc := createGRPCClient(ch)
	//		defer cc()
	//
	//		permissionsStr := r.FormValue("permissions")
	//		if permissionsStr == "" {
	//			permissionsStr = strconv.Itoa(int(auth.DEFAULT_PERMISSIONS))
	//		}
	//		permissions, err := strconv.ParseUint(permissionsStr, 10, 0)
	//
	//		if err != nil {
	//			return
	//		}
	//
	//		var t *proto.Token
	//
	//		switch r.URL.Query().Get(query) {
	//		case "basic":
	//			t, err = c.LogIn(r.Context(), &pb.CredentialsWantPermissions{
	//				Credentials: &pb.Credentials{
	//					Credentials: &proto.Credentials_Basic{
	//						Basic: &pb.Credentials_BasicCredentials{
	//							Email:    r.Form["email"][0],
	//							Password: r.Form["pass"][0],
	//						},
	//					},
	//				},
	//				Permissions: permissions,
	//			})
	//		}
	//		if err != nil {
	//			w.Write([]byte(err.Error()))
	//			w.WriteHeader(500)
	//			return
	//		}
	//
	//		if t == nil {
	//			w.WriteHeader(400)
	//			return
	//		}
	//
	//		w.Header().Add("BLOQS_JWT", string(t.Jwt))
	//		w.Header().Add("Content-Type", "application/json")
	//
	//		if err := json.NewEncoder(w).Encode(t); err != nil {
	//			w.WriteHeader(500)
	//		}
	//
	//		if err != nil {
	//			w.Write([]byte(err.Error()))
	//			w.WriteHeader(500)
	//			return
	//		}
	//
	//		if v == nil {
	//			w.WriteHeader(400)
	//			return
	//		}
	//
	//		if v.Message != nil {
	//			w.Write([]byte(*v.Message))
	//		}
	//
	//		if v.Valid {
	//			w.WriteHeader(200)
	//		} else {
	//			w.WriteHeader(400)
	//		}
	//	})
	//

	fmt.Printf("Auth HTTP server port:\t %d\n", *httpPort)
	ch <- http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), auth_http.Server())
}
