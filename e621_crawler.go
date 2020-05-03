package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var HomeURL string = "https://www.e621.net/posts.json"

func safeGet(URL string) *http.Response {
	// !! the caller are responsible to call "defer response.Body.Close()" for clean up
	response, err := http.Get(URL)
	if err != nil {
		log.Fatal(err)
	}
	return response
}

func getE621Post(tag map[string]string, limit int, page int) string {
	return "hi"
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

func getFileNameFromPost(post e621Post) string {
	return strconv.Itoa(post.ID) + "." + post.File.Ext
}

func downloadPhoto(URL string, path string) {
	fmt.Println("downloading: " + URL + "\n\tto: " + path)

}

func main() {
	var UserName = "eiffelwong1"
	var StorageLoc = "/Users/eiffelwong1/Desktop/e621"
	var Limit = 2
	var Page = 0

	fmt.Println("start")

	params := map[string]string{"limit": strconv.Itoa(Limit), "tags": "fav:" + UserName, "page": strconv.Itoa(Page)}
	URL := addParamToURL(HomeURL, params)
	fmt.Println(URL)

	posts := e621PostList{}

	response := safeGet(URL)
	defer response.Body.Close()

	json.NewDecoder(response.Body).Decode(&posts)

	fmt.Println(posts.Posts)

	for _, p := range posts.Posts {
		downloadPhoto(p.File.URL, StorageLoc+"/"+getFileNameFromPost(p))
	}

	return
}
