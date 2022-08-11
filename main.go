package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	mux                = sync.Mutex{}
	postRequestCounter = map[string]uint{}
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3030"
	}
	err := os.MkdirAll("./content", 0755)
	fileServer := http.FileServer(http.Dir("./content"))
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			fileServer.ServeHTTP(resp, req)
			return
		case "POST", "PUT":
			reqPath := strings.TrimPrefix(req.URL.Path, "/")
			counter := func() uint {
				mux.Lock()
				defer mux.Unlock()
				// Should work since the default value is 1
				postRequestCounter[reqPath] += 1
				return postRequestCounter[reqPath]
			}()
			// Always fail the first request to trigger a retry
			if counter == 1 {
				fmt.Println("Failing first POST to ", reqPath)
				resp.WriteHeader(500)
				return
			}

			filePath := "./content/" + reqPath
			fmt.Println("Writing ", filePath)
			err := os.MkdirAll(filepath.Dir(filePath), 0755)
			if err != nil {
				panic(err)
			}
			file, err := os.Create(filePath)
			if err != nil {
				panic(err)
			}

			written, err := io.Copy(file, req.Body)
			if err != nil {
				fmt.Println("Error after writing ", written, " bytes to ", filePath, " : ", err)
				resp.WriteHeader(500)
				return
			}
			fmt.Println("Wrote ", written, " bytes to ", filePath)
			resp.WriteHeader(200)
			return
		}
		resp.WriteHeader(404)
	})
	err = http.ListenAndServe(":"+port, nil)
	if err != http.ErrServerClosed {
		panic(err)
	}
}
