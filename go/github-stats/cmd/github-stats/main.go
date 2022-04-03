package main

import (
	"fmt"
	github_stats "github.com/ethereum-optimism/optimism/go/github-stats"
	"github.com/ethereum-optimism/optimism/go/github-stats/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
	"os"
)

var (
	GitVersion = ""
	GitCommit  = ""
	GitDate    = ""
)

func main() {
	// Set up logger with a default INFO level in case we fail to parse flags.
	// Otherwise the final critical log won't show what the parsing error was.
	log.Root().SetHandler(
		log.LvlFilterHandler(
			log.LvlInfo,
			log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
		),
	)

	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Version = fmt.Sprintf("%s-%s-%s", GitVersion, GitCommit, GitDate)
	app.Name = "github-stats"
	app.Usage = "GitHub stats daemon"
	app.Description = "Returns data about GitHub so we can put it on Grafana"

	app.Action = github_stats.Main(GitVersion)
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
