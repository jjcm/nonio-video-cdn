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
	fmt.Printf("Encoding starting for %v\n", filename)
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", fmt.Sprintf("files/temp-videos/%v", filename)).Output()
	if err != nil {
		fmt.Println(err)
	}
	resolution := strings.Split(strings.TrimSpace(string(out)), "x")
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
	out, err = exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream_side_data=rotation", "-of", "default=nw=1:nk=1", "-i", fmt.Sprintf("files/temp-videos/%v", filename)).Output()
	if err != nil {
		fmt.Println("Error getting rotation from ffprobe")
		fmt.Println(err)
	}
	rotation, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		fmt.Println("Error parsing rotation")
		fmt.Println(err)
	}

	nativeSize := math.Max(x, y)
	aspectRatio := y / x

	// Check and see if there's metadata for rotation. If so and it's 90deg or 270deg, swap the x and y resolution.
	if (rotation-90)%180 == 0 {
		tmpX := x
		x = y
		y = tmpX
	}

	ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("resolution:%vx%v", x, y)))
	fmt.Printf("resolution:%vx%v\n", x, y)

	// For bitrates, we use 2 * the number of pixels
	// I.e. 8k has 33 million pixels, so we use 66Mbps
	nativeBitrate := "1M"
	var downscaleResolution string
	var time int64

	// Halfway between 720p and 480p
	// ( 1280 + 854 ) / 2 == 1600
	if nativeSize > 1067 {
		nativeBitrate = "1.8M"
		if x > y {
			downscaleResolution = fmt.Sprintf("854x%0.f", math.RoundToEven(854*aspectRatio/2)*2)
		} else {
			downscaleResolution = fmt.Sprintf("%0.fx854", math.RoundToEven(854*aspectRatio/2)*2)
		}
		if time, err = EncodeToFormat(ws, filename, "-480p", "0.8M", downscaleResolution); err != nil {
			return
		}
		fmt.Printf("480p finished: %vs\n", time)
	}

	// Halfway between 1080p and 720p
	// ( 1920 + 1280 ) / 2 == 1600
	if nativeSize > 1600 {
		nativeBitrate = "4M"
		if x > y {
			downscaleResolution = fmt.Sprintf("1280x%0.f", math.RoundToEven(1280*aspectRatio/2)*2)
		} else {
			downscaleResolution = fmt.Sprintf("%0.fx1280", math.RoundToEven(1280*aspectRatio/2)*2)
		}
		if time, err = EncodeToFormat(ws, filename, "-720p", "1.8M", downscaleResolution); err != nil {
			return
		}
		fmt.Printf("720p finished: %vs\n", time)
	}

	// Halfway between 1440p and 1080p
	// ( 2560 + 1920 ) / 2 == 2240
	if nativeSize > 2240 {
		nativeBitrate = "7.2M"
		if x > y {
			downscaleResolution = fmt.Sprintf("1920x%0.f", math.RoundToEven(1920*aspectRatio/2)*2)
		} else {
			downscaleResolution = fmt.Sprintf("%0.fx1920", math.RoundToEven(1920*aspectRatio/2)*2)
		}
		if time, err = EncodeToFormat(ws, filename, "-1080p", "4M", downscaleResolution); err != nil {
			return
		}
		fmt.Printf("1080p finished: %vs\n", time)
	}

	// Halfway between 4k and 1440p
	// ( 3840 + 2560 ) / 2 == 3200
	if nativeSize > 3200 {
		nativeBitrate = "16.6M"
		if x > y {
			downscaleResolution = fmt.Sprintf("2560x%0.f", math.RoundToEven(2560*aspectRatio/2)*2)
		} else {
			downscaleResolution = fmt.Sprintf("%0.fx2560", math.RoundToEven(2560*aspectRatio/2)*2)
		}
		if time, err = EncodeToFormat(ws, filename, "-1440p", "7.2M", downscaleResolution); err != nil {
			return
		}
		fmt.Printf("1440p finished: %vs\n", time)
	}

	// Seriously what are they uploading that we need to downscale to 4k?
	// Halfway between 8k and 4k
	// ( 7680 + 3840 ) / 2 == 5760
	if nativeSize > 5760 {
		nativeBitrate = "66M"
		if x > y {
			downscaleResolution = fmt.Sprintf("3840x%0.f", math.RoundToEven(3840*aspectRatio/2)*2)
		} else {
			downscaleResolution = fmt.Sprintf("%0.fx3840", math.RoundToEven(3840*aspectRatio/2)*2)
		}
		if time, err = EncodeToFormat(ws, filename, "-2160p", "16.6M", downscaleResolution); err != nil {
			return
		}
		fmt.Printf("4k finished: %vs\n", time)
	}

	if time, err = EncodeToFormat(ws, filename, "", nativeBitrate, fmt.Sprintf("%0.fx%0.f", x, y)); err != nil {
		return
	}
	fmt.Printf("Source finished: %vs\n", time)
	fmt.Printf("Encoding finished for %v\n", filename)
}

// EncodeToFormat encodes a video source and sends back progress via the websocket
func EncodeToFormat(ws *websocket.Conn, filename string, suffix string, bitrate string, size string) (int64, error) {
	now := time.Now()
	start := now.Unix()
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

	progress, err := ffmpeg.
		New(ffmpegConf).
		Input(fmt.Sprintf("files/temp-videos/%v", filename)).
		Output(fmt.Sprintf("files/videos/%v%v.mp4", strings.TrimSuffix(filename, filepath.Ext(filename)), suffix)).
		WithOptions(opts).
		Start(opts)

	if err != nil {
		// TODO: this doesn't seem to throw an error when things go wrong.
		// Easiest way to test: try and encode a h264 video at 25x25 resolution
		// It will fail since h264 doesn't allow resolutions that arent divisible by 2
		fmt.Println(err)
		ws.WriteMessage(websocket.TextMessage, []byte("Error"))
		return 0, err
	}

	if len(suffix) == 0 {
		suffix = "source"
	} else {
		suffix = strings.TrimPrefix(suffix, "-")
	}
	for msg := range progress {
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%v:%.1f", suffix, msg.GetProgress())))
	}
	ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%v:100", suffix)))

	now = time.Now()
	end := now.Unix()

	return end - start, nil
}
