// Copyright 2017 The go-ethereum Authors
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
	"bytes"
	"crypto/ecdsa"
	"fmt"
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
	"golang.org/x/exp/slices"
)

// testerAccountPool is a pool to maintain currently active tester accounts,
// mapped from textual names used in the tests below to actual Ethereum private
// keys capable of signing transactions.
type testerAccountPool struct {
	accounts map[string]*ecdsa.PrivateKey
}

func newTesterAccountPool() *testerAccountPool {
	return &testerAccountPool{
		accounts: make(map[string]*ecdsa.PrivateKey),
	}
}

// checkpoint creates a Thora checkpoint signer section from the provided list
// of authorized signers and embeds it into the provided header.
func (ap *testerAccountPool) checkpoint(header *types.Header, signers []string) {
	auths := make([]common.Address, len(signers))
	for i, signer := range signers {
		auths[i] = ap.address(signer)
	}
	slices.SortFunc(auths, common.Address.Less)
	for i, auth := range auths {
		copy(header.Extra[extraVanity+i*common.AddressLength:], auth.Bytes())
	}
}

// address retrieves the Ethereum address of a tester account by label, creating
// a new account if no previous one exists yet.
func (ap *testerAccountPool) address(account string) common.Address {
	// Return the zero account for non-addresses
	if account == "" {
		return common.Address{}
	}
	// Ensure we have a persistent key for the account
	if ap.accounts[account] == nil {
		ap.accounts[account], _ = crypto.GenerateKey()
	}
	// Resolve and return the Ethereum address
	return crypto.PubkeyToAddress(ap.accounts[account].PublicKey)
}

// sign calculates a Thora digital signature for the given block and embeds it
// back into the header.
func (ap *testerAccountPool) sign(header *types.Header, signer string) {
	// Ensure we have a persistent key for the signer
	if ap.accounts[signer] == nil {
		ap.accounts[signer], _ = crypto.GenerateKey()
	}
	// Sign the header and embed the signature in extra data
	sig, _ := crypto.Sign(SealHash(header).Bytes(), ap.accounts[signer])
	copy(header.Extra[len(header.Extra)-extraSeal:], sig)
}

// testerVote represents a single block signed by a particular account, where
// the account may or may not have cast a Thora vote.
type testerVote struct {
	signer     string
	voted      string
	auth       bool // auth: true, deauth: false
	checkpoint []string
	newbatch   bool
}

type thoraTest struct {
	epoch   uint64
	signers []string
	votes   []testerVote
	results []string
	failure error
}

