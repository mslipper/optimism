package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
	migrator "migrate-ovm-eth"
	"os"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "migrate",
		Usage: "migrates data from v1 to Bedrock",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "data-dir",
				Aliases:  []string{"d"},
				Usage:    "data directory to read",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "state-root",
				Aliases: []string{"r"},
				Usage:   "state root to dump",
				Value:   "0x5d4e7f7332568a6063a268db1bb518cbd5cd62e3f1933ee078a9c4a7c44b28c0",
			},
			&cli.StringFlag{
				Name:     "out-file",
				Aliases:  []string{"o"},
				Usage:    "path to output file",
				Required: true,
			},
		},
		Action: action,
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}

func action(cliCtx *cli.Context) error {
	dataDir := cliCtx.String("data-dir")
	stateRoot := cliCtx.String("state-root")
	outFile := cliCtx.String("out-file")
	stateRootHash := common.HexToHash(stateRoot)
	return migrator.Migrate(dataDir, stateRootHash, outFile)
}
