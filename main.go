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
	Title   string
	URL     string
	Summary string
	Priority string
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

	// Function to periodically fetch, analyze, and update the feed
	go func() {
		for {
			stories, err := fetchTopStories()
			if err != nil {
				appendMessage(feedView, fmt.Sprintf("[red]Error: %v[-]", err))
			} else {
				for _, story := range stories {
					// Use Ollama to determine if this story is high-value
					insight, err := analyzeWithOllama(story)
					if err != nil {
						insight.Summary = "[red]Analysis not available[-]"
					}
					appendMessage(feedView, fmt.Sprintf("[yellow]Priority: %s[-]\n[green]%s[-]\n%s\n%s",
						insight.Priority, insight.Title, insight.URL, insight.Summary))
				}
			}

			// Wait before fetching again
			time.Sleep(10 * time.Second) // Fetch new headlines every minute
		}
	}()

	// Set up and run the app
	if err := app.SetRoot(feedView, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// Fetches the top story IDs from Hacker News API and retrieves their details
func fetchTopStories() ([]Story, error) {
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

	// Fetch details for the top 10 stories
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

// Uses Ollama to analyze and classify the importance of an article
func analyzeWithOllama(story Story) (HighValueInsight, error) {
	// Format the prompt for Ollama to analyze the story
	prompt := fmt.Sprintf("Analyze the following headline and URL to determine its relevance and priority in cybersecurity:\n\nTitle: %s\nURL: %s\n\nOutput a priority level (e.g., High, Medium, Low) and provide a summary if relevant. Keep everything very short and succinct. The rating should be short as possible.", story.Title, story.URL)

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
			Title:   story.Title,
			URL:     story.URL,
			Summary: "[red]Invalid response format from Ollama[-]",
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

// Appends a message to the feed and handles "fading" by limiting line count
func appendMessage(view *tview.TextView, message string) {
	// Get current text and split by lines
	currentText := view.GetText(false)
	lines := strings.Split(currentText, "\n")

	// Limit to the last 20 lines for fading effect
	if len(lines) >= 20 {
		lines = lines[1:] // Remove the oldest line to simulate fading out
	}

	// Append the new message and update the view
	lines = append(lines, message)
	view.SetText(strings.Join(lines, "\n"))
}

