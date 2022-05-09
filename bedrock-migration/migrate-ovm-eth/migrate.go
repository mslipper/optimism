package migrator

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
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

func Migrate(dataDir string, stateRoot common.Hash, outFile string) error {
	db, err := rawdb.NewLevelDBDatabase(
		filepath.Join(dataDir, "geth", "chaindata"),
		0,
		0,
		"",
		true,
	)
	if err != nil {
		return fmt.Errorf("error opening DB: %w", err)
	}

	stateDB, err := state.New(stateRoot, state.NewDatabase(db), nil)
	if err != nil {
		return fmt.Errorf("error opening state db: %w", err)
	}

	iter := db.NewIterator([]byte("addr-preimage-"), nil)
	var i int
	for iter.Next() {
		if iter.Error() != nil {
			return fmt.Errorf("error in iterator: %w", iter.Error())
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
		i++
	}
	log.Info("successfully migrated accounts", "count", i)
	log.Info("writing trie modifications")
	root, err := stateDB.Commit(true)
	if err != nil {
		log.Crit("error writing trie", "err", err)
	}
	log.Info("successfully migrated trie", "root", root)
	log.Info("dumping state")

	f, err := os.OpenFile(outFile, os.O_CREATE | os.O_TRUNC | os.O_RDWR, 0o777)
	if err != nil {
		return fmt.Errorf("error opening dump file: %w", err)
	}
	dumper := &AllocDumper{
		w: f,
	}
	stateDB.DumpToCollector(dumper, &state.DumpConfig{})
	return nil
}