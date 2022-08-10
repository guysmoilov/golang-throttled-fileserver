package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
			filePath := "./content/" + strings.TrimPrefix(req.URL.Path, "/")
			fmt.Println("Writing ", filePath)
			err := os.MkdirAll(filepath.Dir(filePath), 0755)
			if err != nil {
				panic(err)
			}
			file, err := os.Create(filePath)
			if err != nil {
				panic(err)
			}

			// The key part - limit writing to 30KB per second
			throttleWriter := NewThrottleWriter(file, 30000, 1<<12)

			written, err := io.Copy(throttleWriter, req.Body)
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
