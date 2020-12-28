package main

import (
	"fmt"
	"net/http"
	"os"
	"soci-video-cdn/route"
)

func setupRoutes() {
	http.Handle("/", http.FileServer(http.Dir("./files")))
	http.HandleFunc("/upload", route.UploadFile)
	http.HandleFunc("/move", route.MoveFile)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "4204"
	}

	fmt.Printf("Listening on %v\n", port)
	http.ListenAndServe(":"+port, nil)
}

func main() {
	fmt.Println("Starting video encoding server...")
	setupRoutes()
}
