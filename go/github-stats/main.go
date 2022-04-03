package github_stats

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/go-github/v43/github"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Main(gitVersion string) func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		cfg, err := NewConfig(cliCtx)
		if err != nil {
			return err
		}

		if err := configureLogger(cfg); err != nil {
			return err
		}

		log.Info("configuring stats daemon", "version", gitVersion)

		ctx := context.Background()
		ts := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: cfg.AccessToken,
		})
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)

		go func() {
			addr := fmt.Sprintf("%s:%d", cfg.MetricsHostname, cfg.MetricsPort)
			log.Info("starting metrics server", "addr", addr)
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Crit("failed to start metrics server", "err", err)
			}
		}()

		log.Info("starting poller loop", "interval", cfg.PollInterval)
		repoPath := fmt.Sprintf("%s/%s", cfg.RepoOrg, cfg.RepoName)
		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)
		tick := time.NewTicker(cfg.PollInterval)
		for {
			log.Info("calculating stats", "repo", repoPath)
			start := time.Now()
			if err := CalculateStats(client, cfg.RepoOrg, cfg.RepoName); err != nil {
				log.Error("error calculating stats", "err", err)
			} else {
				log.Info("done calculating stats", "duration", time.Since(start))
			}
			select {
			case <-tick.C:
				continue
			case sig := <-sigC:
				log.Info("caught signal, shutting down", "signal", sig)
				os.Exit(0)
			}
		}
	}
}

func configureLogger(cfg Config) error {
	var logHandler log.Handler
	if cfg.LogTerminal {
		logHandler = log.StreamHandler(os.Stdout, log.TerminalFormat(true))
	} else {
		logHandler = log.StreamHandler(os.Stdout, log.JSONFormat())
	}

	logLevel, err := log.LvlFromString(cfg.LogLevel)
	if err != nil {
		return err
	}

	log.Root().SetHandler(log.LvlFilterHandler(logLevel, logHandler))
	return nil
}
