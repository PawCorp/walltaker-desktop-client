package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"walltaker/icon"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/guregu/null"
	"github.com/hugolgst/rich-go/client"
	"github.com/juju/fslock"
	"github.com/kardianos/osext"
	"github.com/martinlindhe/inputbox"
	"github.com/pelletier/go-toml"
	"github.com/pkg/browser"
	"github.com/reujab/wallpaper"
)

var VERSION string = "v2.0.2"

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
	SetBy            null.String `json:"set_by"`
	URL              string      `json:"url"`
}

type E621PostsData struct {
	Posts []struct {
		ID        int    `json:"id"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		File      struct {
			Width  int    `json:"width"`
			Height int    `json:"height"`
			Ext    string `json:"ext"`
			Size   int    `json:"size"`
			Md5    string `json:"md5"`
			URL    string `json:"url"`
		} `json:"file"`
		Preview struct {
			Width  int    `json:"width"`
			Height int    `json:"height"`
			URL    string `json:"url"`
		} `json:"preview"`
		Sample struct {
			Has        bool   `json:"has"`
			Height     int    `json:"height"`
			Width      int    `json:"width"`
			URL        string `json:"url"`
			Alternates struct {
			} `json:"alternates"`
		} `json:"sample"`
		Score struct {
			Up    int `json:"up"`
			Down  int `json:"down"`
			Total int `json:"total"`
		} `json:"score"`
		Tags struct {
			General   []string      `json:"general"`
			Species   []string      `json:"species"`
			Character []string      `json:"character"`
			Copyright []string      `json:"copyright"`
			Artist    []string      `json:"artist"`
			Invalid   []interface{} `json:"invalid"`
			Lore      []interface{} `json:"lore"`
			Meta      []string      `json:"meta"`
		} `json:"tags"`
		LockedTags []interface{} `json:"locked_tags"`
		ChangeSeq  int           `json:"change_seq"`
		Flags      struct {
			Pending      bool `json:"pending"`
			Flagged      bool `json:"flagged"`
			NoteLocked   bool `json:"note_locked"`
			StatusLocked bool `json:"status_locked"`
			RatingLocked bool `json:"rating_locked"`
			Deleted      bool `json:"deleted"`
		} `json:"flags"`
		Rating        string        `json:"rating"`
		FavCount      int           `json:"fav_count"`
		Sources       []string      `json:"sources"`
		Pools         []interface{} `json:"pools"`
		Relationships struct {
			ParentID          interface{}   `json:"parent_id"`
			HasChildren       bool          `json:"has_children"`
			HasActiveChildren bool          `json:"has_active_children"`
			Children          []interface{} `json:"children"`
		} `json:"relationships"`
		ApproverID   int         `json:"approver_id"`
		UploaderID   int         `json:"uploader_id"`
		Description  string      `json:"description"`
		CommentCount int         `json:"comment_count"`
		IsFavorited  bool        `json:"is_favorited"`
		HasNotes     bool        `json:"has_notes"`
		Duration     interface{} `json:"duration"`
	} `json:"posts"`
}

func getWalltakerData(url string) WalltakerData {
	webClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "Walltaker Go Client/"+VERSION+"-"+runtime.GOOS)

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

func clearWindowsWallpaperCache() {
	// Remove cached wallpaper files, issue #12
	if runtime.GOOS == "windows" {
		windowsWallpaperCacheDir := os.Getenv("APPDATA") + "\\Microsoft\\Windows\\Themes"
		if _, err := os.Stat(windowsWallpaperCacheDir + "\\TranscodedWallpaper"); !os.IsNotExist(err) {
			e := os.Remove(windowsWallpaperCacheDir + "\\TranscodedWallpaper")
			if e != nil {
				log.Fatal(e)
			}
		}
		if _, err2 := os.Stat(windowsWallpaperCacheDir + "\\CachedFiles"); !os.IsNotExist(err2) {
			e2 := os.RemoveAll(windowsWallpaperCacheDir + "\\CachedFiles")
			if e2 != nil {
				log.Fatal(e2)
			}
		}
	}
}

func goSetWallpaper(url string, saveLocally bool, setterName string, setAt string, notify bool) {
	clearWindowsWallpaperCache()
	if runtime.GOOS != "windows" {
		file, err := downloadImageForMac(url)
		if err != nil {
			log.Println("Ouch! Had a problem while downloading your wallpaper. This is a mac specific bug!")
			log.Println("Full error: ", err)
		}
		wallpaper.SetFromFile(file)
		defer cleanUpCacheForMac(file) // OK to delete after setting wallpaper, MacOS shows bg w/o file remaining there
	} else {
		err := wallpaper.SetFromURL(url)

		if err != nil {
			log.Println("Ouch! Had a problem while setting your wallpaper.")
			log.Println("Full error: ", err)
		}
	}

	if notify {
		notifyStr := ""
		if setterName == "" {
			notifyStr = "Someone changed your wallpaper~"
		} else {
			notifyStr = fmt.Sprintf("%s changed your wallpaper~", setterName)
		}
		errNotify := beeep.Notify("Walltaker", notifyStr, "")
		if errNotify != nil {
			panic(errNotify)
		}
	}

	if saveLocally {
		saveWallpaperLocally(url, setterName, setAt)
	}
	return
}

func saveWallpaperLocally(url string, setterName string, setAt string) {
	if setterName == "" {
		setterName = "anonymous"
	}

	folderPath, err := osext.ExecutableFolder()
	_, err = os.Stat(filepath.Join(folderPath, "download"))
	if os.IsNotExist(err) {
		log.Println("Created download directory since it did not exist")
		os.Mkdir(filepath.Join(folderPath, "download"), os.FileMode(0777))
	}

	filename := filepath.Join(folderPath, "download", "walltaker_"+setterName+"_"+setAt+"_"+path.Base(url))
	_, err = os.Stat(filename)

	if os.IsNotExist(err) {

		//log.Printf("Downloading", url, " to ", filename)
		response, err := http.Get(url)
		if err != nil {
			return
		}

		defer response.Body.Close()

		file, err := os.Create(filename)
		if err != nil {
			return
		}
		defer file.Close()
		_, err = io.Copy(file, response.Body)
	} else {
		log.Printf("Wallpaper file already exists, skipping! ")
	}
	return
}

func openMyWtWebAppLink(base string, feed int64) {
	browser.OpenURL(fmt.Sprintf("%s%d", base, feed))
}

func openWtSetterPage(setterName string) {
	if setterName != "" {
		browser.OpenURL(fmt.Sprintf("https://walltaker.joi.how/users/%s", setterName))
	}
}

func extractMD5(url string) string {
	if url != "" {
		md5str := url[strings.LastIndex(url, "/")+1:]
		md5str = strings.Split(md5str, ".")[0]
		return md5str
	}
	return ""
}

func formatE621SearchByMD5(md5 string) string {
	return fmt.Sprintf("https://e621.net/posts?tags=md5%%3A%s", md5)
}

func formatE621APISearchByMD5(md5 string) string {
	return fmt.Sprintf("https://e621.net/posts.json?tags=md5%%3A%s", md5)
}

func openE621(postUrl string) {
	// extract md5 from post url
	if postUrl != "" {
		e621Posts := getE621Data(postUrl)
		if len(e621Posts.Posts) > 0 {
			browser.OpenURL(fmt.Sprintf("https://e621.net/posts/%d", e621Posts.Posts[0].ID))
		}
		// md5url := formatE621SearchByMD5(extractMD5(postUrl))
		// browser.OpenURL(md5url)
	}
}

func getE621Data(postUrl string) E621PostsData {
	// extract md5 from post url
	if postUrl != "" {
		postsDataUrl := formatE621APISearchByMD5(extractMD5(postUrl))

		// get json data from url
		webClient := http.Client{
			Timeout: time.Second * 2, // Timeout after 2 seconds
		}

		req, err := http.NewRequest(http.MethodGet, postsDataUrl, nil)
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Set("User-Agent", "Walltaker Go Client/"+VERSION+"-"+runtime.GOOS)

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

		postsData := E621PostsData{}
		jsonErr := json.Unmarshal(body, &postsData)
		if jsonErr != nil {
			log.Fatal(jsonErr)
		}
		return postsData
	}
	return E621PostsData{}
}

func getImageUrlWithAppropriateSize(postUrl string) string {
	if postUrl != "" {
		postsData := getE621Data(postUrl)
		if len(postsData.Posts) > 0 {
			if postsData.Posts[0].File.Size > 17000000 {
				return postsData.Posts[0].Sample.URL
			} else {
				return postsData.Posts[0].File.URL
			}
		}
	}
	return ""
}

func performVersionCheck() {
	// get latest version tag from Github
	resp, err := http.Get("https://api.github.com/repos/PawCorp/walltaker-desktop-client/releases/latest")
	if err != nil {
		log.Println("Failed to check for updates: ", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read response: ", err)
		return
	}

	// parse response
	var latestRelease struct {
		TagName string `json:"tag_name"`
	}
	json.Unmarshal(body, &latestRelease)

	// compare versions
	if latestRelease.TagName != VERSION {
		log.Println("A new version of Walltaker is available!")
		log.Println("Current:", VERSION, "Latest:", latestRelease.TagName)
		log.Println("You can download it from ", "https://github.com/PawCorp/walltaker-desktop-client/releases/latest")
		errNotify := beeep.Notify("Walltaker", "A new version of Walltaker is available! Please visit https://q.pawcorp.org/wtgo to download.", "")
		if errNotify != nil {
			panic(errNotify)
		}
	}
}

func main() {
	// log to file
	fn := logOutput()
	defer fn()
	// use file lock to determine if walltaker is already running
	lockPath := "./walltaker.lock"
	if runtime.GOOS == "darwin" {
		lockPath = "/tmp/walltaker.lock"
	}
	lock := fslock.New(lockPath)
	err := lock.TryLock()
	if err != nil {
		log.Println(err.Error())
		errNotify := beeep.Notify("Walltaker", "Note: Walltaker is already running!", "")
		if errNotify != nil {
			panic(errNotify)
		}
		return
	}

	bg, err := wallpaper.Get()
	if err != nil {
		panic(err)
	}
	log.Println("Detected original wallpaper as: ", bg)

	defer lock.Unlock()
	onExit := func() {
		log.Println("Reverting wallpaper to: ", bg)
		wallpaper.SetFromFile(bg)
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	// log.Println("WALLTAKER CLIENT")
	log.Println(`
	██╗    ██╗ █████╗ ██╗     ██╗  ████████╗ █████╗ ██╗  ██╗███████╗██████╗
	██║    ██║██╔══██╗██║     ██║  ╚══██╔══╝██╔══██╗██║ ██╔╝██╔════╝██╔══██╗
	██║ █╗ ██║███████║██║     ██║     ██║   ███████║█████╔╝ █████╗  ██████╔╝
	██║███╗██║██╔══██║██║     ██║     ██║   ██╔══██║██╔═██╗ ██╔══╝  ██╔══██╗
	╚███╔███╔╝██║  ██║███████╗███████╗██║   ██║  ██║██║  ██╗███████╗██║  ██║
	 ╚══╝╚══╝ ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝

	 	` + VERSION + `. Go client by @OddPawsX
	 		 	Walltaker by Gray over at joi.how <3

	(You can minimize this window; it will periodically check in for new wallpapers)
	`)
	performVersionCheck()
	start := time.Now()

	folderPath, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	dat, err := os.ReadFile(filepath.Join(folderPath, "walltaker.toml"))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Loaded config from " + filepath.Join(folderPath, "walltaker.toml"))

	tomlDat := string(dat)

	config, _ := toml.Load(tomlDat)

	defer func() {
		if r := recover(); r != nil {
			log.Println("Ensure your .toml file is up to date!")
			errNotify := beeep.Notify("Walltaker", "Could not launch Walltaker! Ensure your .toml file is up to date.", "")
			if errNotify != nil {
				panic(errNotify)
			}
			systray.Quit()
		}
	}()

	// store crop bool
	crop := true // true before read in by config

	base := config.Get("Base.base").(string)
	feed := config.Get("Feed.feed").(int64)
	freq := config.Get("Preferences.interval").(int64)
	mode := config.Get("Preferences.mode").(string)
	saveLocally := config.Get("Preferences.saveLocally").(bool)
	useDiscord := config.Get("Preferences.discordPresence").(bool)
	notifications := config.Get("Preferences.notifications").(bool)

	if strings.ToLower(mode) == "fit" {
		crop = false
	} else if strings.ToLower(mode) == "crop" {
		crop = true
	} else {
		crop = true
	}

	builtUrl := base + strconv.FormatInt(feed, 10) + ".json"

	timeNow := time.Now() // start time for discord purposes
	if useDiscord == true {
		discorderr := client.Login("942796233033019504")
		if discorderr != nil {
			log.Fatal(discorderr)
		}

		discorderr = client.SetActivity(client.Activity{
			State: "Set my wallpaper~",
			// Details:    strings.Replace(builtUrl, ".json", "", -1),
			Details:    fmt.Sprintf("https://wt.pawcorp.org/%d", feed),
			LargeImage: "eggplant",
			LargeText:  "Powered by joi.how",
			Timestamps: &client.Timestamps{
				Start: &timeNow,
			},
		})

		if discorderr != nil {
			log.Fatal(discorderr)
		}
	}

	if saveLocally == true {
		log.Println("Local saving enabled")
		_, err := os.Stat(filepath.Join(folderPath, "download"))
		if os.IsNotExist(err) {
			log.Println("Created download directory since it did not exist")
			os.Mkdir(filepath.Join(folderPath, "download"), os.FileMode(0777))
		}
	}

	systray.SetTemplateIcon(icon.Data, icon.Data)
	systray.SetTitle("Walltaker")
	systray.SetTooltip("Walltaker")
	menuAppTimer := systray.AddMenuItem("Elapsed: 0", "Time since Walltaker started")
	menuAppTimer.SetIcon(icon.Data)
	menuAppTimer.Disabled()
	menuAppSetBy := systray.AddMenuItem("-", "Who sent your most recent wallpaper~")
	menuE621 := systray.AddMenuItem("Open e621", "Open image on e621")
	// menuAppSetBy.Disabled()
	setterName := ""
	oldWallpaperUrl := ""

	// timer loop
	go func() {
		for range time.Tick(time.Second) {
			elapsed := time.Since(start)
			menuAppTimer.SetTitle(fmt.Sprintf("Elapsed: %s", elapsed.Round(time.Second)))
		}
	}()

	// wallpaper loop
	go func() {
		bg, err := wallpaper.Get()
		if err != nil {
			panic(err)
		}
		log.Println("Detected original wallpaper as: ", bg)

		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			wallpaper.SetFromFile(bg)
			os.Exit(0)
		}()

		log.Printf("Checking in every %d seconds...\r\n", freq)

		userData := getWalltakerData(builtUrl)

		wallpaperUrl, noDataErr := getWallpaperUrlFromData(userData)
		ready := noDataErr == nil
		for ready == false {
			if noDataErr != nil {
				// log.Fatal(noDataErr)
				log.Printf("No data for ID %d, trying again in %d seconds...\r\n", feed, freq)
				time.Sleep(time.Second * time.Duration(freq))
				builtUrl = base + strconv.FormatInt(feed, 10) + ".json" // account for runtime change of poll ID
				userData = getWalltakerData(builtUrl)
				wallpaperUrl, noDataErr = getWallpaperUrlFromData(userData)
			} else {
				ready = true
			}
		}

		setterName = userData.SetBy.String
		setAt := strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", "-")
		if setterName != "" {
			log.Printf(setterName)
			log.Printf(" set your initial wallpaper: Setting... ")
			menuAppSetBy.SetTitle(fmt.Sprintf("Set by %s", setterName))
		} else {
			log.Printf("Anonymous set your initial wallpaper: Setting... ")
			menuAppSetBy.SetTitle(fmt.Sprintf("Set by %s", "Anonymous"))
		}
		goSetWallpaper(wallpaperUrl, saveLocally, setterName, setAt, notifications)

		log.Printf("Set!")

		if !crop {
			err = wallpaper.SetMode(wallpaper.Fit)
		} else if crop {
			err = wallpaper.SetMode(wallpaper.Crop)
		} else {
			err = wallpaper.SetMode(wallpaper.Crop)
		}

		oldWallpaperUrl = wallpaperUrl

		for range time.Tick(time.Second * time.Duration(freq)) {
			log.Printf("Polling... ")
			builtUrl = base + strconv.FormatInt(feed, 10) + ".json" // account for runtime change of poll ID
			userData := getWalltakerData(builtUrl)
			wallpaperUrl := userData.PostURL.String
			setterName = userData.SetBy.String
			setAt := strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", "-")

			if wallpaperUrl != oldWallpaperUrl {
				if setterName != "" {
					log.Printf(setterName)
					log.Printf(" set your wallpaper! Setting... ")
					menuAppSetBy.SetTitle(fmt.Sprintf("Set by %s", setterName))
				} else {
					log.Printf("New wallpaper found! Setting... ")
					menuAppSetBy.SetTitle(fmt.Sprintf("Set by %s", "Anonymous"))
				}
				goSetWallpaper(wallpaperUrl, saveLocally, setterName, setAt, notifications)
				log.Printf("Set!")
				oldWallpaperUrl = wallpaperUrl
			} else {
				log.Printf("Nothing new yet.")
			}
			log.Printf("\r\n")
		}
	}()

	go func() {
		menuOpenMyWtWebAppLink := systray.AddMenuItem(fmt.Sprintf("Open my Walltaker Page (%d)", feed), "Opens your link in a web browser")
		systray.AddSeparator()
		menuCropImages := systray.AddMenuItemCheckbox("Crop", "Crop images to fill the whole screen", crop)
		menuSaveImages := systray.AddMenuItemCheckbox("Save Images", "Check to save images to disk", saveLocally)
		menuDiscordPresence := systray.AddMenuItemCheckbox("Discord Presence", "Let your friends know what you're up to~", useDiscord)
		menuNotifications := systray.AddMenuItemCheckbox("Notifications", "Get a desktop notification for new wallpapers, in case you've got something maximized", notifications)
		menuSetID := systray.AddMenuItem("Set ID", "Change which IDs wallpaper feed to use")

		systray.AddSeparator()
		mQuit := systray.AddMenuItem("QUIT", "Quit the whole app")

		systray.AddSeparator()

		for {
			select {
			case <-menuE621.ClickedCh:
				openE621(oldWallpaperUrl)
			case <-menuAppSetBy.ClickedCh:
				openWtSetterPage(setterName)
			case <-menuOpenMyWtWebAppLink.ClickedCh:
				openMyWtWebAppLink(base, feed)
			case <-menuSetID.ClickedCh:
				getInputText := "Enter a Walltaker ID to poll"
				for {
					var i int
					got, ok := inputbox.InputBox("Change active Walltaker ID", getInputText, "0")
					if ok {
						log.Println("you entered:", got)
					} else {
						log.Println("No value entered")
					}
					if got == "" {
						log.Println(fmt.Sprintf("No value entered; keeping old value of %d", feed))
						break
					}
					i, err = strconv.Atoi(got)
					if err != nil {
						log.Println("Enter a valid number")
						getInputText = "Enter a Walltaker ID to poll (you entered something that was not a number last time; try again)"
					} else {
						log.Println("Got: " + strconv.Itoa(i))
						feed = int64(i)
						if useDiscord == true {
							discorderr := client.SetActivity(client.Activity{
								State: "Set my wallpaper~",
								// Details:    base + strconv.FormatInt(feed, 10),
								Details:    fmt.Sprintf("https://wt.pawcorp.org/%d", feed),
								LargeImage: "eggplant",
								LargeText:  "Powered by joi.how",
								Timestamps: &client.Timestamps{
									Start: &timeNow,
								},
							})

							if discorderr != nil {
								log.Fatal(discorderr)
							}
						}
						menuOpenMyWtWebAppLink.SetTitle(fmt.Sprintf("Open my Walltaker Page (%d)", feed))
						log.Println("Set new Walltaker poll ID")
						break
					}
				}
			case <-menuCropImages.ClickedCh:
				if menuCropImages.Checked() {
					menuCropImages.Uncheck()
					err = wallpaper.SetMode(wallpaper.Fit)
				} else {
					menuCropImages.Check()
					err = wallpaper.SetMode(wallpaper.Crop)
				}
				crop = !crop
			case <-menuSaveImages.ClickedCh:
				if menuSaveImages.Checked() {
					menuSaveImages.Uncheck()
				} else {
					menuSaveImages.Check()
				}
				saveLocally = !saveLocally
				log.Println(fmt.Sprintf("Changed saveLocally to %t", saveLocally))
			case <-menuDiscordPresence.ClickedCh:
				if menuDiscordPresence.Checked() {
					menuDiscordPresence.Uncheck()
					client.Logout()
					log.Println("Stopped Discord Presence")
				} else {
					menuDiscordPresence.Check()
					discorderr := client.Login("942796233033019504")
					if discorderr != nil {
						log.Fatal(discorderr)
					}

					discorderr = client.SetActivity(client.Activity{
						State: "Set my wallpaper~",
						// Details:    strings.Replace(builtUrl, ".json", "", -1),
						Details:    fmt.Sprintf("https://wt.pawcorp.org/%d", feed),
						LargeImage: "eggplant",
						LargeText:  "Powered by joi.how",
						Timestamps: &client.Timestamps{
							Start: &timeNow,
						},
					})

					if discorderr != nil {
						log.Fatal(discorderr)
					}
					log.Println("Started Discord Presence")
				}
				useDiscord = !useDiscord
			case <-menuNotifications.ClickedCh:
				if menuNotifications.Checked() {
					menuNotifications.Uncheck()
				} else {
					menuNotifications.Check()
				}
				notifications = !notifications
				log.Println(fmt.Sprintf("notifications set to %t", notifications))
			case <-mQuit.ClickedCh:
				systray.Quit()
				log.Println("Quit now...")
				return
			}
		}
	}()
}

func logOutput() func() {
	// modified from https://gist.github.com/jerblack/4b98ba48ed3fb1d9f7544d2b1a1be287
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	wtCacheDir := filepath.Join(cacheDir, ".walltaker")
	wtCacheLogsDir := filepath.Join(wtCacheDir, "logs")
	if _, err := os.Stat(wtCacheDir); os.IsNotExist(err) {
		err := os.Mkdir(wtCacheDir, os.FileMode(0777))
		if err != nil {
			panic(err)
		}
	}
	if _, err := os.Stat(wtCacheLogsDir); os.IsNotExist(err) {
		err := os.Mkdir(wtCacheLogsDir, os.FileMode(0777))
		if err != nil {
			panic(err)
		}
	}
	logfile := filepath.Join(wtCacheLogsDir, "walltaker.log")
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	// defer f.Close()

	log.SetOutput(f)
	// log.Println("This is a test log entry")
	return func() {
		// close file after all writes have finished
		_ = f.Close()
	}
}
