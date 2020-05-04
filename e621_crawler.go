package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var homeURL string = "https://www.e621.net/posts.json"
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

	_, err = io.Copy(fileResponse, webResponse.Body)
	if err != nil {
		log.Fatal(err)
		return false
	}

	return true
}

func downloadE621Page(userName string, limit int, page int) e621PostList {
	params := map[string]string{"tags": "fav:" + userName, "limit": strconv.Itoa(limit), "page": strconv.Itoa(page)}
	URL := addParamToURL(homeURL, params)
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

func main() {
	var userName = "eiffelwong1"
	var storagePath = "/Users/eiffelwong1/Desktop/e621"
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
