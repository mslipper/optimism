package github_stats

import (
	"github.com/ethereum-optimism/optimism/go/github-stats/flags"
	"github.com/urfave/cli"
	"time"
)

type Config struct {
	AccessToken     string
	RepoOrg         string
	RepoName        string
	PollInterval    time.Duration
	MetricsHostname string
	MetricsPort     uint64
	LogLevel        string
	LogTerminal     bool
}

func NewConfig(ctx *cli.Context) (Config, error) {
	return Config{
		AccessToken:     ctx.GlobalString(flags.AccessTokenFlag.Name),
		RepoOrg:         ctx.GlobalString(flags.RepoOrgFlag.Name),
		RepoName:        ctx.GlobalString(flags.RepoNameFlag.Name),
		PollInterval:    ctx.GlobalDuration(flags.PollIntervalFlag.Name),
		MetricsHostname: ctx.GlobalString(flags.MetricsHostnameFlag.Name),
		MetricsPort:     ctx.GlobalUint64(flags.MetricsPortFlag.Name),
		LogLevel:        ctx.GlobalString(flags.LogLevelFlag.Name),
		LogTerminal:     ctx.GlobalBool(flags.LogTerminalFlag.Name),
	}, nil
}
