package route

import (
	"fmt"
	"net/http"
	"os"
	"soci-video-cdn/util"
)

// MoveFile takes the temp file and renames it to match the url
func MoveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		util.SendResponse(w, "", 200)
		return
	}

	r.ParseMultipartForm(1 << 30)

	// Get the user's email if we're authorized
	bearerToken := r.Header.Get("Authorization")
	fmt.Println(bearerToken)
	user, err := util.GetUserEmail(bearerToken)
	fmt.Println(user)
	if err != nil {
		util.SendError(w, "User is not authorized.", 400)
		return
	}

	// Parse our url, and check if the url is available
	url := r.FormValue("url")
	urlIsAvailable, err := util.CheckIfURLIsAvailable(url)
	if err != nil {
		util.SendError(w, fmt.Sprintf("Error checking requested url: %v", url), 500)
		fmt.Println(err)
		return
	}
	if urlIsAvailable == false {
		util.SendError(w, fmt.Sprintf("Url \"%v\" is taken.", url), 400)
		return
	}

	// Check if the file we're moving exists
	tempFile := r.FormValue("oldUrl")
	if _, err := os.Stat(fmt.Sprintf("files/videos/%v.mp4", tempFile)); os.IsNotExist(err) {
		util.SendError(w, "No temp image exists with that name.", 400)
		fmt.Println(err)
		return
	}
	/*
		if _, err := os.Stat(fmt.Sprintf("files/thumbnails/%v.webp", tempFile)); os.IsNotExist(err) {
			util.SendError(w, "No temp thumbnail exists with that name.", 400)
			fmt.Println(err)
			return
		}
	*/

	// If everything else looks good, lets move the file.
	err = os.Rename(fmt.Sprintf("files/videos/%v.mp4", tempFile), fmt.Sprintf("files/images/%v.mp4", url))
	if err != nil {
		util.SendError(w, "Error renaming file.", 500)
		return
	}
	/*
		err = os.Rename(fmt.Sprintf("files/thumbnails/%v.webp", tempFile), fmt.Sprintf("files/thumbnails/%v.webp", url))
		if err != nil {
			util.SendError(w, "Error renaming thumbnail.", 500)
			return
		}
	*/
	if _, err := os.Stat("files/videos/%v-2160p.mp4"); err == nil {
		err = os.Rename(fmt.Sprintf("files/videos/%v-2160.mp4", tempFile), fmt.Sprintf("files/images/%v-2160.mp4", url))
		if err != nil {
			util.SendError(w, "Error renaming 2160p res file.", 500)
			return
		}
	}

	if _, err := os.Stat("files/videos/%v-1440p.mp4"); err == nil {
		err = os.Rename(fmt.Sprintf("files/videos/%v-1440.mp4", tempFile), fmt.Sprintf("files/images/%v-1440.mp4", url))
		if err != nil {
			util.SendError(w, "Error renaming 1440p res file.", 500)
			return
		}
	}

	if _, err := os.Stat("files/videos/%v-1080p.mp4"); err == nil {
		err = os.Rename(fmt.Sprintf("files/videos/%v-1080.mp4", tempFile), fmt.Sprintf("files/images/%v-1080.mp4", url))
		if err != nil {
			util.SendError(w, "Error renaming 1080p res file.", 500)
			return
		}
	}

	if _, err := os.Stat("files/videos/%v-720p.mp4"); err == nil {
		err = os.Rename(fmt.Sprintf("files/videos/%v-720.mp4", tempFile), fmt.Sprintf("files/images/%v-720.mp4", url))
		if err != nil {
			util.SendError(w, "Error renaming 720p res file.", 500)
			return
		}
	}

	if _, err := os.Stat("files/videos/%v-480p.mp4"); err == nil {
		err = os.Rename(fmt.Sprintf("files/videos/%v-480.mp4", tempFile), fmt.Sprintf("files/images/%v-480.mp4", url))
		if err != nil {
			util.SendError(w, "Error renaming 480p res file.", 500)
			return
		}
	}

	// Send back a response.
	util.SendResponse(w, url, 200)
}
