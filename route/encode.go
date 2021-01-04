package route

import (
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"path/filepath"
	"soci-video-cdn/util"
	"strconv"
	"strings"
	"time"

	ffmpeg "github.com/floostack/transcoder/ffmpeg"
	"github.com/gorilla/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

// Encode accepts a filename as a param. File must be the temp file returned by upload.go
func Encode(w http.ResponseWriter, r *http.Request) {
	// Allow connections from anywhere
	if r.Method == "OPTIONS" {
		util.SendResponse(w, "", 200)
		return
	}
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	// Upgrade the connection to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		if _, ok := err.(websocket.HandshakeError); !ok {
			fmt.Println(err)
		}
		return
	}
	defer ws.Close()

	// Get the value of the width and height of the video, then store whichever is largest
	filename := r.URL.Query()["file"][0]
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", fmt.Sprintf("files/temp-videos/%v", filename)).Output()
	if err != nil {
		fmt.Println(err)
	}
	resolution := strings.Split(strings.TrimSuffix(string(out), "\n"), "x")
	if len(resolution) < 1 {
		fmt.Println("Cannot detect resolution")
	}
	x, err := strconv.ParseFloat(resolution[0], 64)
	if err != nil {
		fmt.Println("Error detecting x resolution")
		fmt.Println(err)
	}
	y, err := strconv.ParseFloat(resolution[1], 64)
	if err != nil {
		fmt.Println("Error detecting y resolution")
		fmt.Println(err)
	}
	nativeSize := math.Max(x, y)

	/*
		if err = EncodeToFormat(ws, filename, "webm", "libvpx-vp9", "2M"); err != nil {
			return
		}
	*/

	if err = EncodeToFormat(ws, filename, "mp4", "h264", "2M"); err != nil {
		return
	}

	if nativeSize > 3840 {
		fmt.Println("need a 4k encode")
	}

	if nativeSize > 2560 {
		fmt.Println("need a 1440p encode")
	}

	if nativeSize > 1920 {
		fmt.Println("need a 1080p encode")
	}

	if nativeSize > 1280 {
		fmt.Println("need a 720p encode")
	}

	if nativeSize > 854 {
		fmt.Println("need a 480p encode")
	}

	fmt.Printf("Encoding finished for %v\n", filename)
}

// EncodeToFormat encodes a video source and sends back progress via the websocket
func EncodeToFormat(ws *websocket.Conn, filename string, format string, codec string, bitrate string) error {
	// Set up our encoding options
	ffmpegConf := &ffmpeg.Config{
		FfmpegBinPath:   "/usr/local/bin/ffmpeg",
		FfprobeBinPath:  "/usr/local/bin/ffprobe",
		ProgressEnabled: true,
	}

	overwrite := true

	opts := ffmpeg.Options{
		OutputFormat: &format,
		Overwrite:    &overwrite,
		VideoCodec:   &codec,
		VideoBitRate: &bitrate,
	}

	ws.WriteMessage(websocket.TextMessage, []byte("Starting webm encode"))
	progress, err := ffmpeg.
		New(ffmpegConf).
		Input(fmt.Sprintf("files/temp-videos/%v", filename)).
		Output(fmt.Sprintf("files/videos/%v.webm", strings.TrimSuffix(filename, filepath.Ext(filename)))).
		WithOptions(opts).
		Start(opts)

	if err != nil {
		fmt.Println(err)
		ws.WriteMessage(websocket.TextMessage, []byte("Error encoding webm"))
		return err
	}

	for msg := range progress {
		fmt.Println(msg.GetProgress())
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%.6f", msg.GetProgress())))
	}

	return nil
}
