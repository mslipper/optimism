package main

import (
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
)

var (
	OVMETHAddress    = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")
	Block7412000Root = common.HexToHash("0x6894615eaf0954b58ad66adc077449e9ad4885824ee80def7a8a7873bfdade19")
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

	iter := st.NodeIterator(nil)
	balKey := GetOVMBalanceKey(common.HexToAddress("0xa84c44ffd029674f00affcb67b42ee0c7ab1c194"))
	//stBal := stateDB.GetState(OVMETHAddress, balKey)
	fmt.Println(balKey)
	res, err := st.TryGet(balKey[:])
	fmt.Println(err)
	_, decRes, _, _ := rlp.Split(res)
	fmt.Println(new(big.Int).SetBytes(decRes).String())


	for iter.Next(true) {
		if !iter.Leaf() {
			continue
		}
		preimage := st.GetKey(iter.LeafKey())
		fmt.Printf("%x|%x|%x\n", preimage,iter.LeafKey(), iter.LeafBlob())
		return
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