// Tests that Thora signer voting is evaluated correctly for various simple and
// complex scenarios, as well as that a few special corner cases fail correctly.
func TestThora(t *testing.T) {
	// Define the various voting scenarios to test
	tests := []thoraTest{
		{
			// Single signer, no votes cast
			signers: []string{"A"},
			votes:   []testerVote{{signer: "A"}},
			results: []string{"A"},
		}, {
			// Single signer, voting to add two others (only accept first, second needs 2 votes)
			signers: []string{"A"},
			votes: []testerVote{
				{signer: "A", voted: "B", auth: true},
				{signer: "B"},
				{signer: "A", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			// Two signers, voting to add three others (only accept first two, third needs 3 votes already)
			signers: []string{"A", "B"},
			votes: []testerVote{
				{signer: "A", voted: "C", auth: true},
				{signer: "B", voted: "C", auth: true},
				{signer: "A", voted: "D", auth: true},
				{signer: "B", voted: "D", auth: true},
				{signer: "C"},
				{signer: "A", voted: "E", auth: true},
				{signer: "B", voted: "E", auth: true},
			},
			results: []string{"A", "B", "C", "D"},
		}, {
			// Single signer, dropping itself (weird, but one less cornercase by explicitly allowing this)
			signers: []string{"A"},
			votes: []testerVote{
				{signer: "A", voted: "A", auth: false},
			},
			results: []string{},
		}, {
			// Two signers, actually needing mutual consent to drop either of them (not fulfilled)
			signers: []string{"A", "B"},
			votes: []testerVote{
				{signer: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Two signers, actually needing mutual consent to drop either of them (fulfilled)
			signers: []string{"A", "B"},
			votes: []testerVote{
				{signer: "A", voted: "B", auth: false},
				{signer: "B", voted: "B", auth: false},
			},
			results: []string{"A"},
		}, {
			// Three signers, two of them deciding to drop the third
			signers: []string{"A", "B", "C"},
			votes: []testerVote{
				{signer: "A", voted: "C", auth: false},
				{signer: "B", voted: "C", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Four signers, consensus of two not being enough to drop anyone
			signers: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{signer: "A", voted: "C", auth: false},
				{signer: "B", voted: "C", auth: false},
			},
			results: []string{"A", "B", "C", "D"},
		}, {
			// Four signers, consensus of three already being enough to drop someone
			signers: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{signer: "A", voted: "D", auth: false},
				{signer: "B", voted: "D", auth: false},
				{signer: "C", voted: "D", auth: false},
			},
			results: []string{"A", "B", "C"},
		}, {
			// Authorizations are counted once per signer per target
			signers: []string{"A", "B"},
			votes: []testerVote{
				{signer: "A", voted: "C", auth: true},
				{signer: "B"},
				{signer: "A", voted: "C", auth: true},
				{signer: "B"},
				{signer: "A", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			// Authorizing multiple accounts concurrently is permitted
			signers: []string{"A", "B"},
			votes: []testerVote{
				{signer: "A", voted: "C", auth: true},
				{signer: "B"},
				{signer: "A", voted: "D", auth: true},
				{signer: "B"},
				{signer: "A"},
				{signer: "B", voted: "D", auth: true},
				{signer: "A"},
				{signer: "B", voted: "C", auth: true},
			},
			results: []string{"A", "B", "C", "D"},
		}, {
			// Deauthorizations are counted once per signer per target
			signers: []string{"A", "B"},
			votes: []testerVote{
				{signer: "A", voted: "B", auth: false},
				{signer: "B"},
				{signer: "A", voted: "B", auth: false},
				{signer: "B"},
				{signer: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Deauthorizing multiple accounts concurrently is permitted
			signers: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{signer: "A", voted: "C", auth: false},
				{signer: "B"},
				{signer: "C"},
				{signer: "A", voted: "D", auth: false},
				{signer: "B"},
				{signer: "C"},
				{signer: "A"},
				{signer: "B", voted: "D", auth: false},
				{signer: "C", voted: "D", auth: false},
				{signer: "A"},
				{signer: "B", voted: "C", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Votes from deauthorized signers are discarded immediately (deauth votes)
			signers: []string{"A", "B", "C"},
			votes: []testerVote{
				{signer: "C", voted: "B", auth: false},
				{signer: "A", voted: "C", auth: false},
				{signer: "B", voted: "C", auth: false},
				{signer: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Votes from deauthorized signers are discarded immediately (auth votes)
			signers: []string{"A", "B", "C"},
			votes: []testerVote{
				{signer: "C", voted: "D", auth: true},
				{signer: "A", voted: "C", auth: false},
				{signer: "B", voted: "C", auth: false},
				{signer: "A", voted: "D", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			// Cascading changes are not allowed, only the account being voted on may change
			signers: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{signer: "A", voted: "C", auth: false},
				{signer: "B"},
				{signer: "C"},
				{signer: "A", voted: "D", auth: false},
				{signer: "B", voted: "C", auth: false},
				{signer: "C"},
				{signer: "A"},
				{signer: "B", voted: "D", auth: false},
				{signer: "C", voted: "D", auth: false},
			},
			results: []string{"A", "B", "C"},
		}, {
			// Changes reaching consensus out of bounds (via a deauth) execute on touch
			signers: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{signer: "A", voted: "C", auth: false},
				{signer: "B"},
				{signer: "C"},
				{signer: "A", voted: "D", auth: false},
				{signer: "B", voted: "C", auth: false},
				{signer: "C"},
				{signer: "A"},
				{signer: "B", voted: "D", auth: false},
				{signer: "C", voted: "D", auth: false},
				{signer: "A"},
				{signer: "C", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			// Changes reaching consensus out of bounds (via a deauth) may go out of consensus on first touch
			signers: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{signer: "A", voted: "C", auth: false},
				{signer: "B"},
				{signer: "C"},
				{signer: "A", voted: "D", auth: false},
				{signer: "B", voted: "C", auth: false},
				{signer: "C"},
				{signer: "A"},
				{signer: "B", voted: "D", auth: false},
				{signer: "C", voted: "D", auth: false},
				{signer: "A"},
				{signer: "B", voted: "C", auth: true},
			},
			results: []string{"A", "B", "C"},
		}, {
			// Ensure that pending votes don't survive authorization status changes. This
			// corner case can only appear if a signer is quickly added, removed and then
			// re-added (or the inverse), while one of the original voters dropped. If a
			// past vote is left cached in the system somewhere, this will interfere with
			// the final signer outcome.
			signers: []string{"A", "B", "C", "D", "E"},
			votes: []testerVote{
				{signer: "A", voted: "F", auth: true}, // Authorize F, 3 votes needed
				{signer: "B", voted: "F", auth: true},
				{signer: "C", voted: "F", auth: true},
				{signer: "D", voted: "F", auth: false}, // Deauthorize F, 4 votes needed (leave A's previous vote "unchanged")
				{signer: "E", voted: "F", auth: false},
				{signer: "B", voted: "F", auth: false},
				{signer: "C", voted: "F", auth: false},
				{signer: "D", voted: "F", auth: true}, // Almost authorize F, 2/3 votes needed
				{signer: "E", voted: "F", auth: true},
				{signer: "B", voted: "A", auth: false}, // Deauthorize A, 3 votes needed
				{signer: "C", voted: "A", auth: false},
				{signer: "D", voted: "A", auth: false},
				{signer: "B", voted: "F", auth: true}, // Finish authorizing F, 3/3 votes needed
			},
			results: []string{"B", "C", "D", "E", "F"},
		}, {
			// Epoch transitions reset all votes to allow chain checkpointing
			epoch:   3,
			signers: []string{"A", "B"},
			votes: []testerVote{
				{signer: "A", voted: "C", auth: true},
				{signer: "B"},
				{signer: "A", checkpoint: []string{"A", "B"}},
				{signer: "B", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			// An unauthorized signer should not be able to sign blocks
			signers: []string{"A"},
			votes: []testerVote{
				{signer: "B"},
			},
			failure: errUnauthorizedSigner,
		}, {
			// An authorized signer that signed recently should not be able to sign again
			signers: []string{"A", "B"},
			votes: []testerVote{
				{signer: "A"},
				{signer: "A"},
			},
			failure: errRecentlySigned,
		}, {
			// Recent signatures should not reset on checkpoint blocks imported in a batch
			epoch:   3,
			signers: []string{"A", "B", "C"},
			votes: []testerVote{
				{signer: "A"},
				{signer: "B"},
				{signer: "A", checkpoint: []string{"A", "B", "C"}},
				{signer: "A"},
			},
			failure: errRecentlySigned,
		}, {
			// Recent signatures should not reset on checkpoint blocks imported in a new
			// batch (https://github.com/ethereum/go-ethereum/issues/17593). Whilst this
			// seems overly specific and weird, it was a Rinkeby consensus split.
			epoch:   3,
			signers: []string{"A", "B", "C"},
			votes: []testerVote{
				{signer: "A"},
				{signer: "B"},
				{signer: "A", checkpoint: []string{"A", "B", "C"}},
				{signer: "A", newbatch: true},
			},
			failure: errRecentlySigned,
		},
	}

	// Run through the scenarios and test them
	for i, tt := range tests {
		t.Run(fmt.Sprint(i), tt.run)
	}
}

func (tt *thoraTest) run(t *testing.T) {
	// Create the account pool and generate the initial set of signers
	accounts := newTesterAccountPool()

	signers := make([]common.Address, len(tt.signers))
	for j, signer := range tt.signers {
		signers[j] = accounts.address(signer)
	}
	for j := 0; j < len(signers); j++ {
		for k := j + 1; k < len(signers); k++ {
			if bytes.Compare(signers[j][:], signers[k][:]) > 0 {
				signers[j], signers[k] = signers[k], signers[j]
			}
		}
	}
	// Create the genesis block with the initial set of signers
	genesis := &core.Genesis{
		ExtraData: make([]byte, extraVanity+common.AddressLength*len(signers)+extraSeal),
		BaseFee:   big.NewInt(params.InitialBaseFee),
	}
	for j, signer := range signers {
		copy(genesis.ExtraData[extraVanity+j*common.AddressLength:], signer[:])
	}

	// Assemble a chain of headers from the cast votes
	config := *params.TestChainConfig
	config.Thora = &params.ThoraConfig{
		Period:          1,
		Epoch:           tt.epoch,
		BlockReward:     new(big.Int).Mul(big.NewInt(100), big.NewInt(params.Ether)),
		RewardRecipient: nil,
	}
	genesis.Config = &config

	engine := New(config.Thora, rawdb.NewMemoryDatabase())
	engine.fakeDiff = true

	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, len(tt.votes), func(j int, gen *core.BlockGen) {
		// Cast the vote contained in this block
		gen.SetCoinbase(accounts.address(tt.votes[j].voted))
		if tt.votes[j].auth {
			var nonce types.BlockNonce
			copy(nonce[:], nonceAuthVote)
			gen.SetNonce(nonce)
		}
	})

	blocks = makePlatformChain2(genesis, blocks, accounts, tt)

	// Iterate through the blocks and seal them individually
	for j, block := range blocks {
		// Get the header and prepare it for signing
		header := block.Header()
		if j > 0 {
			header.ParentHash = blocks[j-1].Hash()
		}
		header.Extra = make([]byte, extraVanity+extraSeal)
		if auths := tt.votes[j].checkpoint; auths != nil {
			header.Extra = make([]byte, extraVanity+len(auths)*common.AddressLength+extraSeal)
			accounts.checkpoint(header, auths)
		}
		header.Difficulty = diffInTurn // Ignored, we just need a valid number

		// Generate the signature, embed it into the header and the block
		accounts.sign(header, tt.votes[j].signer)
		blocks[j] = block.WithSeal(header)
	}
	// Split the blocks up into individual import batches (cornercase testing)
	batches := [][]*types.Block{nil}
	for j, block := range blocks {
		if tt.votes[j].newbatch {
			batches = append(batches, nil)
		}
		batches[len(batches)-1] = append(batches[len(batches)-1], block)
	}
	// Pass all the headers through thora and ensure tallying succeeds
	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, genesis, nil, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test chain: %v", err)
	}
	defer chain.Stop()

	for j := 0; j < len(batches)-1; j++ {
		if k, err := chain.InsertChain(batches[j]); err != nil {
			t.Fatalf("failed to import batch %d, block %d: %v", j, k, err)
			break
		}
	}
	if _, err = chain.InsertChain(batches[len(batches)-1]); err != tt.failure {
		t.Errorf("failure mismatch: have %v, want %v", err, tt.failure)
	}
	if tt.failure != nil {
		return
	}

	// No failure was produced or requested, generate the final voting snapshot
	head := blocks[len(blocks)-1]

	snap, err := engine.snapshot(chain, head.NumberU64(), head.Hash(), nil)
	if err != nil {
		t.Fatalf("failed to retrieve voting snapshot: %v", err)
	}
	// Verify the final list of signers against the expected ones
	signers = make([]common.Address, len(tt.results))
	for j, signer := range tt.results {
		signers[j] = accounts.address(signer)
	}
	for j := 0; j < len(signers); j++ {
		for k := j + 1; k < len(signers); k++ {
			if bytes.Compare(signers[j][:], signers[k][:]) > 0 {
				signers[j], signers[k] = signers[k], signers[j]
			}
		}
	}
	result := snap.signers()
	if len(result) != len(signers) {
		t.Fatalf("signers mismatch: have %x, want %x", result, signers)
	}
	for j := 0; j < len(result); j++ {
		if !bytes.Equal(result[j][:], signers[j][:]) {
			t.Fatalf("signer %d: signer mismatch: have %x, want %x", j, result[j], signers[j])
		}
	}
}

func makePlatformChain2(genesis *core.Genesis, blocks []*types.Block, accounts *testerAccountPool, tt *thoraTest) []*types.Block {
	newDb := rawdb.NewMemoryDatabase()
	parent, err := genesis.Commit(newDb, trie.NewDatabase(newDb))
	if err != nil {
		panic(err)
	}

	newBlocks := make(types.Blocks, len(blocks))

	for i, block := range blocks {
		header := block.Header()

		statedb, err := state.New(parent.Root(), state.NewDatabase(newDb), nil)
		if err != nil {
			panic(err)
		}
		receipts := []*types.Receipt{}
		txs := []*types.Transaction{}

		header.ParentHash = parent.Hash()

		if i > 0 {
			signer := accounts.address(tt.votes[i-1].signer)
			statedb.AddBalance(signer, genesis.Config.Thora.BlockReward)
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
	}

	return newBlocks
}
