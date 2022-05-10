package migrator

import (
	"bufio"
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
	"os"
	"path/filepath"
	"strconv"
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

	f, err := os.OpenFile(outFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o777)
	if err != nil {
		return fmt.Errorf("error opening dump file: %w", err)
	}
	builder := NewJSONBuilder(bufio.NewWriter(f))
	return CustomDump(root, underlyingDB, builder)
}

func CustomDump(root common.Hash, db state.Database, builder *JSONBuilder) error {
	tr, err := db.OpenTrie(root)
	if err != nil {
		panic(err)
	}

	var (
		accounts uint64
		start    = time.Now()
		logged   = time.Now()
	)
	log.Info("Trie dumping started", "root", tr.Hash())

	if err := builder.Begin(); err != nil {
		return err
	}

	it := trie.NewIterator(tr.NodeIterator(nil))
	for it.Next() {
		if accounts > 0 {
			if err := builder.Next(true); err != nil {
				return err
			}
		}

		var data types.StateAccount
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			panic(err)
		}

		addrBytes := tr.GetKey(it.Key)
		addr := common.BytesToAddress(addrBytes)
		addrHash := crypto.Keccak256Hash(addr[:])
		code := getCode(addrHash, data, db)

		if err := builder.Enter(hex.EncodeToString(addr.Bytes())); err != nil {
			return err
		}

		isFirstStorage := true
		var hasStorage bool
		storageTrie, err := db.OpenStorageTrie(addrHash, data.Root)
		if err != nil {
			panic(err)
		}
		storageIt := trie.NewIterator(storageTrie.NodeIterator(nil))
		for storageIt.Next() {
			if isFirstStorage {
				if err := builder.Enter("storage"); err != nil {
					return err
				}
			} else {
				if err := builder.Next(true); err != nil {
					return err
				}
			}
			_, content, _, err := rlp.Split(storageIt.Value)
			if err != nil {
				log.Error("Failed to decode the value returned by iterator", "error", err)
				continue
			}
			if err := builder.SetString(
				common.Bytes2Hex(storageTrie.GetKey(storageIt.Key)),
				common.Bytes2Hex(content),
			); err != nil {
				return err
			}
			isFirstStorage = false
			hasStorage = true
		}
		if hasStorage {
			if err := builder.Next(false); err != nil {
				return err
			}
			if err := builder.Leave(); err != nil {
				return err
			}
			if err := builder.Next(true); err != nil {
				return err
			}
		}

		if code != "" {
			if err := builder.SetString("code", code); err != nil {
				return err
			}
			if err := builder.Next(true); err != nil {
				return err
			}
		}
		if err := builder.SetString("nonce", strconv.FormatUint(data.Nonce, 10)); err != nil {
			return err
		}
		if err := builder.Next(true); err != nil {
			return err
		}
		if err := builder.SetString("balance", "0x"+data.Balance.Text(16)); err != nil {
			return err
		}
		if err := builder.Next(true); err != nil {
			return err
		}

		accounts++
		if time.Since(logged) > 8*time.Second {
			log.Info("Trie dumping in progress", "at", it.Key, "accounts", accounts,
				"elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}

		if err := builder.Leave(); err != nil {
			return err
		}
	}
	log.Info("Trie dumping complete", "accounts", accounts,
		"elapsed", common.PrettyDuration(time.Since(start)))

	return nil
}

func getCode(addrHash common.Hash, data types.StateAccount, db state.Database) string {
	if bytes.Equal(data.CodeHash, emptyCodeHash) {
		return ""
	}

	code, err := db.ContractCode(
		addrHash,
		common.BytesToHash(data.CodeHash),
	)
	if err != nil {
		panic(err)
	}
	return common.Bytes2Hex(code)
}
