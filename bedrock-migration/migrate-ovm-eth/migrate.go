package migrator

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"math/big"
	"path/filepath"
	"strings"
	"time"
)

var (
	OVMETHAddress    = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")
	Block7412000Root = common.HexToHash("0x5d4e7f7332568a6063a268db1bb518cbd5cd62e3f1933ee078a9c4a7c44b28c0")
	Zero             = new(big.Int)
	emptyCodeHash    = crypto.Keccak256(nil)
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

	underlyingDB := state.NewDatabase(db)
	stateDB, err := state.New(stateRoot, underlyingDB, nil)
	if err != nil {
		return fmt.Errorf("error opening state db: %w", err)
	}

	iter := db.NewIterator([]byte("addr-preimage-"), nil)
	var accounts uint64
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
		accounts++
	}
	log.Info("successfully migrated accounts", "count", accounts)
	log.Info("writing trie modifications")
	root, err := stateDB.Commit(true)
	if err != nil {
		log.Crit("error writing trie", "err", err)
	}
	log.Info("successfully migrated trie", "root", root)
	log.Info("dumping state")

	return DumpInMemory(underlyingDB, root, outFile)
}

func DumpInMemory(inDB state.Database, root common.Hash, outDir string) error {
	outDB, err := rawdb.NewLevelDBDatabase(outDir, 0, 0, "", false)
	if err != nil {
		return err
	}

	outStateDB, err := state.New(common.Hash{}, state.NewDatabase(outDB), nil)
	if err != nil {
		return err
	}

	tr, err := inDB.OpenTrie(root)
	if err != nil {
		panic(err)
	}

	var (
		accounts     uint64
		lastAccounts uint64
		start        = time.Now()
		logged       = time.Now()
	)
	log.Info("Trie dumping started", "root", tr.Hash())

	it := trie.NewIterator(tr.NodeIterator(nil))
	for it.Next() {
		var data types.StateAccount
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			panic(err)
		}

		addrBytes := tr.GetKey(it.Key)
		addr := common.BytesToAddress(addrBytes)
		addrHash := crypto.Keccak256Hash(addr[:])
		code := getCode(addrHash, data, inDB)

		outStateDB.AddBalance(addr, data.Balance)
		outStateDB.SetCode(addr, code)
		outStateDB.SetNonce(addr, data.Nonce)

		storageTrie, err := inDB.OpenStorageTrie(addrHash, data.Root)
		if err != nil {
			panic(err)
		}
		storageIt := trie.NewIterator(storageTrie.NodeIterator(nil))
		var storageSlots uint64
		storageLogged := time.Now()
		for storageIt.Next() {
			storageSlots++
			_, content, _, err := rlp.Split(storageIt.Value)
			if err != nil {
				panic(err)
			}
			outStateDB.SetState(
				addr,
				common.BytesToHash(storageTrie.GetKey(storageIt.Key)),
				common.BytesToHash(content),
			)
			if time.Since(storageLogged) > 8*time.Second {
				since := time.Since(start)
				log.Info("Storage dumping in progress", "addr", addr, "storage_slots", storageSlots,
					"elapsed", common.PrettyDuration(since))
				storageLogged = time.Now()
			}
		}

		accounts++
		if time.Since(logged) > 8*time.Second {
			since := time.Since(start)
			rate := float64(accounts-lastAccounts) / float64(since/time.Second)

			log.Info("Trie dumping in progress", "at", it.Key, "accounts", accounts,
				"elapsed", common.PrettyDuration(since), "accs_per_s", rate)
			logged = time.Now()
			lastAccounts = accounts
		}
	}
	log.Info("Trie dumping complete", "accounts", accounts,
		"elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

func getCode(addrHash common.Hash, data types.StateAccount, db state.Database) []byte {
	if bytes.Equal(data.CodeHash, emptyCodeHash) {
		return nil
	}

	code, err := db.ContractCode(
		addrHash,
		common.BytesToHash(data.CodeHash),
	)
	if err != nil {
		panic(err)
	}
	return code
}
