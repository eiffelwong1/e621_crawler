package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

//HomeURL is the URL for where to scrape E621
var HomeURL string = "https://www.e621.net/posts.json"

//UserDataYAML is path where userdata will be stored
var UserDataYAML string = "./UserData.yaml"
var e621RateLimiter = time.Tick(1010 * time.Millisecond)

var missingPID = []int{}

type e621Post struct {
	ID   int `json:"id"`
	File struct {
		URL string `json:"url"`
		Ext string `json:"ext"`
	} `json:"file"`
}

type e621PostList struct {
	Posts []e621Post `json:"posts"`
}

func safeGet(URL string) *http.Response {
	// !! the caller are responsible to call "defer response.Body.Close()" for clean up
	<-e621RateLimiter
	response, err := http.Get(URL)
	if err != nil {
		fmt.Println("ERROR: " + strconv.Itoa(response.StatusCode) + " when getting: " + URL)
		response.Body.Close()
		if response.StatusCode >= 500 && response.StatusCode < 600 {
			time.Sleep(5 * time.Second)
			return safeGet(URL)
		}
		log.Fatal(response.StatusCode, err)
	}
	return response
}

func addParamToURL(URL string, param map[string]string) string {
	if len(param) == 0 {
		return URL
	}

	var paramStr = make([]string, len(param))
	i := 0
	for k, v := range param {
		paramStr[i] = k + "=" + v
		i++
	}

	return URL + "?" + strings.Join(paramStr, "&")

}

func getFileNameFromPost(post e621Post) string {
	return strconv.Itoa(post.ID) + "." + post.File.Ext
}

func downloadPhoto(path string, URL string) bool {
	webResponse := safeGet(URL)
	defer webResponse.Body.Close()

	fmt.Println(URL + " ->" + path)

	fileResponse, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
		return false
	}
	defer fileResponse.Close()

	//TODO: webResponse can be cut off by e621 server side, so need better error handing here
	_, err = io.Copy(fileResponse, webResponse.Body)
	if err != nil {
		log.Fatal(err)
		return false
	}

	return true
}

func downloadE621Page(userName string, limit int, page int) e621PostList {
	params := map[string]string{"tags": "fav:" + userName, "limit": strconv.Itoa(limit), "page": strconv.Itoa(page)}
	URL := addParamToURL(HomeURL, params)
	fmt.Println(URL)

	response := safeGet(URL)
	defer response.Body.Close()

	posts := e621PostList{}
	json.NewDecoder(response.Body).Decode(&posts)
	return posts
}

func downloadNewPhotosFromE621PageList(storagePath string, posts e621PostList, existingPostIDs map[int]bool) {
	for i := 0; i < len(posts.Posts); i++ {
		if _, ok := existingPostIDs[posts.Posts[i].ID]; ok {
			log.Println(strconv.Itoa(posts.Posts[i].ID) + " exist, skipping download for this PID")
			removeFromE621PageList(&posts, i)
			i--
		}
	}
	downloadAllPhotosFromE621PageList(storagePath, posts)
}

func downloadAllPhotosFromE621PageList(storagePath string, posts e621PostList) {
	for _, p := range posts.Posts {
		if p.File.URL == "" {
			go handleMissingURLWithPostID(p.ID)
			continue
		}
		go downloadPhoto(storagePath+"/"+getFileNameFromPost(p), p.File.URL)
	}
}

func removeFromE621PageList(posts *e621PostList, i int) {
	posts.Posts[i] = posts.Posts[len(posts.Posts)-1]
	posts.Posts = posts.Posts[:len(posts.Posts)-1]
}

func handleMissingURLWithPostID(pid int) {
	//TODO: lol, do something to the pid?
	missingPID = append(missingPID, pid)
}

func getExistingPIDInStorage(storagePath string) map[int]bool {
	exsistingPID := map[int]bool{}
	err := filepath.Walk(
		storagePath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Fatal(err)
			}
			PID, err := strconv.Atoi(removeFileNameExt(info.Name()))
			if err != nil {
				return nil
			}
			exsistingPID[PID] = true
			return nil
		},
	)
	if err != nil {
		log.Println(err)
	}
	return exsistingPID
}

func removeFileNameExt(name string) string {
	splitString := strings.Split(name, ".")
	return strings.Join(splitString[:len(splitString)-1], ".")
}

//UserData stores the basic user information
type UserData struct {
	UserName    string //user name on E621 or whom's fav you wanted to download
	StoragePath string //where photos are stored
}

func getUserData() UserData {
	//check if store user data existed

	var userData UserData
	// if the expected filepath contains a valid user data yaml, use it or promp User for input
	if _, err := os.Stat(UserDataYAML); err == nil {
		yamlFile, err := ioutil.ReadFile(UserDataYAML)
		err = yaml.Unmarshal(yamlFile, &userData)
		if err != nil {
			log.Println(err)
		} else if prompForUsingStoredUserData(userData) {
			return userData
		}
	}
	return prompForUserData()
}

func storeUserData(userData UserData) {

}

func prompForString(question string) string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(question)
	scanner.Scan()
	if scanner.Err() != nil {
		log.Println(scanner.Err())
	}
	return scanner.Text()
}

func prompForUserData() UserData {
	var userData UserData
	userData.UserName = prompForString("please enter username for e621: ")
	userData.StoragePath = prompForString("please enter the where you want to store the photos (or '.' for the current file): ")
	return userData
}

func prompForUsingStoredUserData(userData UserData) bool {

	fmt.Printf("stored info found\n"+
		"------------------------\n"+
		"User Name : %s\n"+
		"Storage Path : %s\n"+
		"use stored info ? (Y/N)\n", userData)
	for {
		result := prompForString("please enter input: ")
		if result[0] == 'y' || result[0] == 'Y' {
			return true
		} else if result[0] == 'n' || result[0] == 'N' {
			return false
		} else {
			fmt.Println("please only input Y or N")
		}

	}
}

func main() {
	currentUserData := prompForUserData()
	var userName = currentUserData.UserName
	var storagePath = currentUserData.StoragePath
	var postLimit = 320

	var pageNum int = 0

	fmt.Println("start")
	existingPostIDs := getExistingPIDInStorage(storagePath)
	print(existingPostIDs)

	for {
		e621Page := downloadE621Page(userName, postLimit, pageNum)
		pageNum++
		go downloadNewPhotosFromE621PageList(storagePath, e621Page, existingPostIDs)

		if len(e621Page.Posts) < postLimit {
			break
		}

	}

	fmt.Println("done")

	return
}
