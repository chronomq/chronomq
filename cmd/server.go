package cmd

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/chronomq/chronomq/pkg/chronomq"
	"github.com/chronomq/chronomq/pkg/persistence"
	"github.com/chronomq/chronomq/pkg/protocol"
)

var (
	// defaultAddrs holds the various endpoints we need to configure
	defaultAddrs = &addrs{
		rpcAddr:   ":11301",
		grpcAddr:  ":9999",
		statsAddr: ":8125",
	}
	// appCfg - wires in the application and configuration
	appCfg = &config{
		addrs: defaultAddrs,
	}
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Run server with a bucket data store",
		Long: `Persists jobs state in this s3 bucket/prefix when SIGUSR1 is received.
Restores from this location at start if journal files are present (and restore flag is set).
Supported s3-compatible storage backends: s3, gs, azblob and others: https://gocloud.dev/howto/blob/#s3-compatible`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Pre-run validates raw app config
			u, err := url.Parse(appCfg.rawStoreCfg.url)
			if err != nil {
				return err
			}
			switch u.Scheme {
			case "s3", "gs", "azblob", "file":
				// supported
			case "":
				// assume file
				u.Scheme = "file"
			default:
				return errors.New("Bucket scheme not supported")
			}
			q := u.Query()
			if !strings.HasSuffix(appCfg.rawStoreCfg.prefix, "/") {
				appCfg.rawStoreCfg.prefix = appCfg.rawStoreCfg.prefix + "/"
			}
			q.Add("prefix", appCfg.rawStoreCfg.prefix)
			u.RawQuery = q.Encode()
			appCfg.storeCfg.Bucket = u
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Int("PID", os.Getpid()).Msg("Starting Server")
			startApp(appCfg)
			log.Info().Msg("Shutdown ok")
		},
	}
)

// config wires in the application and configuration
type config struct {
	addrs       *addrs
	rawStoreCfg struct {
		url    string
		prefix string
	}

	storeCfg  persistence.StoreConfig // Persistence Storage config
	restore   bool                    // If true, hub will attempt restore on startup
	spokeSpan time.Duration           // Spoke duration
}

func init() {
	serverCmd.PersistentFlags().DurationVarP(&appCfg.spokeSpan, "spokeSpan", "S", time.Second*10, "Spoke span (golang duration string format)")
	serverCmd.PersistentFlags().BoolVarP(&appCfg.restore, "restore", "r", false, "Restore existing data if possible from store")
	dataDir, _ := os.Getwd()
	serverCmd.Flags().StringVar(&appCfg.rawStoreCfg.url, "store-url", dataDir, `Filesystem dir (default: PWD) or S3-style url.
Examples: filesystemdir/subdir or {file|s3|gs|azblob}://bucket`)
	serverCmd.Flags().StringVar(&appCfg.rawStoreCfg.prefix, "store-prefix", "", `Store path prefix`)

	rootCmd.AddCommand(serverCmd)
}

func startApp(cfg *config) {
	// More Aggressive GC
	if os.Getenv("GOGC") == "" {
		log.Info().Msg("Applying default GC tuning")
		debug.SetGCPercent(5)
	} else {
		log.Info().Str("GCPercent", os.Getenv("GOGC")).Msg("Using custom GC tuning")
	}
	go func() {
		err := http.ListenAndServe(":6060", nil)
		if err != nil {
			log.Error().Err(err).Msg("pprof server has stopped")
		}
	}()

	log.Info().Msg("Starting Chronomq")

	storage, err := cfg.storeCfg.Storage()
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot initialize storage")
	}

	opts := &chronomq.HubOpts{
		AttemptRestore: cfg.restore,
		SpokeSpan:      cfg.spokeSpan,
		Persister:      persistence.NewJournalPersister(storage),
		MaxCFSize:      chronomq.DefaultMaxCFSize,
	}

	h := chronomq.NewHub(opts)
	var rpcSRV io.Closer
	wg := sync.WaitGroup{}
	go func() {
		rpcSRV, _ = protocol.ServeRPC(h, cfg.addrs.rpcAddr)
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGUSR1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sigc
		log.Info().Msg("Stopping rpc protocol server")
		rpcSRV.Close()
		log.Info().Msg("Stopping rpc protocol server - Done")
		h.Stop(true)
	}()

	wg.Wait()
}
