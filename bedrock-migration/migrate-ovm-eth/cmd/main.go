package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"golang.org/x/crypto/sha3"
	"os"
	"path/filepath"
)

var (
	OVMETHAddress = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")
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

	stateDB, err := state.New(Block7412400Root, state.NewDatabase(db), nil)
	if err != nil {
		log.Crit("error opening state db", "err", err)
	}
	st := stateDB.StorageTrie(OVMETHAddress)
	if st == nil {
		log.Crit("storage trie is nil", "address", OVMETHAddress)
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

func GetOVMBalanceKey(addr common.Address) common.Hash {
	position := common.Big0
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(common.LeftPadBytes(addr.Bytes(), 32))
	hasher.Write(common.LeftPadBytes(position.Bytes(), 32))
	digest := hasher.Sum(nil)
	return common.BytesToHash(digest)
}