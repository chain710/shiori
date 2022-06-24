package cmd

import (
	"github.com/go-shiori/shiori/internal/background"
	"strings"
	"time"

	"github.com/go-shiori/shiori/internal/webserver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve web interface for managing bookmarks",
		Long: "Run a simple and performant web server which " +
			"serves the site for managing bookmarks. If --port " +
			"flag is not used, it will use port 8080 by default.",
		Run: serveHandler,
	}

	cmd.Flags().IntP("port", "p", 8080, "Port used by the server")
	cmd.Flags().StringP("address", "a", "", "Address the server listens to")
	cmd.Flags().StringP("webroot", "r", "/", "Root path that used by server")
	cmd.Flags().Bool("log", true, "Print out a non-standard access log")
	cmd.Flags().Bool("auto-archive", false, "enable auto archive")
	cmd.Flags().Int("auto-archive-concurrent", 2, "num of auto archive workers")
	cmd.Flags().Duration("auto-archive-scan-interval", time.Minute, "delay after each scan")

	return cmd
}

func serveHandler(cmd *cobra.Command, args []string) {
	// Get flags value
	port, _ := cmd.Flags().GetInt("port")
	address, _ := cmd.Flags().GetString("address")
	rootPath, _ := cmd.Flags().GetString("webroot")
	log, _ := cmd.Flags().GetBool("log")
	autoArchive, _ := cmd.Flags().GetBool("auto-archive")
	autoArchiveConcurrent, _ := cmd.Flags().GetInt("auto-archive-concurrent")
	scanInterval, _ := cmd.Flags().GetDuration("auto-archive-scan-interval")

	// Validate root path
	if rootPath == "" {
		rootPath = "/"
	}

	if !strings.HasPrefix(rootPath, "/") {
		rootPath = "/" + rootPath
	}

	if !strings.HasSuffix(rootPath, "/") {
		rootPath += "/"
	}

	// Start server
	serverConfig := webserver.Config{
		DB:            db,
		DataDir:       dataDir,
		ServerAddress: address,
		ServerPort:    port,
		RootPath:      rootPath,
		Log:           log,
	}

	if autoArchive {
		serverConfig.DisableDownloadContentInAPI = true
		opt := background.AutoArchiveOptions{
			Concurrent:   autoArchiveConcurrent,
			ScanInterval: scanInterval,
		}
		aa, err := background.NewAutoArchive(db, dataDir, opt)
		if err != nil {
			logrus.Errorf("create auto archive error: %\n", err)
			return
		}

		serverConfig.Notify = aa
		aa.Start()

		defer func() {
			aa.Stop()
		}()
	} else {
		logrus.Info("auto archive NOT enabled")
	}

	err := webserver.ServeApp(serverConfig)
	if err != nil {
		logrus.Fatalf("Server error: %v\n", err)
	}
}
