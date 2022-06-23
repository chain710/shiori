package background

import (
	"errors"
	"github.com/go-shiori/shiori/internal/core"
	"github.com/go-shiori/shiori/internal/database"
	"github.com/go-shiori/shiori/internal/model"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type AutoArchiveOptions struct {
	Concurrent           int
	ScanDelay, IdleDelay time.Duration
}

func (a *AutoArchiveOptions) Validate() error {
	if a.Concurrent <= 0 {
		return errors.New("concurrent <= 0")
	}
	if a.ScanDelay <= 0 {
		return errors.New("scan delay <= 0")
	}
	if a.IdleDelay <= 0 {
		return errors.New("idle delay <= 0")
	}

	return nil
}

func NewAutoArchive(db database.DB, dataDir string, opt AutoArchiveOptions) (*AutoArchive, error) {
	if err := opt.Validate(); err != nil {
		return nil, err
	}
	return &AutoArchive{
		dataDir:    dataDir,
		db:         db,
		concurrent: opt.Concurrent,
		scanDelay:  opt.ScanDelay,
		idleDelay:  opt.IdleDelay,
		stop:       make(chan struct{}),
	}, nil
}

type AutoArchive struct {
	dataDir    string
	db         database.DB
	concurrent int
	queue      chan model.Bookmark
	scanDelay  time.Duration // sleep after each scan
	idleDelay  time.Duration // sleep idleDelay if all bookmarks are archived
	stop       chan struct{}
	stopWait   sync.WaitGroup
}

func (a *AutoArchive) Start() {
	a.stopWait.Add(1)
	go func() {
		logrus.Info("scan worker started")
		a.scanWorker()
		a.stopWait.Done()
		logrus.Info("scan worker stopped")
	}()

	logrus.Info("auto archive started")
}

// Stop block until all goroutines exit
func (a *AutoArchive) Stop() {
	logrus.Info("auto archive stopping...")
	close(a.stop)
	a.stopWait.Wait()
	logrus.Info("auto archive stopped")
}

func (a *AutoArchive) spawnArchiveWorkers(queue <-chan *model.Bookmark, count int) *sync.WaitGroup {
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

func (a *AutoArchive) archiveWorker(queue <-chan *model.Bookmark) {
	for bookmark := range queue {
		a.archiveOnce(bookmark)
	}
}

func (a *AutoArchive) scanWorker() {
	for {
		select {
		case <-a.stop:
			return
		default:
			if count := a.scanOnce(); count == 0 {
				logrus.Debugf("scan nothing to archive")
				time.Sleep(a.idleDelay)
			} else {
				logrus.Debugf("scan complete: %d", count)
				time.Sleep(a.scanDelay)
			}
		}
	}

}

func (a *AutoArchive) scanOnce() int {
	hasArchive := false
	opts := database.GetBookmarksOptions{
		HasArchive: &hasArchive,
	}
	bookmarks, err := a.db.GetBookmarks(opts)
	if err != nil {
		logrus.Errorf("scan bookmarks error %s", err)
		return -1
	}

	count := len(bookmarks)
	if count == 0 {
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

func (a *AutoArchive) archiveOnce(bookmark *model.Bookmark) {
	log := logrus.WithFields(logrus.Fields{"id": bookmark.ID, "url": bookmark.URL})
	updatedBookmark, err := core.DownloadBookmarkContent(bookmark, a.dataDir)
	if err != nil {
		log.Errorf("download bookmark(%d) error: %s", bookmark.ID, err)
		return
	}

	// Save bookmark to database
	results, err := a.db.SaveBookmarks(*updatedBookmark)
	if err != nil || len(results) == 0 {
		log.Errorf("failed to save bookmark: %s", err)
		return
	}

	log.Infof("auto archive bookmark successfully")
}
