package clique

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
)

func (c *Clique) IsCurrentValidator(etherbase common.Address, chain consensus.ChainHeaderReader) (bool, error) {
	currentHeader := chain.CurrentHeader()
	number := currentHeader.Number.Uint64()

	snap, err := c.snapshot(chain, number, currentHeader.Hash(), nil)
	if err != nil {
		return false, err
	}
	_, ok := snap.Signers[etherbase]
	return ok, nil
}
