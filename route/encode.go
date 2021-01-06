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
	aspectRatio := y / x
	fmt.Println(nativeSize)
	fmt.Println(aspectRatio)

	/*
		if err = EncodeToFormat(ws, filename, "mp4", "h264", "2M", "300x200"); err != nil {
			return
		}
	*/

	// For bitrates, we use 2 * the number of pixels
	// I.e. 8k has 33 million pixels, so we use 66Mbps
	nativeBitrate := "2M"
	downscaleResolution := ""

	// Halfway between 720p and 480p
	// ( 1280 + 854 ) / 2 == 1600
	if nativeSize > 1067 {
		nativeBitrate = "1.8M"
		fmt.Println("need a 480p encode")
		downscaleResolution = fmt.Sprintf("854x%0.f", 854*aspectRatio)
		fmt.Println(downscaleResolution)
		if err = EncodeToFormat(ws, filename, "-480p", "0.8M", downscaleResolution); err != nil {
			return
		}
	}

	// Halfway between 1080p and 720p
	// ( 1920 + 1280 ) / 2 == 1600
	if nativeSize > 1600 {
		nativeBitrate = "4M"
		fmt.Println("need a 720p encode")
		downscaleResolution = fmt.Sprintf("1280x%0.f", 1280*aspectRatio)
		fmt.Println(downscaleResolution)
		if err = EncodeToFormat(ws, filename, "-720p", "1.8M", downscaleResolution); err != nil {
			return
		}
	}

	// Halfway between 1440p and 1080p
	// ( 2560 + 1920 ) / 2 == 2240
	if nativeSize > 2240 {
		nativeBitrate = "7.2M"
		fmt.Println("need a 1080p encode")
		downscaleResolution = fmt.Sprintf("1920x%0.f", 1920*aspectRatio)
		fmt.Println(downscaleResolution)
		if err = EncodeToFormat(ws, filename, "-1080p", "4M", downscaleResolution); err != nil {
			return
		}
	}

	// Halfway between 4k and 1440p
	// ( 3840 + 2560 ) / 2 == 3200
	if nativeSize > 3200 {
		nativeBitrate = "16.6M"
		fmt.Println("need a 1440p encode")
		downscaleResolution = fmt.Sprintf("2560x%0.f", 2560*aspectRatio)
		fmt.Println(downscaleResolution)
		if err = EncodeToFormat(ws, filename, "-1440p", "7.2M", downscaleResolution); err != nil {
			return
		}
	}

	// Seriously what are they uploading that we need to downscale to 4k?
	// Halfway between 8k and 4k
	// ( 7680 + 3840 ) / 2 == 5760
	if nativeSize > 5760 {
		nativeBitrate = "66M"
		fmt.Println("need a 4k encode")
		downscaleResolution = fmt.Sprintf("3840x%0.f", 3840*aspectRatio)
		fmt.Println(downscaleResolution)
		if err = EncodeToFormat(ws, filename, "-2160p", "16.6M", downscaleResolution); err != nil {
			return
		}
	}

	if err = EncodeToFormat(ws, filename, "", "16.6M", fmt.Sprintf("%0.fx%0.f", x, y)); err != nil {
		return
	}

	fmt.Println(nativeBitrate)
	fmt.Printf("Encoding finished for %v\n", filename)
}

// EncodeToFormat encodes a video source and sends back progress via the websocket
func EncodeToFormat(ws *websocket.Conn, filename string, suffix string, bitrate string, size string) error {
	// Set up our encoding options
	ffmpegConf := &ffmpeg.Config{
		FfmpegBinPath:   "/usr/local/bin/ffmpeg",
		FfprobeBinPath:  "/usr/local/bin/ffprobe",
		ProgressEnabled: true,
	}

	overwrite := true
	format := "mp4"
	codec := "h264"

	opts := ffmpeg.Options{
		OutputFormat: &format,
		Overwrite:    &overwrite,
		VideoCodec:   &codec,
		VideoBitRate: &bitrate,
		Resolution:   &size,
	}

	ws.WriteMessage(websocket.TextMessage, []byte("Starting h264 encode"))
	progress, err := ffmpeg.
		New(ffmpegConf).
		Input(fmt.Sprintf("files/temp-videos/%v", filename)).
		Output(fmt.Sprintf("files/videos/%v%v.mp4", strings.TrimSuffix(filename, filepath.Ext(filename)), suffix)).
		WithOptions(opts).
		Start(opts)

	fmt.Println("hmm")
	if err != nil {
		fmt.Println(err)
		fmt.Println("shit son")
		ws.WriteMessage(websocket.TextMessage, []byte("Error encoding webm"))
		return err
	}
	fmt.Println("k")

	for msg := range progress {
		fmt.Println(msg.GetProgress())
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%.6f", msg.GetProgress())))
	}

	return nil
}
