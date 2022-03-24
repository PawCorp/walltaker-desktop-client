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

func getWalltakerData(url string) WalltakerData {
	webClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "Walltaker Go Client/2.0.0-"+runtime.GOOS)

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
			fmt.Println("Ouch! Had a problem while downloading your wallpaper. This is a mac specific bug!")
			fmt.Println("Full error: ", err)
		}
		wallpaper.SetFromFile(file)
		defer cleanUpCacheForMac(file) // OK to delete after setting wallpaper, MacOS shows bg w/o file remaining there
	} else {
		err := wallpaper.SetFromURL(url)

		if err != nil {
			fmt.Println("Ouch! Had a problem while setting your wallpaper.")
			fmt.Println("Full error: ", err)
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
	filename := filepath.Join(folderPath, "download", "walltaker_"+setterName+"_"+setAt+"_"+path.Base(url))
	_, err = os.Stat(filename)

	if os.IsNotExist(err) {

		//fmt.Printf("Downloading", url, " to ", filename)
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
		fmt.Printf("Wallpaper file already exists, skipping! ")
	}
	return
}

func openMyWtWebAppLink(base string, feed int64) {
	browser.OpenURL(fmt.Sprintf("%s%d", base, feed))
}

func main() {
	// use file lock to determine if walltaker is already running
	lock := fslock.New("./walltaker.lock")
	err := lock.TryLock()
	if err != nil {
		fmt.Println(err.Error())
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
	fmt.Println("Detected original wallpaper as: ", bg)

	defer lock.Unlock()
	onExit := func() {
		fmt.Println("Reverting wallpaper to: ", bg)
		wallpaper.SetFromFile(bg)
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	// fmt.Println("WALLTAKER CLIENT")
	fmt.Println(`
	██╗    ██╗ █████╗ ██╗     ██╗  ████████╗ █████╗ ██╗  ██╗███████╗██████╗
	██║    ██║██╔══██╗██║     ██║  ╚══██╔══╝██╔══██╗██║ ██╔╝██╔════╝██╔══██╗
	██║ █╗ ██║███████║██║     ██║     ██║   ███████║█████╔╝ █████╗  ██████╔╝
	██║███╗██║██╔══██║██║     ██║     ██║   ██╔══██║██╔═██╗ ██╔══╝  ██╔══██╗
	╚███╔███╔╝██║  ██║███████╗███████╗██║   ██║  ██║██║  ██╗███████╗██║  ██║
	 ╚══╝╚══╝ ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝

	 	v2.0.0. Go client by @OddPawsX
	 		 	Walltaker by Gray over at joi.how <3

	(You can minimize this window; it will periodically check in for new wallpapers)
	`)

	start := time.Now()

	folderPath, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	dat, err := os.ReadFile(filepath.Join(folderPath, "walltaker.toml"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Loaded config from " + filepath.Join(folderPath, "walltaker.toml"))

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

	base := config.Get("Base.base").(string)
	feed := config.Get("Feed.feed").(int64)
	freq := config.Get("Preferences.interval").(int64)
	mode := config.Get("Preferences.mode").(string)
	saveLocally := config.Get("Preferences.saveLocally").(bool)
	useDiscord := config.Get("Preferences.discordPresence").(bool)
	notifications := config.Get("Preferences.notifications").(bool)

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
		fmt.Println("Local saving enabled")
		_, err := os.Stat(filepath.Join(folderPath, "download"))
		if os.IsNotExist(err) {
			fmt.Println("Created download directory since it did not exist")
			os.Mkdir(filepath.Join(folderPath, "download"), os.FileMode(0777))
		}
	}

	systray.SetTemplateIcon(icon.Data, icon.Data)
	systray.SetTitle("Walltaker")
	systray.SetTooltip("Walltaker")
	menuAppTimer := systray.AddMenuItem("Elapsed: 0", "Time since Walltaker started")
	menuAppTimer.SetIcon(icon.Data)
	menuAppTimer.Disabled()

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
		fmt.Println("Detected original wallpaper as: ", bg)

		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			wallpaper.SetFromFile(bg)
			os.Exit(0)
		}()

		fmt.Printf("Checking in every %d seconds...\r\n", freq)

		userData := getWalltakerData(builtUrl)

		wallpaperUrl, noDataErr := getWallpaperUrlFromData(userData)
		ready := noDataErr == nil
		for ready == false {
			if noDataErr != nil {
				// log.Fatal(noDataErr)
				fmt.Printf("No data for ID %d, trying again in %d seconds...\r\n", feed, freq)
				time.Sleep(time.Second * time.Duration(freq))
				builtUrl = base + strconv.FormatInt(feed, 10) + ".json" // account for runtime change of poll ID
				userData = getWalltakerData(builtUrl)
				wallpaperUrl, noDataErr = getWallpaperUrlFromData(userData)
			} else {
				ready = true
			}
		}

		setterName := userData.SetBy.String
		setAt := strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", "-")
		if setterName != "" {
			fmt.Printf(setterName)
			fmt.Printf(" set your initial wallpaper: Setting... ")
		} else {
			fmt.Printf("Anonymous set your initial wallpaper: Setting... ")
		}
		goSetWallpaper(wallpaperUrl, saveLocally, setterName, setAt, notifications)
		fmt.Printf("Set!")

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
			builtUrl = base + strconv.FormatInt(feed, 10) + ".json" // account for runtime change of poll ID
			userData := getWalltakerData(builtUrl)
			wallpaperUrl := userData.PostURL.String
			setterName := userData.SetBy.String
			setAt := strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", "-")

			if wallpaperUrl != oldWallpaperUrl {
				if setterName != "" {
					fmt.Printf(setterName)
					fmt.Printf(" set your wallpaper! Setting... ")
				} else {
					fmt.Printf("New wallpaper found! Setting... ")
				}
				goSetWallpaper(wallpaperUrl, saveLocally, setterName, setAt, notifications)
				fmt.Printf("Set!")
				oldWallpaperUrl = wallpaperUrl
			} else {
				fmt.Printf("Nothing new yet.")
			}
			fmt.Printf("\r\n")
		}
	}()

	go func() {
		menuOpenMyWtWebAppLink := systray.AddMenuItem(fmt.Sprintf("Open my Walltaker Page (%d)", feed), "Opens your link in a web browser")
		systray.AddSeparator()
		menuSaveImages := systray.AddMenuItemCheckbox("Save Images", "Check to save images to disk", saveLocally)
		menuDiscordPresence := systray.AddMenuItemCheckbox("Discord Presence", "Let your friends know what you're up to~", useDiscord)
		menuNotifications := systray.AddMenuItemCheckbox("Notifications", "Get a desktop notification for new wallpapers, in case you've got something maximized", notifications)
		menuSetID := systray.AddMenuItem("Set ID", "Change which IDs wallpaper feed to use")

		systray.AddSeparator()
		mQuit := systray.AddMenuItem("QUIT", "Quit the whole app")

		systray.AddSeparator()

		for {
			select {
			case <-menuOpenMyWtWebAppLink.ClickedCh:
				openMyWtWebAppLink(base, feed)
			case <-menuSetID.ClickedCh:
				getInputText := "Enter a Walltaker ID to poll"
				for {
					var i int
					got, ok := inputbox.InputBox("Change active Walltaker ID", getInputText, "0")
					if ok {
						fmt.Println("you entered:", got)
					} else {
						fmt.Println("No value entered")
					}
					if got == "" {
						fmt.Println(fmt.Sprintf("No value entered; keeping old value of %d", feed))
						break
					}
					i, err = strconv.Atoi(got)
					if err != nil {
						fmt.Println("Enter a valid number")
						getInputText = "Enter a Walltaker ID to poll (you entered something that was not a number last time; try again)"
					} else {
						fmt.Println("Got: " + strconv.Itoa(i))
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
						fmt.Println("Set new Walltaker poll ID")
						break
					}
				}
			case <-menuSaveImages.ClickedCh:
				if menuSaveImages.Checked() {
					menuSaveImages.Uncheck()
				} else {
					menuSaveImages.Check()
				}
				saveLocally = !saveLocally
				fmt.Println(fmt.Sprintf("Changed saveLocally to %t", saveLocally))
			case <-menuDiscordPresence.ClickedCh:
				if menuDiscordPresence.Checked() {
					menuDiscordPresence.Uncheck()
					client.Logout()
					fmt.Println("Stopped Discord Presence")
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
					fmt.Println("Started Discord Presence")
				}
				useDiscord = !useDiscord
			case <-menuNotifications.ClickedCh:
				if menuNotifications.Checked() {
					menuNotifications.Uncheck()
				} else {
					menuNotifications.Check()
				}
				notifications = !notifications
				fmt.Println(fmt.Sprintf("notifications set to %t", notifications))
			case <-mQuit.ClickedCh:
				systray.Quit()
				fmt.Println("Quit now...")
				return
			}
		}
	}()
}
