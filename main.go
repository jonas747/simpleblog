package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/zenazn/goji"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type Post struct {
	Posted string
	Author string
	Path   string
	Title  string
	Id     int

	HTML                template.HTML
	Markdown            string
	currentModifiedTime time.Time // Used to check if the post has been updated
}

var (
	templates      *template.Template
	templatesMutex sync.RWMutex

	mainListing *Listing
)

// Get posts from before id(or most recent if id is -1)
func getPosts(before, amount int) ([]Post, error) {
	if amount < 1 {
		return nil, errors.New("Trying to get less than 1 posts?!")
	}

	out := make([]Post, amount)
	nId := before - 1
	n := 0
	if before == -1 {
		// Get latest...
		p, err := getPost(-1)
		if err != nil {
			return nil, err
		}
		out[0] = p

		if amount == 1 {
			return out, nil
		}
		n = 1
		nId = p.Id - 1
	}

	for n < amount {
		if nId < 0 {
			return out[:n], nil
		}

		p, err := getPost(nId)
		if err == nil {
			out[n] = p
		} else {
			log.Printf("Error getting post %d, Reason: %s", nId, err.Error())
		}

		nId--
		n++
	}

	return out, nil
}

func getPost(id int) (Post, error) {
	mainListing.Mutex.RLock()

	if id >= len(mainListing.Posts) {
		mainListing.Mutex.RUnlock()
		return Post{}, errors.New(fmt.Sprintf("Attempting to get non-existant post %d", id))
	}

	if id < 0 {
		// Get Latest post
		id = len(mainListing.Posts) - 1
	}

	// Make a copy of the post
	post := *mainListing.Posts[id]
	mainListing.Mutex.RUnlock()

	// Recompile if not compiled allready
	if string(post.HTML) == "" {
		// Load the markdown from fs if needed
		if post.Markdown == "" {
			file, err := ioutil.ReadFile(post.Path)
			if err != nil {
				return Post{}, err
			}
			post.Markdown = string(file)
		}
		post.HTML = template.HTML(compileMarkdown(post.Markdown))

		// Also apply the changes to the main listing
		mainListing.Mutex.Lock()
		mainListing.Posts[id] = &post
		mainListing.Mutex.Unlock()
	}

	return post, nil
}

func compileMarkdown(md string) string {
	out := github_flavored_markdown.Markdown([]byte(md))
	return string(out)
}

func main() {
	flag.Parse()

	// Do an initial update of the post listing
	ml, err := updateListing(nil)
	if err != nil {
		panic(err)
	}
	mainListing = ml

	// Update all the templates
	loadTemplates()

	stopChan := make(chan bool)
	go intervalUpdater(10*time.Second, stopChan)

	goji.Get("/", handleHome)
	goji.Handle("/static/*", http.FileServer(http.Dir(".")))
	goji.Serve()
	stopChan <- true
}

func loadTemplates() {
	tmpl := template.New("main")

	_, err := tmpl.ParseFiles("templates/main.tmpl")
	if err != nil {
		log.Println("Error parsing templates..", err)
		return
	}
	templatesMutex.Lock()
	templates = tmpl
	templatesMutex.Unlock()
}
