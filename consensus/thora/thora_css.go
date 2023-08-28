package thora

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

func (c *Thora) IsCurrentValidator(etherbase common.Address, chain consensus.ChainHeaderReader) (bool, error) {
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
// consensus rules in thora, do nothing here.
func (c *Thora) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, withdrawals []*types.Withdrawal) {
	if c.config.BlockReward == nil {
		return
	}

	if c.config.RewardRecipient != nil && *c.config.RewardRecipient != (common.Address{}) {
		if header.Number.Uint64() > 0 {
			state.AddBalance(*c.config.RewardRecipient, c.config.BlockReward)
		}
	} else {
		if header.Number.Uint64() > 1 {
			// Reward the signer.
			parentHeader := chain.GetHeaderByHash(header.ParentHash)

			if parentHeader != nil {
				if parentHeader.Extra != nil {
					parentSigner, err := c.Author(parentHeader)
					if err != nil {
						log.Error("Thora Finalize: failed to get Author", "err", err)
						return
					}
					state.AddBalance(parentSigner, c.config.BlockReward)
				}

			}
		}
	}
}
