package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// CreateFolder creates the directory to save images
func createFolder(folder string) error {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err := os.Mkdir(folder, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create folder: %v", err)
		}
	}
	return nil
}

// DownloadImage downloads the image from the given URL to the specified folder with a sequential name
func downloadImage(url, folder, query string, counter int, extension string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()

	// Create a file with sequential name
	fileName := filepath.Join(folder, fmt.Sprintf("%s%d%s", query, counter, extension))
	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save image: %v", err)
	}

	fmt.Printf("Downloaded %s\n", fileName)
	return nil
}

// SearchYandexImages searches for images on Yandex using chromedp and returns the image URLs
func searchYandexImages(ctx context.Context, query string) ([]string, error) {
	var links []string
	searchURL := fmt.Sprintf("https://yandex.com/images/search?text=%s", strings.Replace(query, " ", "+", -1))

	// Run tasks to load the Yandex image search page and extract image URLs from <a> tags
	err := chromedp.Run(ctx,
		// Navigate to Yandex image search
		chromedp.Navigate(searchURL),
		chromedp.Sleep(2*time.Second), // Wait for the page to load

		// Extract href attributes from <a> tags with class "Link ContentImage-Cover"
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a.Link.ContentImage-Cover')).map(a => a.href)`, &links),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Yandex image links: %v", err)
	}

	// Parse img_url parameter from the href attribute to get the actual image URLs
	imageURLs := parseYandexImageURLs(links)

	return imageURLs, nil
}

// Parse img_url parameter from the Yandex href to extract the actual image URLs
func parseYandexImageURLs(links []string) []string {
	var imageURLs []string
	for _, link := range links {
		// Parse the href to extract the img_url query parameter
		u, err := url.Parse(link)
		if err != nil {
			continue
		}
		// Extract img_url parameter from the href
		imgURL := u.Query().Get("img_url")
		if imgURL != "" {
			imageURLs = append(imageURLs, imgURL)
		}
	}
	return imageURLs
}

// SearchGoogleImages searches for images on Google using chromedp and returns the image URLs
func searchGoogleImages(ctx context.Context, query string) ([]string, error) {
	var imageURLs []string
	searchURL := fmt.Sprintf("https://www.google.com/search?q=%s&tbm=isch&udm=2", strings.Replace(query, " ", "+", -1))

	// Run tasks to load the Google image search page, scroll, and extract full-size image URLs
	err := chromedp.Run(ctx,
		// Navigate to Google image search
		chromedp.Navigate(searchURL),
		chromedp.Sleep(2*time.Second), // Wait for the page to load

		// Scroll down to load more images (simulate user interaction)
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i := 0; i < 10; i++ { // Scroll multiple times to load more images
				err := chromedp.Run(ctx, chromedp.Evaluate(`window.scrollBy(0, document.body.scrollHeight);`, nil))
				if err != nil {
					return err
				}
				time.Sleep(500 * time.Millisecond) // Wait for images to load after each scroll
			}
			return nil
		}),

		// Wait for additional images to load
		chromedp.Sleep(2*time.Second),

		// Extract full-size image URLs from the page (use 'src' from 'img' elements)
		chromedp.Evaluate(`Array.from(document.querySelectorAll('img')).map(img => img.src)`, &imageURLs),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google images: %v", err)
	}

	// Filter out irrelevant images (Google logos, base64 images, favicon images, etc.)
	filteredImageURLs := filterGoogleImageURLs(imageURLs)
	return filteredImageURLs, nil
}

// Filter out irrelevant Google image URLs (like Google logos, base64 images, and favicon images)
func filterGoogleImageURLs(imageURLs []string) []string {
	var filtered []string
	for _, url := range imageURLs {
		// Filter out small icons, base64 images, favicon images, and irrelevant URLs
		if strings.HasPrefix(url, "https") && !strings.Contains(url, "google") && !strings.Contains(url, "base64") && !strings.Contains(url, "FAVICON") {
			filtered = append(filtered, url)
		}
	}
	return filtered
}

