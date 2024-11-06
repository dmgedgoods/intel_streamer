package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/rivo/tview"
)

// Structure to parse story details
type Story struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// HighValueInsight represents high-value information as classified by Ollama
type HighValueInsight struct {
	Title    string
	URL      string
	Summary  string
	Priority string
}

const (
	maxEntries      = 20  // Maximum number of entries to display
	numStoriesFetch = 1   // Number of stories to fetch each time
)

// Fade levels with different color intensities
var fadeLevels = []string{
	"[white]", // Newest entry (brightest)
	"[lightgray]",
	"[gray]",
	"[darkgray]",
	"[black]", // Oldest entry (faded out)
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

	feedView.SetBorder(true).SetTitle("High-Value Intelligence Feed")

	// List to store entries and track seen stories
	var entries []string
	seenStoryIDs := make(map[int]bool)

	// Function to periodically fetch, analyze, and update the feed
	go func() {
		for {
			stories, err := fetchTopStories(seenStoryIDs)
			if err != nil {
				entries = addEntry(entries, fmt.Sprintf("[red]Error: %v[-]", err))
			} else {
				for _, story := range stories {
					// Use Ollama to determine if this story is high-value
					insight, err := analyzeWithOllama(story)
					if err != nil {
						insight.Summary = "[red]Analysis not available[-]"
					}
					message := fmt.Sprintf("[yellow]Priority: %s[-]\n[green]%s[-]\n%s\n%s",
						insight.Priority, insight.Title, insight.URL, insight.Summary)
					entries = addEntry(entries, message)
				}
			}

			// Update the TextView with the faded entries list
			feedView.SetText(formatEntriesWithFade(entries))

			// Wait before fetching again
			time.Sleep(5 * time.Second) // Adjust interval as needed
		}
	}()

	// Set up and run the app
	if err := app.SetRoot(feedView, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// Adds a new entry to the top of the list and keeps the most recent maxEntries entries
func addEntry(entries []string, message string) []string {
	// Add the new message to the top of the list
	entries = append([]string{message}, entries...)

	// If the list exceeds the maximum number of entries, remove the oldest one
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}

	return entries
}

// Formats entries with a fading effect by applying different colors based on age
func formatEntriesWithFade(entries []string) string {
	var formattedEntries []string

	for i, entry := range entries {
		// Determine the fade level based on the entry's position in the list
		fadeIndex := i * (len(fadeLevels) - 1) / len(entries)
		color := fadeLevels[fadeIndex]
		formattedEntries = append(formattedEntries, color+entry+"[-]")
	}

	return strings.Join(formattedEntries, "\n\n")
}

// Fetches the top stories from Hacker News API, filtering out already-seen stories
func fetchTopStories(seenStoryIDs map[int]bool) ([]Story, error) {
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

	// Fetch details for the first numStoriesFetch unique stories that haven't been seen
	var stories []Story
	for _, id := range storyIDs {
		if !seenStoryIDs[id] { // Check if story has already been displayed
			story, err := fetchStoryDetails(id)
			if err == nil {
				stories = append(stories, story)
				seenStoryIDs[id] = true // Mark as seen
			}
		}
		if len(stories) >= numStoriesFetch {
			break
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

// Uses Ollama to analyze and classify the importance of an article
func analyzeWithOllama(story Story) (HighValueInsight, error) {
	// Format the prompt for Ollama to analyze the story
	prompt := fmt.Sprintf("You are an expert cybersecurity analyst. Analyze the following headline and URL to determine its relevance and priority in cybersecurity. Respond with a priority level (e.g., High, Medium, Low) and provide a summary if relevant. Keep everything very short.\n\nTitle: %s\nURL: %s", story.Title, story.URL)

	// Run Ollama command with `ollama run`
	cmd := exec.Command("ollama", "run", "llama3.2", prompt)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return HighValueInsight{}, fmt.Errorf("failed to execute Ollama command: %v", err)
	}

	// Parse the output from Ollama
	output := out.String()
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return HighValueInsight{
			Title:    story.Title,
			URL:      story.URL,
			Summary:  "[red]Invalid response format from Ollama[-]",
			Priority: "Low",
		}, nil
	}

	priority := lines[0] // Assuming the first line is the priority
	summary := strings.Join(lines[1:], " ")

	return HighValueInsight{
		Title:    story.Title,
		URL:      story.URL,
		Summary:  summary,
		Priority: priority,
	}, nil
}

