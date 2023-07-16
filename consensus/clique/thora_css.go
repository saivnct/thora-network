package clique

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

// BlockReward is the reward in wei distributed each block.
var BlockReward = new(big.Int).Mul(big.NewInt(100), big.NewInt(params.GWei))

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

// Finalize implements consensus.Engine. There is no post-transaction
// consensus rules in clique, do nothing here.
func (c *Clique) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, withdrawals []*types.Withdrawal) {
	// Reward the signer.
	parentHeader := chain.GetHeaderByHash(header.ParentHash)
	if parentHeader.Number.Uint64() > 0 {
		parentSigner, err := ecrecover(parentHeader, c.signatures)
		if err != nil {
			log.Error("Clique Finalize: ecrecover failed", "err", err)
			return
		}

		//log.Info("Clique Finalize:", "blockNumber", header.Number, "coinbase", header.Coinbase, "reward", BlockReward, "signer", parentSigner)
		state.AddBalance(parentSigner, BlockReward)
	}
}
