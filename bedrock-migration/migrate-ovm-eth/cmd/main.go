package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mattn/go-isatty"
	"golang.org/x/crypto/sha3"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

var (
	OVMETHAddress    = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")
	Block7412000Root = common.HexToHash("0x5d4e7f7332568a6063a268db1bb518cbd5cd62e3f1933ee078a9c4a7c44b28c0")
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

	stateDB, err := state.New(Block7412000Root, state.NewDatabase(db), nil)
	if err != nil {
		log.Crit("error opening state db", "err", err)
	}
	st := stateDB.StorageTrie(OVMETHAddress)
	if st == nil {
		log.Crit("storage trie is nil", "address", OVMETHAddress)
	}
	log.Info("opened storage trie")

	iter := db.NewIterator([]byte("addr-preimage-"), nil)
	for iter.Next() {
		if iter.Error() != nil {
			log.Crit("error in iterator", "err", iter.Error())
		}

		addr := hex.EncodeToString([]byte(strings.TrimPrefix(string(iter.Key()), "addr-preimage-")))
		balKey := iter.Value()
		res, err := st.TryGet(balKey[:])
		if err != nil {
			log.Crit("error reading storage trie", "err", err)
		}
		if len(res) > 0 {
			log.Warn("address has zero length state", "addr", addr)
			continue
		}
		_, balBytes, _, err := rlp.Split(res)
		if err != nil {
			log.Crit("error decoding storage trie value", "err", err)
		}
		fmt.Printf("%s,%s\n", addr, new(big.Int).SetBytes(balBytes).String())
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