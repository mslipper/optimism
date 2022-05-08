package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"os"
	"path/filepath"
)

var (
	OVMETHAddress = common.HexToAddress("0x4200000000000000000000000000000000000006")
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	args := os.Args

	if len(args) != 3 {
		log.Crit("must pass in an input and output file")
	}

	log.Info("starting migrator", "db_path", args[0])

	db, err := rawdb.NewLevelDBDatabaseWithFreezer(
		args[0],
		0,
		0,
		filepath.Join(args[0], "freezer"),
		"",
		true,
	)
	if err != nil {
		log.Crit("error opening DB", "err", err)
	}

	stateDB := state.NewDatabase(db)
	st, err := stateDB.OpenStorageTrie(OVMETHAddress.Hash(), common.Hash{})
	if err != nil {
		log.Crit("error opening storage trie", "err", err)
	}

	iter := st.NodeIterator(nil)

	for iter.Next(true) {
		if !iter.Leaf() {
			continue
		}

		fmt.Println(hex.EncodeToString(iter.LeafKey()))
		fmt.Println(hex.EncodeToString(iter.LeafBlob()))
	}
}
