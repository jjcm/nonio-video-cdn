package main

import (
	"fmt"
	"net/http"
	"os"
	"soci-video-cdn/config"
	"soci-video-cdn/route"
)

func setupRoutes(settings *config.Config) {
	http.Handle("/", http.FileServer(http.Dir("./files")))
	http.HandleFunc("/upload", route.UploadFile)
	http.HandleFunc("/move", route.MoveFile)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = settings.Port
		if port == "" {
			port = "4204"
		}
	}

	fmt.Printf("Listening on %v\n", port)
	http.ListenAndServe(":"+port, nil)
}

func main() {
	var settings config.Config
	// parse the config file
	if err := config.ParseJSONFile("./config.json", &settings); err != nil {
		panic(err)
	}

	fmt.Println("Starting video encoding server...")
	setupRoutes(&settings)
}
