// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package thora

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func makePlatformChain(genesis *core.Genesis, engine consensus.Engine, blocks []*types.Block, signer common.Address, key *ecdsa.PrivateKey, txSigner *types.HomesteadSigner) []*types.Block {
	newDb := rawdb.NewMemoryDatabase()
	parent, err := genesis.Commit(newDb, trie.NewDatabase(newDb))
	var parentSigner *common.Address

	if err != nil {
		panic(err)
	}
	chain, _ := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, genesis, nil, engine, vm.Config{}, nil, nil)
	defer chain.Stop()

	newBlocks := make(types.Blocks, len(blocks))

	for i, block := range blocks {
		header := block.Header()

		receipts := []*types.Receipt{}
		txs := []*types.Transaction{}

		statedb, err := state.New(parent.Root(), state.NewDatabase(newDb), nil)
		if err != nil {
			panic(err)
		}

		header.ParentHash = parent.Hash()
		header.GasLimit = parent.GasLimit()
		header.Coinbase = parent.Coinbase()
		if chain.Config().IsLondon(header.Number) {
			header.BaseFee = misc.CalcBaseFee(chain.Config(), parent.Header())
			if !chain.Config().IsLondon(parent.Number()) {
				parentGasLimit := parent.GasLimit() * chain.Config().ElasticityMultiplier()
				header.GasLimit = core.CalcGasLimit(parentGasLimit, parentGasLimit)
			}
		}

		// We want to simulate an empty middle block, having the same state as the
		// first one. The last is needs a state change again to force a reorg.
		if i != 1 {
			txNonce := statedb.GetNonce(signer)
			baseFee := new(big.Int).Set(header.BaseFee)
			tx, err := types.SignTx(types.NewTransaction(txNonce, common.Address{0x00}, new(big.Int), params.TxGas, baseFee, nil), txSigner, key)
			if err != nil {
				panic(err)
			}
			gasPool := new(core.GasPool).AddGas(header.GasLimit)

			statedb.SetTxContext(tx.Hash(), 0)

			receipt, err := core.ApplyTransaction(genesis.Config, chain, &header.Coinbase, gasPool, statedb, header, tx, &header.GasUsed, vm.Config{})
			if err != nil {
				panic(err)
			}

			receipts = append(receipts, receipt)
			txs = append(txs, tx)
		}

		if parentSigner != nil {
			statedb.AddBalance(*parentSigner, genesis.Config.Thora.BlockReward)
		}

		header.Root = statedb.IntermediateRoot(genesis.Config.IsEIP158(header.Number))

		newBlock := types.NewBlock(header, txs, nil, receipts, trie.NewStackTrie(nil))

		// Write state changes to db
		root, err := statedb.Commit(genesis.Config.IsEIP158(header.Number))
		if err != nil {
			panic(fmt.Sprintf("state write error: %v", err))
		}

		if err := statedb.Database().TrieDB().Commit(root, false); err != nil {
			panic(fmt.Sprintf("trie write error: %v", err))
		}

		newBlocks[i] = newBlock
		parent = newBlock
		parentSigner = &signer
	}

	return newBlocks
}

// This test case is a repro of an annoying bug that took us forever to catch.
// In Thora network, consecutive blocks might have
// the same state root (no block subsidy, empty block). If a node crashes, the
// chain ends up losing the recent state and needs to regenerate it from blocks
// already in the database. The bug was that processing the block *prior* to an
// empty one **also completes** the empty one, ending up in a known-block error.
func TestReimportMirroredState(t *testing.T) {
	// Initialize a Thora chain with a single signer
	var (
		db     = rawdb.NewMemoryDatabase()
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		engine = New(params.AllThoraProtocolChanges.Thora, db)
		signer = new(types.HomesteadSigner)
	)
	genspec := &core.Genesis{
		Config:    params.AllThoraProtocolChanges,
		ExtraData: make([]byte, extraVanity+common.AddressLength+extraSeal),
		Alloc: map[common.Address]core.GenesisAccount{
			addr: {Balance: big.NewInt(10000000000000000)},
		},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	copy(genspec.ExtraData[extraVanity:], addr[:])

	// Generate a batch of blocks, each properly signed
	chain, _ := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, genspec, nil, engine, vm.Config{}, nil, nil)
	defer chain.Stop()

	_, blocks, _ := core.GenerateChainWithGenesis(genspec, engine, 3, func(i int, block *core.BlockGen) {
		// The chain maker doesn't have access to a chain, so the difficulty will be
		// lets unset (nil). Set it here to the correct value.
		block.SetDifficulty(diffInTurn)

		// We want to simulate an empty middle block, having the same state as the
		// first one. The last is needs a state change again to force a reorg.
		//if i != 1 {
		//	tx, err := types.SignTx(types.NewTransaction(block.TxNonce(addr), common.Address{0x00}, new(big.Int), params.TxGas, block.BaseFee(), nil), signer, key)
		//	if err != nil {
		//		panic(err)
		//	}
		//block.AddTxWithChain(chain, tx)
		//}
	})

	blocks = makePlatformChain(genspec, engine, blocks, addr, key, signer)

	for i, block := range blocks {
		header := block.Header()
		if i > 0 {
			header.ParentHash = blocks[i-1].Hash()
		}
		header.Extra = make([]byte, extraVanity+extraSeal)
		header.Difficulty = diffInTurn

		sig, _ := crypto.Sign(SealHash(header).Bytes(), key)
		copy(header.Extra[len(header.Extra)-extraSeal:], sig)
		blocks[i] = block.WithSeal(header)
	}
	// Insert the first two blocks and make sure the chain is valid
	db = rawdb.NewMemoryDatabase()
	chain, _ = core.NewBlockChain(db, nil, genspec, nil, engine, vm.Config{}, nil, nil)
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:2]); err != nil {
		t.Fatalf("failed to insert initial blocks: %v", err)
	}
	if head := chain.CurrentBlock().Number.Uint64(); head != 2 {
		t.Fatalf("chain head mismatch: have %d, want %d", head, 2)
	}

	// Simulate a crash by creating a new chain on top of the database, without
	// flushing the dirty states out. Insert the last block, triggering a sidechain
	// reimport.
	chain, _ = core.NewBlockChain(db, nil, genspec, nil, engine, vm.Config{}, nil, nil)
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[2:]); err != nil {
		t.Fatalf("failed to insert final block: %v", err)
	}
	if head := chain.CurrentBlock().Number.Uint64(); head != 3 {
		t.Fatalf("chain head mismatch: have %d, want %d", head, 3)
	}
}

func TestSealHash(t *testing.T) {
	have := SealHash(&types.Header{
		Difficulty: new(big.Int),
		Number:     new(big.Int),
		Extra:      make([]byte, 32+65),
		BaseFee:    new(big.Int),
	})
	want := common.HexToHash("0xbd3d1fa43fbc4c5bfcc91b179ec92e2861df3654de60468beb908ff805359e8f")
	if have != want {
		t.Errorf("have %x, want %x", have, want)
	}
}
