package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/auth"
	dbh "github.com/bloqs-sites/bloqsenjin/internal/db"
	auth_server "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/proto"

	pb "github.com/bloqs-sites/bloqsenjin/proto"
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

	ch := make(chan error)

    // TODO: This needs to be credentials and at the same time have support for
    // the various db.KVDBer
	creds := dbh.NewKeyDB(dbh.NewRedisCreds("localhost", 6379, "", 0))
	secrets := dbh.NewKeyDB(dbh.NewRedisCreds("localhost", 6379, "", 0))

	go startGRPCServer(ch, creds, secrets)
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

func startGRPCServer(ch chan error, creds, secrets db.KVDBer) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *gRPCPort))
	if err != nil {
		ch <- err
	}

	s = grpc.NewServer()

	auther := auth.NewAuther(creds, secrets)

	pb.RegisterAuthServer(s, auth_server.NewAuthServer(auther, auther))
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		ch <- err
	}
}

func createGRPCClient(ch chan error) (pb.AuthClient, func()) {
	conn, err := grpc.Dial(net.JoinHostPort("localhost", fmt.Sprint(*gRPCPort)), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		ch <- err
	}
	return pb.NewAuthClient(conn), func() {
		conn.Close()
	}
}

func startHTTPServer(ch chan error) {
	sign_in_route := conf.MustGetConfOrDefault("/sign-in", "auth", "signInPath")
	sign_out_route := conf.MustGetConfOrDefault("/sign-out", "auth", "signOutPath")
	log_in_route := conf.MustGetConfOrDefault("/log-in", "auth", "logInPath")
	log_out_route := conf.MustGetConfOrDefault("/log-out", "auth", "logOutPath")

	r := mux.NewRouter()
	r.Route(sign_in_route, validationRoute(ch, func(t string, c pb.AuthClient, r *http.Request) (v *pb.Validation, err error) {
		switch t {
		case "basic":
			v, err = c.SignIn(r.Context(), &pb.Credentials{
				Creds: &proto.Credentials_Basic{
					Basic: &pb.Credentials_BasicCredentials{
						Email:    r.Form["email"][0],
						Password: r.Form["pass"][0],
					},
				},
			})
		}
		return
	}))

	r.Route(sign_out_route, validationRoute(ch, func(t string, c pb.AuthClient, r *http.Request) (v *pb.Validation, err error) {
		switch t {
		case "basic":
			v, err = c.SignOut(r.Context(), &pb.Credentials{
				Creds: &proto.Credentials_Basic{
					Basic: &pb.Credentials_BasicCredentials{
						Email:    r.Form["email"][0],
						Password: r.Form["pass"][0],
					},
				},
			})
		}
		return
	}))

	r.Route(log_in_route, tokenRoute(ch, func(t string, c pb.AuthClient, r *http.Request) (tk *pb.Token, err error) {
        permissionsStr := r.FormValue("permissions")
        if permissionsStr == "" {
            permissionsStr = strconv.Itoa(int(auth.DEFAULT_PERMISSIONS))
        }
        permissions, err := strconv.ParseUint(permissionsStr, 10, 0)

        if err != nil {
            return
        }

		switch t {
		case "basic":
			tk, err = c.LogIn(r.Context(), &pb.CredentialsWantPermissions{
				Credentials: &pb.Credentials{
					Creds: &proto.Credentials_Basic{
						Basic: &pb.Credentials_BasicCredentials{
							Email:    r.Form["email"][0],
							Password: r.Form["pass"][0],
						},
					},
				},
                Permissions:  permissions,
			})
		}

		return
	}))

	r.Route(log_out_route, validationRoute(ch, func(t string, c pb.AuthClient, r *http.Request) (v *pb.Validation, err error) {
		return
	}))

	fmt.Printf("Auth HTTP server port:\t %d\n", *httpPort)
	ch <- http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), r)
}

func validationRoute(ch chan error, match func(string, pb.AuthClient, *http.Request) (*pb.Validation, error)) func(http.ResponseWriter, *http.Request) {
	query := conf.MustGetConfOrDefault("type", "auth", "signInTypeQueryParam")

	return func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			s.ServeHTTP(w, r)
			return
		}

		var err error

		if err = r.ParseMultipartForm(64 << 20); err != nil {
			return
		}

		var v *pb.Validation

		c, cc := createGRPCClient(ch)
		defer cc()

		v, err = match(r.URL.Query().Get(query), c, r)

		if err != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(500)
			return
		}

		if v == nil {
			w.WriteHeader(400)
			return
		}

		if v.Message != nil {
			w.Write([]byte(*v.Message))
		}

		if v.Valid {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(400)
		}
	}
}

func tokenRoute(ch chan error, match func(string, pb.AuthClient, *http.Request) (*pb.Token, error)) func(http.ResponseWriter, *http.Request) {
	query := conf.MustGetConfOrDefault("type", "auth", "signInTypeQueryParam")

	return func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			s.ServeHTTP(w, r)
			return
		}

		var err error

		if err = r.ParseMultipartForm(64 << 20); err != nil {
			return
		}

		var t *pb.Token

		c, cc := createGRPCClient(ch)
		defer cc()

		t, err = match(r.URL.Query().Get(query), c, r)

		if err != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(500)
			return
		}

		if t == nil {
			w.WriteHeader(400)
			return
		}

		w.Header().Add("BLOQS_JWT", string(t.Jwt))
		w.Header().Add("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(t); err != nil {
			w.WriteHeader(500)
		}
	}
}