// SearchBingImages searches for images on Bing using chromedp and returns the image URLs
func searchBingImages(ctx context.Context, query string) ([]string, error) {
	var imageURLs []string
	searchURL := fmt.Sprintf("https://www.bing.com/images/search?q=%s", strings.Replace(query, " ", "+", -1))

	// Run tasks to load the Bing image search page and extract image URLs
	err := chromedp.Run(ctx,
		// Navigate to Bing image search
		chromedp.Navigate(searchURL),
		chromedp.Sleep(2*time.Second), // Wait for the page to load

		// Scroll down to load more images (simulate user interaction)
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i := 0; i < 5; i++ { // Scroll multiple times to load more images
				err := chromedp.Run(ctx, chromedp.Evaluate(`window.scrollBy(0, document.body.scrollHeight);`, nil))
				if err != nil {
					return err
				}
				time.Sleep(500 * time.Millisecond) // Wait for images to load after each scroll
			}
			return nil
		}),

		chromedp.Evaluate(`Array.from(document.querySelectorAll('a.iusc')).map(a => a.getAttribute('m')).map(json => JSON.parse(json).murl)`, &imageURLs),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Bing images: %v", err)
	}

	return imageURLs, nil
}

// SaveImagesConcurrently saves images concurrently from a list of URLs to the specified folder
func downloadImages(imageURLs []string, folder, query string) {
	// Create the folder if it doesn't exist
	err := os.MkdirAll(folder, os.ModePerm)
	if err != nil {
		fmt.Printf("Failed to create folder: %v\n", err)
		return
	}

	// Set up a wait group to download images concurrently
	var wg sync.WaitGroup
	for i, url := range imageURLs {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			// Append .jpg extension to all downloaded images
			err := downloadImage(url, folder, query, i+1, ".jpg")
			if err != nil {
				fmt.Printf("Failed to download image %d: %v\n", i+1, err)
			}
		}(i, url)
	}

	// Wait for all download tasks to complete
	wg.Wait()
}

func defineStringFlag(longName string, shortName string, defaultValue string, usage string) *string {
	val := flag.String(longName, defaultValue, usage)
	flag.StringVar(val, shortName, defaultValue, usage)
	return val
}

func main() {
	// Parse CLI arguments
	query := defineStringFlag("query", "q", "", "Search query for images (required)")
	targets := defineStringFlag("targets", "t", "all", "Comma-separated search targets: google, bing, yandex, or all (default: all)")
	out := defineStringFlag("out", "o", "images", "Directory to save images (default: images)")

	flag.Parse()

	// Validate query input
	if *query == "" {
		log.Fatal("Please provide a search query using the -query flag.")
	}

	// Set up search targets
	var searchTargets []string
	if *targets == "all" {
		searchTargets = []string{"google", "bing", "yandex"}
	} else {
		searchTargets = strings.Split(*targets, ",")
		for i := range searchTargets {
			searchTargets[i] = strings.TrimSpace(searchTargets[i]) // Trim whitespace
		}
	}

	// Set up a wait group to handle concurrency across search engines
	var wg sync.WaitGroup

	// Iterate over the search targets and run each search concurrently
	for _, target := range searchTargets {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()

			fmt.Printf("Searching on %s...\n", target)

			// Create a new context and ChromeDP instance for this search
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// Start a new ChromeDP instance
			opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", true))
			allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
			defer cancelAlloc()

			// Create a new ChromeDP context
			taskCtx, cancelTask := chromedp.NewContext(allocCtx)
			defer cancelTask()

			switch target {
			case "google":
				googleImages, err := searchGoogleImages(taskCtx, *query)
				if err == nil {
					downloadImages(googleImages, filepath.Join(*out, "google"), *query)
				} else {
					log.Printf("Failed to search on Google: %v\n", err)
				}
			case "bing":
				bingImages, err := searchBingImages(taskCtx, *query)
				if err == nil {
					downloadImages(bingImages, filepath.Join(*out, "bing"), *query)
				} else {
					log.Printf("Failed to search on Bing: %v\n", err)
				}
			case "yandex":
				yandexImages, err := searchYandexImages(taskCtx, *query)
				if err == nil {
					downloadImages(yandexImages, filepath.Join(*out, "yandex"), *query)
				} else {
					log.Printf("Failed to search on Yandex: %v\n", err)
				}
			default:
				log.Printf("Unknown search target: %s\n", target)
			}
		}(target)
	}

	// Wait for all search engine tasks to complete
	wg.Wait()
	fmt.Println("Image search and download completed.")
}
