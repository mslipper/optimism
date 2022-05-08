package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"os"
	"path/filepath"
)

var (
	OVMETHAddress = common.HexToAddress("0x4200000000000000000000000000000000000006")
	Block7412400Root = common.HexToHash("0xd4a9d6b2446a3153caaa4189c327def0673f85609c2c277befa0860b68b8d6cd")
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	args := os.Args

	if len(args) != 3 {
		log.Crit("must pass in an input and output file")
	}

	log.Info("starting migrator", "db_path", args[1])

	db, err := rawdb.NewLevelDBDatabase(
		filepath.Join(args[1], "geth", "chaindata"),
		0,
		0,
		"",
		true,
	)
	if err != nil {
		log.Crit("error opening DB", "err", err)
	}

	stateDB := state.NewDatabase(db)

	accountDB, err := stateDB.OpenTrie(Block7412400Root)
	if err != nil {
		log.Crit("error opening account trie", "err", err)
	}
	entry, err := accountDB.TryGet(crypto.Keccak256Hash(OVMETHAddress[:]).Bytes())
	if err != nil {
		log.Crit("error readying account state", "err", err)
	}
	fmt.Println(entry)

	st, err := stateDB.OpenStorageTrie(OVMETHAddress.Hash(), common.Hash{})
	if err != nil {
		log.Crit("error opening storage trie", "err", err)
	}



	log.Info("opened storage trie")

	iter := st.NodeIterator(nil)

	for iter.Next(true) {
		if !iter.Leaf() {
			continue
		}

		fmt.Println(hex.EncodeToString(iter.LeafKey()))
		fmt.Println(hex.EncodeToString(iter.LeafBlob()))
	}
}
