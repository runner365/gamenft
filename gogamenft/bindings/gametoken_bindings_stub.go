// Codegen placeholder for GameToken (ERC20) bindings.
// To generate real bindings with abigen, place the ABI at ./abis/GameToken.abi and run:
//   abigen --abi=abis/GameToken.abi --pkg=bindings --out=bindings/gametoken_bindings.go --type=GameTokenContract

package bindings

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// GameTokenContract is a placeholder. Generated bindings will replace this file.
type GameTokenContract struct {
	address common.Address
}

func NewGameTokenContract(addr common.Address, backend bind.ContractBackend) (*GameTokenContract, error) {
	return &GameTokenContract{address: addr}, nil
}

func (g *GameTokenContract) BalanceOf(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (g *GameTokenContract) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return nil, nil
}

func (g *GameTokenContract) Address() common.Address { return g.address }
