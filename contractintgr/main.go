//go:build none
// +build none

package main

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/contractintgr"
	"github.com/ethereum/go-ethereum/core"
	"log"
	"math/big"
	"os"
)

func makeThoraAlloc(g *core.Genesis) (string, error) {
	code, storage, err := contractintgr.DeploySMC()
	if err != nil {
		return "", err
	}
	g.Alloc[common.HexToAddress(common.MasterSMC)] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    code,
		Storage: storage,
	}

	js, err := json.Marshal(g.Alloc)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(js), nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: mkalloc genesis.json")
		os.Exit(1)
	}

	g := new(core.Genesis)
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to Open file: %v", err)
	}
	if err := json.NewDecoder(file).Decode(g); err != nil {
		log.Fatalf("Failed to Decode genesis file: %v", err)
	}

	allocData, err := makeThoraAlloc(g)
	if err != nil {
		log.Fatalf("Failed to make Thora Alloc: %v", err)
	}

	log.Printf("const allocData: %v", allocData)

	decodeg, err := contractintgr.DecodePreAlloc(allocData)
	if err != nil {
		log.Fatalf("Failed to decode Thora Alloc: %v", err)
	}
	for address, account := range decodeg {
		log.Printf("address: %v\n", address.Hex())
		log.Printf("balance: %v\n", account.Balance)
		log.Printf("code: %v\n", account.Code)
		log.Printf("storage: %v\n", account.Storage)
		log.Println("--------------------------------------------------")
	}

}
