package main

import (
	"bytes"
	"github.com/russross/blackfriday"
	"log"
	"net/http"
	"strings"
)

func buildHTML(posts []Post) ([]byte, error) {
	var buf bytes.Buffer
	for _, v := range posts {
		err := postTemplate.Execute(&buf, v)
		if err != nil {
			return []byte{}, err
		}
	}
	compiledMD := blackfriday.MarkdownCommon(buf.Bytes())

	var finalHTML bytes.Buffer
	err := mainTemplate.Execute(&finalHTML, string(compiledMD))

	return finalHTML.Bytes(), err
}

func HandleFront(w http.ResponseWriter, r *http.Request) {
	if r.URL.String() != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	posts, err := getPosts(-1, 10)
	if err != nil {
		log.Println("Error in Handling / (getting posts): ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Increase the views of the posts
	incr := make(map[int][]string, 0)
	li := strings.LastIndex(r.RemoteAddr, ":")
	remote := r.RemoteAddr[:li]
	for _, post := range posts {
		incr[post.Id] = []string{remote}
	}
	incrViewCountChan <- incr

	html, err := buildHTML(posts)
	if err != nil {
		log.Println("Error in Handling / (build html): ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(html)
}
