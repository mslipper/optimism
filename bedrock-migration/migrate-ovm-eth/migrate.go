package migrator

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/crypto/sha3"
	"math/big"
	"path/filepath"
	"time"
)

var (
	OVMETHAddress = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")
	emptyCodeHash = crypto.Keccak256(nil)
)

var zeroHash common.Hash

func Migrate(dataDir string, stateRoot common.Hash, genesis *core.Genesis, outDir string) error {
	inDB, err := rawdb.NewLevelDBDatabase(
		filepath.Join(dataDir, "geth", "chaindata"),
		0,
		0,
		"",
		true,
	)
	if err != nil {
		log.Crit("error opening raw DB", "err", err)
	}

	inUnderlyingDB := state.NewDatabase(inDB)
	inStateDB, err := state.New(stateRoot, inUnderlyingDB, nil)
	if err != nil {
		log.Crit("error opening state db", "err", err)
	}

	outDB, err := rawdb.NewLevelDBDatabase(outDir, 0, 0, "", false)
	if err != nil {
		return err
	}

	log.Info("writing genesis data")
	if _, err := genesis.Commit(outDB); err != nil {
		log.Crit("error writing genesis data")
	}

	if stateRoot == zeroHash {
		stateRoot = getStateRoot(inDB)
	}

	outStateDB, err := state.New(common.Hash{}, state.NewDatabase(outDB), nil)
	if err != nil {
		return fmt.Errorf("error opening output state DB: %w", err)
	}

	log.Info("dumping state")
	dumpState(inUnderlyingDB, inStateDB, outStateDB, stateRoot, genesis)
	return nil
}

func dumpState(inDB state.Database, inStateDB *state.StateDB, outStateDB *state.StateDB, root common.Hash, genesis *core.Genesis) common.Hash {
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

	totalOVM := new(big.Int)
	it := trie.NewIterator(tr.NodeIterator(nil))
	for it.Next() {
		var data types.StateAccount
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			panic(err)
		}

		addrBytes := tr.GetKey(it.Key)

		addr := common.BytesToAddress(addrBytes)
		if _, ok := genesis.Alloc[addr]; ok {
			log.Info("skipping preallocated account", "addr", addr)
			continue
		}

		addrHash := crypto.Keccak256Hash(addr[:])
		code := getCode(addrHash, data, inDB)

		ovmBalance := getOVMETHBalance(inStateDB, addr)
		if data.Balance.Sign() > 0 {
			log.Crit("account has non-zero OVM eth balance", "addr", addr)
		}
		totalOVM = totalOVM.Add(totalOVM, ovmBalance)
		outStateDB.AddBalance(addr, ovmBalance)
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
		"elapsed", common.PrettyDuration(time.Since(start)), "total_ovm_eth", totalOVM)

	log.Info("committing state DB")
	newRoot, err := outStateDB.Commit(false)
	if err != nil {
		log.Crit("error writing output state DB", "err", err)
	}
	log.Info("committed state DB", "root", newRoot)
	log.Info("committing trie DB")
	if err := outStateDB.Database().TrieDB().Commit(newRoot, true, nil); err != nil {
		log.Crit("error writing output trie DB", "err", err)
	}
	log.Info("committed trie DB")

	return newRoot
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

func getStateRoot(db ethdb.Reader) common.Hash {
	block := rawdb.ReadHeadBlock(db)
	return block.Root()
}

func getOVMETHBalance(inStateDB *state.StateDB, addr common.Address) *big.Int {
	position := common.Big0
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(common.LeftPadBytes(addr.Bytes(), 32))
	hasher.Write(common.LeftPadBytes(position.Bytes(), 32))
	digest := hasher.Sum(nil)
	balKey := common.BytesToHash(digest)
	return inStateDB.GetState(OVMETHAddress, balKey).Big()
}
