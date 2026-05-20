package services

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// Listing is a minimal representation of a marketplace listing.
type Listing struct {
	Active bool
	Price  *big.Int
}

// MarketI represents the minimal marketplace contract methods used here.
type MarketI interface {
	GetListing(opts *bind.CallOpts, nft common.Address, tokenId *big.Int) (Listing, error)
	Address() common.Address
}
