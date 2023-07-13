package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	http_helpers "github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
	"github.com/google/uuid"
)

var supported_formats = []string{".webp", ".avif"}

const (
	magick      = "convert"
	uploads_dir = "BLOQS_IMAGE_UPLOADS_DIRECTORY"
)

func uploadsDir() (string, error) {
	path, exists := os.LookupEnv(uploads_dir)
	if !exists {
		return "", fmt.Errorf("env var `%s` not specified", uploads_dir)
	}

	path = strings.TrimSpace(path)

	return filepath.Abs(path)
}

func Save(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	path, err := uploadsDir()
	if err != nil {
		return "", err
	}

	dir, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := dir.Stat()
	if err != nil {
		return "", err
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("env var `%s` does not specify a directory", uploads_dir)
	}

	ext := filepath.Ext(header.Filename)

	tempfile, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("*%s", ext))
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err = io.Copy(tempfile, file); err != nil {
		return "", nil
	}

	tmp := fmt.Sprintf("%s/%s%s", os.TempDir(), rndName(), ext)
	cmd := exec.CommandContext(ctx, magick, tempfile.Name(),
		"-gravity", "center",
		//"-extent", "\"%[fx:h<w?h:w]x%[fx:h<w?h:w]\"",
		"-extent", "1:1",
		tmp,
	)

	var str strings.Builder
	buf, err := cmd.CombinedOutput()
	str.Write(buf)
	if err != nil {
		str.WriteString(err.Error())
		return "", errors.New(str.String())
	}

	name, err := createName(path)
	if err != nil {
		return "", err
	}

	fn := func(ctx context.Context, ext string, ch chan error, wg *sync.WaitGroup) {
		defer wg.Done()
		path := fmt.Sprintf("%s/%s%s", path, name, ext)
		cmd = exec.CommandContext(ctx, magick, tmp, "-resize", "512x512", path)
		var str strings.Builder
		buf, err := cmd.CombinedOutput()
		str.Write(buf)
		if err != nil {
			str.WriteString(err.Error())
			ch <- errors.New(str.String())
		}
	}

	var (
		ch = make(chan error)
		wg = &sync.WaitGroup{}
	)
	wg.Add(len(supported_formats) + 1)
	go fn(ctx, ext, ch, wg)
	for _, i := range supported_formats {
		go fn(ctx, i, ch, wg)
	}

	wg.Wait()
    close(ch)

	println(7)
	err = nil
	for err := range ch {
		if err != nil {
			os.Remove(fmt.Sprintf("%s/%s%s", path, name, ext))
			for _, i := range supported_formats {
				os.Remove(fmt.Sprintf("%s/%s%s", path, name, i))
			}
			return "", err
		}
	}

	println(8, name)
	return name, nil
}

func createName(dir string) (string, error) {
	name := rndName()
	for {
		found := false
		files, err := os.ReadDir(dir)
		if err != nil {
			return "", err
		}

		for _, f := range files {
			if strings.HasPrefix(f.Name(), name) {
				found = true
				break
			}
		}

		if !found {
			break
		}

		name = rndName()
	}

	return name, nil
}

func rndName() string {
	return uuid.NewString()
}

func Server(endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string

		h := w.Header()
		status, err := helpers.CheckOriginHeader(&h, r, false)

		switch r.Method {
		case "":
			fallthrough
		case http.MethodGet:
			if err != nil {
				break
			}

			path := r.URL.Path[len("/"):]

			segs := strings.Split(path, "/")

			if l := len(segs); l != 1 {
				http.NotFound(w, r)
				return
			}

			path, err = uploadsDir()
			if err != nil {
				break
			}
			path = fmt.Sprintf("%s/%s", path, segs[0])
			var file *os.File
			file, err = os.Open(path)
			if err != nil {
				break
			}
			defer file.Close()

			contentType := "application/octet-stream"
			if ext := filepath.Ext(path); ext != "" {
				contentType = mime.TypeByExtension(ext)
			}

			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(path)))
			w.Header().Set("Content-Type", contentType)

			_, err = io.Copy(w, file)
			if err != nil {
				break
			}
		case http.MethodOptions:
			http_helpers.Append(&h, "Access-Control-Allow-Methods", http.MethodGet)
			http_helpers.Append(&h, "Access-Control-Allow-Methods", http.MethodOptions)
			h.Set("Access-Control-Max-Age", fmt.Sprint(time.Hour*24/time.Second))
		default:
			status = http.StatusMethodNotAllowed
		}

		if err != nil {
			msg = err.Error()
		}

		w.WriteHeader(int(status))
		w.Write([]byte(msg))
		w.Header().Add("Content-Type", http_helpers.PLAIN)
	}
}

func Serve(endpoint string, w http.ResponseWriter, r *http.Request) {
	Server(endpoint)(w, r)
}
