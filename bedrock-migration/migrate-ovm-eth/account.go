package migrator

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
)

type MigratedAccount struct {
	Code    string                      `json:"code"`
	Nonce   math.HexOrDecimal64         `json:"nonce"`
	Storage map[common.Hash]common.Hash `json:"storage"`
	Balance *math.HexOrDecimal64        `json:"balance"`
}
