package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	rest "github.com/bloqs-sites/bloqsenjin/pkg/rest/http"
	"github.com/santhosh-tekuri/jsonschema/v5"

	_ "github.com/joho/godotenv/autoload"
)

func REST(w http.ResponseWriter, r *http.Request) {
	if err := conf.Compile(); err != nil {
		var err_msg strings.Builder
		switch err := err.(type) {
		case *jsonschema.SchemaError:
			err_msg.WriteString(fmt.Sprintf("failed to compile schema `%s`.\n", err.SchemaURL))
			switch err := err.Err.(type) {
			case *jsonschema.ValidationError:
				err_msg.WriteString(fmt.Sprintf("schema is not valid: %s.\n", err.DetailedOutput().Error))
			default:
				err_msg.WriteString("unknown error.\n")
			}
		case *jsonschema.ValidationError:
			err_msg.WriteString(fmt.Sprintf("schema is not valid: %s.\n", err.Error()))
		case *jsonschema.InfiniteLoopError:
			err_msg.WriteString(fmt.Sprintf("schema compilation/validation found infinite loop at `%s`.\n", err.Error()))
		case *jsonschema.InvalidJSONTypeError:
			err_msg.WriteString(fmt.Sprintf("received invalid JSON: %s.\n", err.Error()))
		case *os.PathError:
			err_msg.WriteString(fmt.Sprintf("error in operation `%s` on file `%s`: %s.\n.\n", err.Op, err.Path, err.Error()))
		default:
			err_msg.WriteString(fmt.Sprintf("unexpected error: %s.\n", err.Error()))
		}

		w.Write([]byte(err_msg.String()))
		w.Header().Add("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	rest.Serve("/api/rest/", w, r)
}
