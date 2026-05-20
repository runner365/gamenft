package bindings

// Codegen placeholder for GameItem bindings.
// To generate real bindings with abigen, place the ABI at ./abis/GameItem.abi and run:
//   abigen --abi=abis/GameItem.abi --pkg=bindings --out=bindings/gameitem_bindings.go --type=GameItemContract

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// GameItemContract is a placeholder. Generated bindings will replace this file.
type GameItemContract struct {
	address common.Address
}

func NewGameItemContract(addr common.Address, backend bind.ContractBackend) (*GameItemContract, error) {
	return &GameItemContract{address: addr}, nil
}

func (g *GameItemContract) BalanceOf(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (g *GameItemContract) TokenOfOwnerByIndex(opts *bind.CallOpts, owner common.Address, index *big.Int) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (g *GameItemContract) TokenURI(opts *bind.CallOpts, tokenId *big.Int) (string, error) {
	return "", nil
}

func (g *GameItemContract) Address() common.Address { return g.address }
