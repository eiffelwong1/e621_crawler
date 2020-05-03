package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var homeURL string = "https://www.e621.net/posts.json"
var e621RateLimiter = time.Tick(501 * time.Millisecond)

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
	response, err := http.Get(URL)
	if err != nil {
		log.Fatal(err)
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
	fmt.Println("downloading: " + URL + "\n\tto: " + path)

	webResponse := safeGet(URL)
	defer webResponse.Body.Close()

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

func downloadE621Page(storagePath string, userName string, limit int, page int) int {
	params := map[string]string{"tags": "fav:" + userName, "limit": strconv.Itoa(limit), "page": strconv.Itoa(page)}
	URL := addParamToURL(homeURL, params)
	fmt.Println(URL)

	posts := e621PostList{}

	response := safeGet(URL)
	defer response.Body.Close()

	json.NewDecoder(response.Body).Decode(&posts)

	fmt.Println(posts.Posts)

	for _, p := range posts.Posts {
		<-e621RateLimiter
		if p.File.URL == "" {
			go handleMissingURLWithPostID(p.ID)
			continue
		}
		go downloadPhoto(storagePath+"/"+getFileNameFromPost(p), p.File.URL)
	}

	return len(posts.Posts)
}

func handleMissingURLWithPostID(pid int) {
	missingPID = append(missingPID, pid)
}

func main() {
	var userName = "eiffelwong1"
	var storagePath = "/Users/eiffelwong1/Desktop/e621"
	var postLimit = 320

	var page int = 0

	fmt.Println("start")

	for {
		<-e621RateLimiter
		downloadCount := downloadE621Page(storagePath, userName, postLimit, page)
		if downloadCount < postLimit {
			break
		}
	}

	return
}
