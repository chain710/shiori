package core

import (
	"fmt"
	"github.com/go-shiori/shiori/internal/model"
	"io"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: time.Minute}

// DownloadBookmark downloads bookmarked page from specified URL.
// Return response body, make sure to close it later.
func DownloadBookmark(url string) (io.ReadCloser, string, error) {
	// Prepare download request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	// Send download request
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}

	// Get content type
	contentType := resp.Header.Get("Content-Type")

	return resp.Body, contentType, nil
}

func DownloadBookmarkContent(book *model.Bookmark, dataDir string) (*model.Bookmark, error) {
	content, contentType, err := DownloadBookmark(book.URL)
	if err != nil {
		return nil, fmt.Errorf("error downloading bookmark: %s", err)
	}

	processRequest := ProcessRequest{
		DataDir:     dataDir,
		Bookmark:    *book,
		Content:     content,
		ContentType: contentType,
	}
	processRequest.Bookmark.CreateArchive = true
	result, isFatalErr, err := ProcessBookmark(processRequest)
	_ = content.Close()

	if err != nil && isFatalErr {
		panic(fmt.Errorf("failed to process bookmark: %v", err))
	}

	return &result, err
}
