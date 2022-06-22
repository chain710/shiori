package core

import (
	"context"
	"errors"
	"github.com/go-shiori/shiori/internal/database"
	"github.com/go-shiori/shiori/internal/model"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type BackgroundArchiverOptions struct {
	Concurrent   int
	ScanInterval time.Duration
}

func (o *BackgroundArchiverOptions) Validate() error {
	if o.Concurrent <= 0 {
		return errors.New("concurrent <= 0")
	}
	if o.ScanInterval <= 0 {
		return errors.New("scan delay <= 0")
	}

	return nil
}

func NewBackgroundArchiver(db database.DB, dataDir string, opt BackgroundArchiverOptions) (*BackgroundArchiver, error) {
	if err := opt.Validate(); err != nil {
		return nil, err
	}
	return &BackgroundArchiver{
		dataDir:      dataDir,
		db:           db,
		concurrent:   opt.Concurrent,
		scanInterval: opt.ScanInterval,
		stop:         make(chan struct{}),
		notify:       make(chan struct{}, 1),
	}, nil
}

type ArchiverNotifier interface {
	Notify()
}

var _ ArchiverNotifier = &BackgroundArchiver{}

type BackgroundArchiver struct {
	dataDir      string
	db           database.DB
	concurrent   int
	queue        chan model.Bookmark
	scanInterval time.Duration // sleep after each scan
	stop         chan struct{}
	stopWait     sync.WaitGroup
	notify       chan struct{}
}

func (a *BackgroundArchiver) Start() {
	a.stopWait.Add(1)
	go func() {
		logrus.Info("scan worker started")
		a.scanWorker()
		a.stopWait.Done()
		logrus.Info("scan worker stopped")
	}()

	logrus.Info("background archiver started")
}

func (a *BackgroundArchiver) Notify() {
	select {
	case a.notify <- struct{}{}:
		logrus.Debugf("notify background archiver")
	default:
		// do nothing
	}
}

// Stop block until all goroutines exit
func (a *BackgroundArchiver) Stop() {
	logrus.Info("background archiver stopping...")
	close(a.stop)
	a.stopWait.Wait()
	logrus.Info("background archiver stopped")
}

func (a *BackgroundArchiver) spawnArchiveWorkers(queue <-chan *model.Bookmark, count int) *sync.WaitGroup {
	if count > a.concurrent {
		count = a.concurrent
	}

	var stopWait sync.WaitGroup
	stopWait.Add(count)
	for i := 0; i < count; i++ {
		go func(idx int) {
			logrus.Debugf("archive worker[%d] started", idx)
			a.archiveWorker(queue)
			stopWait.Done()
			logrus.Debugf("archive worker[%d] stopped", idx)
		}(i)
	}

	return &stopWait
}

func (a *BackgroundArchiver) archiveWorker(queue <-chan *model.Bookmark) {
	for bookmark := range queue {
		a.archiveOnce(bookmark)
	}
}

func (a *BackgroundArchiver) scanWorker() {
	ticker := time.NewTicker(a.scanInterval)
	for {
		select {
		case <-a.stop:
			return
		case <-a.notify:
			logrus.Debugf("scan wake up by notification")
			a.scanOnce()
		case <-ticker.C:
			logrus.Debugf("scan wake up by ticker")
			a.scanOnce()
		}
	}

}

func (a *BackgroundArchiver) scanOnce() int {
	hasArchive := false
	opts := database.GetBookmarksOptions{
		HasArchive: &hasArchive,
	}
	bookmarks, err := a.db.GetBookmarks(context.TODO(), opts)
	if err != nil {
		logrus.Errorf("scan bookmarks error %s", err)
		return -1
	}

	count := len(bookmarks)
	if count == 0 {
		logrus.Debugf("scan nothing")
		return 0
	}

	queue := make(chan *model.Bookmark, count)
	stopWait := a.spawnArchiveWorkers(queue, count)

	for i := range bookmarks {
		logrus.Debugf("scan bookmark %d", bookmarks[i].ID)
		queue <- &bookmarks[i]
	}

	close(queue)
	stopWait.Wait()
	return count
}

func (a *BackgroundArchiver) archiveOnce(bookmark *model.Bookmark) {
	log := logrus.WithFields(logrus.Fields{"id": bookmark.ID, "url": bookmark.URL})
	bookmark.CreateArchive = true
	updatedBookmark, err := DownloadBookmarkContent(bookmark, a.dataDir)
	if err != nil {
		log.Errorf("download bookmark(%d) error: %s", bookmark.ID, err)
		return
	}

	// Save bookmark to database
	results, err := a.db.SaveBookmarks(context.TODO(), false, *updatedBookmark)
	if err != nil || len(results) == 0 {
		log.Errorf("failed to save bookmark: %s", err)
		return
	}

	log.Infof("archive bookmark successfully")
}
