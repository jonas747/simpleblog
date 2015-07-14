package main

import (
	"github.com/zenazn/goji/web"
	"log"
	"net/http"
)

type HomeContent struct {
	Posts []Post
	Err   string
}

func commonResp(w http.ResponseWriter, tmpl string, data interface{}, status int) {
	w.WriteHeader(status)

	templatesMutex.RLock()
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		log.Println("Error executing template: ")
	}
	templatesMutex.RUnlock()
}

func handleHome(c web.C, w http.ResponseWriter, r *http.Request) {
	content := HomeContent{}

	// Get the 5 latest posts
	p, err := getPosts(-1, 5)

	if err != nil {
		content.Err = err.Error()
		commonResp(w, "postspage", content, http.StatusInternalServerError)
		return
	}

	content.Posts = p
	commonResp(w, "postspage", content, http.StatusOK)
}

func handleViewPost(c web.C, w http.ResponseWriter, r *http.Request) {

}

func handleViewBefore(c web.C, w http.ResponseWriter, r *http.Request) {

}
