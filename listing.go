package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

type Listing struct {
	Posts []*Post
	Mutex sync.RWMutex
}

func updateListing(current *Listing) (*Listing, error) {
	// First we load the listing.json
	file, err := ioutil.ReadFile("listing.json")
	if err != nil {
		return nil, err
	}

	var newListing Listing
	err = json.Unmarshal(file, &newListing)
	if err != nil {
		return nil, err
	}

	if current == nil {
		return &newListing, nil
	}

	current.Mutex.RLock()
	defer current.Mutex.RUnlock()

	for nIndex, nPost := range newListing.Posts {
		found := -1
		for cIndex, cPost := range current.Posts {
			if nPost.Id == cPost.Id {
				found = cIndex
				break
			}
		}

		info, err := os.Stat(nPost.Path)
		if err != nil {
			log.Printf("Error loading file info for: (id: %d) %s, skipping... Reason: %s\n", nPost.Id, nPost.Path, err.Error())
			continue
		}

		if found != -1 {
			// We allready have this post loaded, so we dont want to erase the preompiled html unless it has been changed
			cPost := current.Posts[found]
			// load fle info

			newMTime := info.ModTime()
			if newMTime.Sub(cPost.currentModifiedTime) == 0 {
				//not changed, just take the the entire post from prevous update
				newListing.Posts[nIndex] = cPost
			}
		}

		// Set last modified time
		nPost.currentModifiedTime = info.ModTime()
	}

	return &newListing, nil
}

// update at a interval, maybe use fsnotify later?
func intervalUpdater(interval time.Duration, stop chan bool) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			// Update post listing
			nl, err := updateListing(mainListing)
			if err != nil {
				log.Printf("Error updating post listing, Reason: ", err.Error())
			}
			if nl != nil {
				mainListing.Mutex.Lock()
				mainListing.Posts = nl.Posts
				mainListing.Mutex.Unlock()
			}

			loadTemplates()
		case <-stop:
			return
		}
	}
}
