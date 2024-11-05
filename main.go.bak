package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/rivo/tview"
)

// Structure to parse story details
type Story struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func main() {
	app := tview.NewApplication()

	// Create a TextView for the scrolling feed
	feedView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	feedView.SetBorder(true).SetTitle("Latest Hacker News Headlines")

	// Function to periodically fetch and update the feed
	go func() {
		for {
			stories, err := fetchTopStories()
			if err != nil {
				appendMessage(feedView, fmt.Sprintf("[red]Error: %v[-]", err))
			} else {
				for _, story := range stories {
					appendMessage(feedView, fmt.Sprintf("[green]%s[-] - %s", story.Title, story.URL))
				}
			}

			// Wait before fetching again
			time.Sleep(3 * time.Second) // Fetch new headlines every minute
		}
	}()

	// Set up and run the app
	if err := app.SetRoot(feedView, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// Fetches the top story IDs from Hacker News API and retrieves their details
func fetchTopStories() ([]Story, error) {
	// Step 1: Fetch top story IDs
	resp, err := http.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var storyIDs []int
	if err := json.Unmarshal(body, &storyIDs); err != nil {
		return nil, err
	}

	// Step 2: Fetch details for the top 10 stories
	var stories []Story
	for i := 0; i < 10 && i < len(storyIDs); i++ {
		story, err := fetchStoryDetails(storyIDs[i])
		if err == nil {
			stories = append(stories, story)
		}
	}

	return stories, nil
}

// Fetches story details for a given story ID
func fetchStoryDetails(id int) (Story, error) {
	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	resp, err := http.Get(url)
	if err != nil {
		return Story{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Story{}, err
	}

	var story Story
	if err := json.Unmarshal(body, &story); err != nil {
		return Story{}, err
	}

	return story, nil
}

// Appends a message to the feed and handles "fading" by limiting line count
func appendMessage(view *tview.TextView, message string) {
	// Get current text and split by lines
	currentText := view.GetText(false)
	lines := strings.Split(currentText, "\n")

	// Limit to the last 20 lines for fading effect
	if len(lines) >= 10 {
		lines = lines[1:] // Remove the oldest line to simulate fading out
	}

	// Append the new message and update the view
	lines = append(lines, message)
	view.SetText(strings.Join(lines, "\n"))
}

