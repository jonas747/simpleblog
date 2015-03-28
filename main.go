package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"
)

type Post struct {
	Posted    string
	Author    string
	Path      string
	Title     string
	Body      string
	ViewCount int
	Id        int
}

var incrViewCountChan chan map[int][]string

var viewCounts [][]string
var viewCountsMutex sync.Mutex

var postListing []Post
var postListingLock sync.Mutex

var mainTemplate = template.New("main.tmpl")
var postTemplate = template.New("post.tmpl")

var addr = flag.String("addr", ":7447", "Address to listen on")

func getPostListing(path string) ([]Post, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return []Post{}, err
	}
	var pl []Post
	err = json.Unmarshal(file, &pl)
	if err != nil {
		return []Post{}, err
	}
	return pl, nil
}

// Increases viewcouts, returns true if any changes were made
func incrViewCounts(pairs map[int][]string) bool {
	viewCountsMutex.Lock()
	changed := false
	for id, sources := range pairs {
		for _, addIp := range sources {
			found := false

			// Extend if needed
			if len(viewCounts) <= id {
				vc2 := make([][]string, id+1)
				copy(vc2, viewCounts)
				viewCounts = vc2
				changed = true
			}

			for _, curIp := range viewCounts[id] {
				if curIp == addIp {
					found = true
					break
				}
			}

			if !found {
				changed = true
				viewCounts[id] = append(viewCounts[id], addIp)
			}
		}
	}
	viewCountsMutex.Unlock()
	return changed
}

func saveViewCounts(counts [][]string) error {
	serilalized, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("counter.json", serilalized, os.FileMode(0744))
	return err
}

func viewCounter(in chan map[int][]string) {
	// Load viewcounts from disk
	file, err := ioutil.ReadFile("counter.json")
	if err != nil {
		log.Println("ViewCounter: ", err)
		log.Println("error loading existing counter, creating new counter...")
		viewCountsMutex.Lock()
		viewCounts = make([][]string, 0)
		viewCountsMutex.Unlock()
	} else {
		viewCountsMutex.Lock()
		err = json.Unmarshal(file, &viewCounts)
		viewCountsMutex.Unlock()
		if err != nil {
			log.Println("ViewCounter: ", err)
			return
		}
	}

	for {
		select {
		case incr := <-in:
			changed := incrViewCounts(incr)
			if changed {
				err = saveViewCounts(viewCounts)
				if err != nil {
					log.Println("ViewCounter: ", err)
				}
			}
		}

	}
}

// Get posts from before id(or most recent if id is -1)
func getPosts(before, amount int) ([]Post, error) {
	postListingLock.Lock()
	defer postListingLock.Unlock()

	out := make([]Post, 0)
	if before == -1 {
		before = len(postListing) - 1
	}
	// most recent posts
	lastPostId := len(postListing) - 1
	startPos := lastPostId - amount
	if startPos < 0 {
		startPos = 0
	}

	viewCountsMutex.Lock()
	defer viewCountsMutex.Unlock()
	for i := before; i >= 0; i-- {
		post := postListing[i]

		if post.Path == "" {
			// Skip...
			continue
		}

		if i < len(viewCounts) {
			post.ViewCount = len(viewCounts[i])
		}

		// Load the body from disk
		file, err := ioutil.ReadFile(post.Path)
		if err != nil {
			return []Post{}, err
		}

		post.Body = string(file)
		post.Id = i
		out = append(out, post)
		if len(out) > amount {
			break
		}
	}
	return out, nil
}

// Peiodically refreshes the post listing
func postLister() {
	ticker := time.NewTicker(time.Duration(10) * time.Second)
	for {
		select {
		case <-ticker.C:
			pl, err := getPostListing("listing.json")
			if err != nil {
				log.Println("PostLister: ", err)
				continue
			}
			postListingLock.Lock()
			postListing = pl
			postListingLock.Unlock()
		}
	}
}

func main() {
	flag.Parse()
	incrViewCountChan = make(chan map[int][]string, 10)
	go viewCounter(incrViewCountChan)
	go postLister()

	// So we dont crash if we receive a visitor before 10 seconds have passed
	pl, err := getPostListing("listing.json")
	if err != nil {
		panic(err)
	}
	postListing = pl

	// Load the main template
	mainTemplate, err = mainTemplate.ParseFiles("main.tmpl")
	if err != nil {
		panic(err)
	}

	postTemplate, err = postTemplate.ParseFiles("post.tmpl")
	if err != nil {
		panic(err)
	}

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))))
	http.HandleFunc("/", HandleFront)
	err = http.ListenAndServe(*addr, nil)
	if err != nil {
		panic(err)
	}
}
