package migrator

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"io"
	"math/big"
)

type AllocDumper struct {
	w io.Writer
}

func (a *AllocDumper) OnRoot(hash common.Hash) {
	// noop
}

func (a *AllocDumper) OnAccount(address common.Address, account state.DumpAccount) {
	balBig, _ := new(big.Int).SetString(account.Balance, 10)
	storage := make(map[common.Hash]common.Hash)
	for k, v := range account.Storage {
		storage[k] = common.HexToHash(v)
	}

	acc := &core.GenesisAccount{
		Code:       account.Code,
		Storage:    storage,
		Balance:    balBig,
		Nonce:      account.Nonce,
	}
	accJSON, err := json.Marshal(acc)
	if err != nil {
		panic(err)
	}

	_, err = a.w.Write([]byte("\"" + address.String() + "\": "))
	if err != nil {
		panic(err)
	}
	_, err = a.w.Write(accJSON)
	if err != nil {
		panic(err)
	}
	_, err = a.w.Write([]byte(",\n"))
	if err != nil {
		panic(err)
	}
}
