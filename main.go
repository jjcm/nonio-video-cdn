package main

import (
	"fmt"
	"net/http"
	"os"
	"soci-video-cdn/route"
)

func setupRoutes() {
	http.HandleFunc("/upload", route.UploadFile)
	http.HandleFunc("/move", route.MoveFile)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "4204"
	}

	http.ListenAndServe(":"+port, nil)
}

func main() {
	fmt.Println("Starting media encoding server")
	setupRoutes()
}
