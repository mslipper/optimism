package main

import (
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

var (
	OVMETHAddress    = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")
	Block7412000Root = common.HexToHash("0x5d4e7f7332568a6063a268db1bb518cbd5cd62e3f1933ee078a9c4a7c44b28c0")
	Zero             = new(big.Int)
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

	iter := db.NewIterator([]byte("addr-preimage-"), nil)
	for iter.Next() {
		if iter.Error() != nil {
			log.Crit("error in iterator", "err", iter.Error())
		}

		addrStr := hex.EncodeToString([]byte(strings.TrimPrefix(string(iter.Key()), "addr-preimage-")))
		addr := common.HexToAddress(addrStr)
		balKey := iter.Value()
		balKeyHash := common.BytesToHash(balKey)
		res := stateDB.GetState(OVMETHAddress, balKeyHash)
		ovmETHBal := res.Big()
		stateBal := stateDB.GetBalance(addr)
		if stateBal.Cmp(Zero) != 0 {
			log.Crit("found account with nonzero balance in state", "addr", addr, "state_bal", stateBal, "ovm_bal", ovmETHBal)
		}
		stateDB.SetState(OVMETHAddress, balKeyHash, common.Hash{})
		// don't bother with zero balances
		if ovmETHBal.Cmp(Zero) != 0 {
			stateDB.SetBalance(addr, ovmETHBal)
		}
		log.Info("migrated account", "addr", addr, "bal", ovmETHBal)
	}
	log.Info("writing trie modifications")
	//root, err := stateDB.Commit(true)
	//if err != nil {
	//	log.Crit("error writing trie", "err", err)
	//}
	//log.Info("successfully migrated trie", "root", root)
	enc := json.NewEncoder(os.Stdout)
	stateDB.IterativeDump(&state.DumpConfig{}, enc)
}
