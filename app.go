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

func main() {
	// Define a flag for the search query
	searchPhrase := flag.String("query", "", "Search phrase to lookup images")
	flag.Parse()

	if *searchPhrase == "" {
		fmt.Println("Please provide a search phrase using -query flag.")
		return
	}

	// Create a folder to store images
	imageFolder := "./images"
	err := createFolder(imageFolder)
	if err != nil {
		log.Fatalf("Error creating folder: %v", err)
	}

	// Initialize the chromedp context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Fetch images from Yandex
	fmt.Println("Searching Yandex images...")
	yandexImages, err := searchYandexImages(ctx, *searchPhrase)
	if err != nil {
		log.Fatalf("Failed to search Yandex images: %v", err)
	}

	counter := 1
	for _, imageURL := range yandexImages {
		downloadImage(imageURL, imageFolder, *searchPhrase, counter, ".jpg")
		counter = counter + 1
	}

	// Fetch images from Google
	fmt.Println("Searching Google images...")
	googleImages, err := searchGoogleImages(ctx, *searchPhrase)
	if err != nil {
		log.Fatalf("Failed to search Google images: %v", err)
	}
	for _, imageURL := range googleImages {
		downloadImage(imageURL, imageFolder, *searchPhrase, counter, ".jpg")
		counter = counter + 1
	}
}
