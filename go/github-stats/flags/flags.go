package flags

import (
	"github.com/urfave/cli"
	"time"
)

const envVarPrefix = "GH_STATS"

func prefixEnvVar(name string) string {
	return envVarPrefix + name
}

var (
	AccessTokenFlag = cli.StringFlag{
		Name:     "access-token",
		Usage:    "GitHub personal access token to authenticate with the GitHub API",
		Required: true,
		EnvVar:   prefixEnvVar("ACCESS_TOKEN"),
	}
	RepoOrgFlag = cli.StringFlag{
		Name:   "repo-org",
		Usage:  "GitHub organization that the target repo belongs to",
		Value:  "ethereum-optimism",
		EnvVar: prefixEnvVar("REPO_ORG"),
	}
	RepoNameFlag = cli.StringFlag{
		Name:   "repo-name",
		Usage:  "Name of the target GitHub repo",
		Value:  "optimism",
		EnvVar: prefixEnvVar("REPO_NAME"),
	}
	PollIntervalFlag = cli.DurationFlag{
		Name:   "poll-interval",
		Usage:  "How often to poll GitHub for new stats.",
		Value:  5 * time.Minute,
		EnvVar: prefixEnvVar("POLL_INTERVAL"),
	}
	MetricsHostnameFlag = cli.StringFlag{
		Name:   "metrics-hostname",
		Usage:  "The hostname of the metrics server",
		Value:  "127.0.0.1",
		EnvVar: prefixEnvVar("METRICS_HOSTNAME"),
	}
	MetricsPortFlag = cli.Uint64Flag{
		Name:   "metrics-port",
		Usage:  "The port of the metrics server",
		Value:  7300,
		EnvVar: prefixEnvVar("METRICS_PORT"),
	}
	LogLevelFlag = cli.StringFlag{
		Name:   "log-level",
		Usage:  "The lowest log level that will be output",
		Value:  "info",
		EnvVar: prefixEnvVar("LOG_LEVEL"),
	}
	LogTerminalFlag = cli.BoolFlag{
		Name: "log-terminal",
		Usage: "If true, outputs logs in terminal format, otherwise prints " +
			"in JSON format. If SENTRY_ENABLE is set to true, this flag is " +
			"ignored and logs are printed using JSON",
		EnvVar: prefixEnvVar("LOG_TERMINAL"),
	}
)

var requiredFlags = []cli.Flag{
	AccessTokenFlag,
}

var optionalFlags = []cli.Flag{
	RepoOrgFlag,
	RepoNameFlag,
	PollIntervalFlag,
	MetricsHostnameFlag,
	MetricsPortFlag,
	LogLevelFlag,
	LogTerminalFlag,
}

var Flags = append(requiredFlags, optionalFlags...)
