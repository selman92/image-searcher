# Image Search CLI Tool

A command-line interface tool for searching and downloading images from multiple search engines, including Google, Bing, and Yandex. The tool leverages the `chromedp` library to interact with search engine results pages and download images efficiently.

## Features

- Supports multiple search targets: Google, Bing, Yandex.
- Download images concurrently from selected search engines.
- Scroll through search results to fetch more images.
- Save images in organized folders based on the search engine.
- Customize output directory and search parameters.

## Installation

1. Ensure you have Go installed. If you haven't installed it yet, you can download it from [the official Go website](https://golang.org/dl/).

2. Clone the repository:
   ```bash
   git clone https://github.com/selman92/image-searcher
   cd image-searcher

3. Install required dependencies:
     ```bash
    go get github.com/chromedp/chromedp
    go get github.com/schollz/progressbar/v3

## Usage
   ```bash
    go run app.go [flags]
   ```

## Flags

* `-query`, `-q`: (Required) Search query for images.
* `-targets`, `-t`: (Optional) Comma-separated search targets: google, bing, yandex, or all (default: all).
* `-out`, `-o`: (Optional) Directory to save images (default: images).
* `-log`, `-l`: (Optional) File to save error logs (default: error.log).

## Example Usages

1. Basic search with default settings:
    ```bash
    go run app.go -query "cats"

2. Search images using specific search engines:
    ```bash
    go run app.go -query "cats" -targets "google,bing"

3. With all arguments
    ```bash
    go run app.go -q "cats" -t "google,bing,yandex" -log "my_log.txt" -o "img/"


