package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	rest "github.com/bloqs-sites/bloqsenjin/pkg/rest/http"
)

var (
	//addr     = flag.String("addr", "localhost:50051", "the address to connect to")
	httpPort = flag.Int("HTTPPort", 8080, "The HTTP server port")
)

func main() {
	flag.Parse()
	if err := conf.Compile(); err != nil {
		switch err := err.(type) {
		case jsonschema.InvalidJSONTypeError:
			panic(err)
		}
	}

	fmt.Printf("Auth HTTP server port:\t %d\n", *httpPort)
	http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), rest.Server(context.Background(), "/"))
	//conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	//if err != nil {
	//	log.Fatalf("did not connect: %v", err)
	//}
	//defer conn.Close()
	//c := pb.NewAuthClient(conn)

	//dbh, err := db.NewMySQL(context.Background(), "owduser:passwd@/owd")

	//if err != nil {
	//	panic(err)
	//}

	//s := rest.NewServer(":8089", dbh, c)

	//file, err := os.Open("./cmd/rest/preferences")

	//if err != nil {
	//	panic(err)
	//}

	//defer file.Close()

	//scanner := bufio.NewScanner(file)
	//for scanner.Scan() {
	//	line := scanner.Text()
	//	parts := strings.Split(line, "=")

	//	if len(parts) == 1 {
	//		parts = append(parts, "")
	//	}

	//	go dbh.Insert(context.Background(), "preference", []map[string]string{
	//		{
	//			"name":        parts[0],
	//			"description": parts[1],
	//		},
	//	})
	//}

	//s.AttachHandler(context.Background(), "preference", new(models.PreferenceHandler))
	//s.AttachHandler(context.Background(), "bloq", new(models.BloqHandler))

	//err = s.Run()
	//if errors.Is(err, http.ErrServerClosed) {

	//} else if err != nil {
	//	os.Exit(1)
	//}
}
