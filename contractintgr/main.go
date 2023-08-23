package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/contractintgr/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/exp/slices"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"
)

type allocItem struct{ Addr, Balance *big.Int }

func makelist(g *core.Genesis) []allocItem {
	items := make([]allocItem, 0, len(g.Alloc))
	for addr, account := range g.Alloc {
		if len(account.Storage) > 0 || len(account.Code) > 0 || account.Nonce != 0 {
			panic(fmt.Sprintf("can't encode account %x", addr))
		}
		bigAddr := new(big.Int).SetBytes(addr.Bytes())
		items = append(items, allocItem{bigAddr, account.Balance})
	}
	slices.SortFunc(items, func(a, b allocItem) bool {
		return a.Addr.Cmp(b.Addr) < 0
	})
	return items
}

func makealloc(g *core.Genesis) string {
	a := makelist(g)
	data, err := rlp.EncodeToBytes(a)
	if err != nil {
		panic(err)
	}
	return strconv.QuoteToASCII(string(data))
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: mkalloc genesis.json")
		os.Exit(1)
	}

	g := new(core.Genesis)
	file, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err := json.NewDecoder(file).Decode(g); err != nil {
		panic(err)
	}

	code, storage, err := deployERC20()
	if err != nil {
		log.Fatalf("Failed deployERC20 - %v", err)
	}
	g.Alloc[common.HexToAddress(common.MasterSMC)] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    code,
		Storage: storage,
	}

	fmt.Println("const allocData =", makealloc(g))
}

func deployERC20() ([]byte, map[common.Hash]common.Hash, error) {
	//key, _ := crypto.GenerateKey()
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr := crypto.PubkeyToAddress(key.PublicKey)

	auth := bind.NewKeyedTransactor(key)

	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000000000000)}}, 30000000)
	gasPrice, err := contractBackend.SuggestGasPrice(context.Background())
	auth.GasPrice = gasPrice

	erc20Address, tx, _, err := contract.DeployERC20Asset(auth, contractBackend, "CRYPFOX", "CFX", 18)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("Contract pending deploy: 0x%x\n", erc20Address)
	log.Printf("Transaction waiting to be mined: 0x%x\n\n", tx.Hash())

	contractBackend.Commit()

	if err != nil {
		log.Fatalf("Failed to deploy new contract: %v", err)
	}

	d := time.Now().Add(1000 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	code, err := contractBackend.CodeAt(ctx, erc20Address, nil)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("code of sc:  %v\n", hexutil.Encode(code[:]))

	storage := make(map[common.Hash]common.Hash)
	f := func(key, val common.Hash) bool {
		decode := []byte{}
		trim := bytes.TrimLeft(val.Bytes(), "\x00")
		rlp.DecodeBytes(trim, &decode)
		storage[key] = common.BytesToHash(decode)
		log.Printf("DecodeBytes: value - %v -  decode %v\n", val.String(), storage[key].String())
		return true
	}
	contractBackend.ForEachStorageAt(ctx, erc20Address, nil, f)

	return code, storage, nil
}
