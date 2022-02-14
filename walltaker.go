package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
    "os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/guregu/null"
	"github.com/kardianos/osext"
	"github.com/pelletier/go-toml"
	"github.com/reujab/wallpaper"
)

type WalltakerData struct {
	ID               int         `json:"id"`
	Expires          time.Time   `json:"expires"`
	UserID           int         `json:"user_id"`
	Terms            string      `json:"terms"`
	Blacklist        string      `json:"blacklist"`
	PostURL          null.String `json:"post_url"`
	PostThumbnailURL interface{} `json:"post_thumbnail_url"`
	PostDescription  interface{} `json:"post_description"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	URL              string      `json:"url"`
}

func getWalltakerData(url string) WalltakerData {
	webClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "Walltaker Go Client/1.0.2")

	res, getErr := webClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	userData := WalltakerData{}
	jsonErr := json.Unmarshal(body, &userData)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	return userData
}

type NoDataError struct {
	IntA int
	IntB int
	Msg  string
}

func (e *NoDataError) Error() string {
	return e.Msg
}

func getWallpaperUrlFromData(userData WalltakerData) (string, error) {
	if userData.PostURL.String == "" {
		return "", &NoDataError{
			Msg: fmt.Sprintf("No data found for ID %d", userData.ID),
		}
	}
	return userData.PostURL.String, nil
}

func main() {
	// fmt.Println("WALLTAKER CLIENT")
	fmt.Println(`
	██╗    ██╗ █████╗ ██╗     ██╗  ████████╗ █████╗ ██╗  ██╗███████╗██████╗ 
	██║    ██║██╔══██╗██║     ██║  ╚══██╔══╝██╔══██╗██║ ██╔╝██╔════╝██╔══██╗
	██║ █╗ ██║███████║██║     ██║     ██║   ███████║█████╔╝ █████╗  ██████╔╝
	██║███╗██║██╔══██║██║     ██║     ██║   ██╔══██║██╔═██╗ ██╔══╝  ██╔══██╗
	╚███╔███╔╝██║  ██║███████╗███████╗██║   ██║  ██║██║  ██╗███████╗██║  ██║
	 ╚══╝╚══╝ ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
																			
	 	v1.0.2. Go client by @OddPawsX
	 		 	Walltaker by Gray over at joi.how <3

	(You can minimize this window; it will periodically check in for new wallpapers)
	`)

	folderPath, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Loaded config from " + filepath.Join(folderPath, "walltaker.toml"))

	dat, err := os.ReadFile(filepath.Join(folderPath, "walltaker.toml"))
	if err != nil {
		log.Fatal(err)
	}

    bg, err := wallpaper.Get()
    fmt.Println("Detected original wallpaper as: ", bg)

    c := make(chan os.Signal)
    signal.Notify(c, os.Interrupt)
    go func() {
        <-c
        wallpaper.SetFromFile(bg)
        os.Exit(0)
    }()

	tomlDat := string(dat)

	config, _ := toml.Load(tomlDat)

	base := config.Get("Base.base").(string)
	feed := config.Get("Feed.feed").(int64)
	freq := config.Get("Preferences.interval").(int64)
	mode := config.Get("Preferences.mode").(string)

	builtUrl := base + strconv.FormatInt(feed, 10) + ".json"
	fmt.Printf("Checking in every %d seconds...\r\n", freq)

	userData := getWalltakerData(builtUrl)

	wallpaperUrl, noDataErr := getWallpaperUrlFromData(userData)
	ready := noDataErr == nil
	for ready == false {
		if noDataErr != nil {
			// log.Fatal(noDataErr)
			fmt.Printf("No data for ID %d, trying again in %d seconds...\r\n", feed, freq)
			time.Sleep(time.Second * time.Duration(freq))
			userData = getWalltakerData(builtUrl)
			wallpaperUrl, noDataErr = getWallpaperUrlFromData(userData)
		} else {
			ready = true
		}
	}

	err = wallpaper.SetFromFile("") // free up for macOS
	err = wallpaper.SetFromURL(wallpaperUrl)
	fmt.Println("Set initial wallpaper: DONE")

	if strings.ToLower(mode) == "fit" {
		err = wallpaper.SetMode(wallpaper.Fit)
	} else if strings.ToLower(mode) == "crop" {
		err = wallpaper.SetMode(wallpaper.Crop)
	} else {
		err = wallpaper.SetMode(wallpaper.Crop)
	}

	oldWallpaperUrl := wallpaperUrl

	for range time.Tick(time.Second * time.Duration(freq)) {
		fmt.Printf("Polling... ")
		userData := getWalltakerData(builtUrl)
		wallpaperUrl := userData.PostURL.String
		if wallpaperUrl != oldWallpaperUrl {
			fmt.Printf("New wallpaper found! Setting...")
			err = wallpaper.SetFromFile("") // free up for macOS
			err = wallpaper.SetFromURL(wallpaperUrl)
			fmt.Printf("Set!")
			oldWallpaperUrl = wallpaperUrl
		} else {
			fmt.Printf("Nothing new yet.")
		}
		fmt.Printf("\r\n")
	}
}
