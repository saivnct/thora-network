package contractintgr

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/contractintgr/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"time"
)

func DecodePreAlloc(data string) (core.GenesisAlloc, error) {
	byteData, err := hexutil.Decode(data)
	if err != nil {
		return nil, err
	}

	g := core.GenesisAlloc{}
	err = json.Unmarshal(byteData, &g)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func DeploySMC() ([]byte, map[common.Hash]common.Hash, error) {
	//key, _ := crypto.GenerateKey()
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr := crypto.PubkeyToAddress(key.PublicKey)

	auth := bind.NewKeyedTransactor(key)

	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000000000000)}}, 30000000)
	gasPrice, err := contractBackend.SuggestGasPrice(context.Background())
	auth.GasPrice = gasPrice

	//erc20Address, tx, _, err := contract.DeployERC20Asset(auth, contractBackend, "CRYPFOX", "CFX", 18)
	erc20Address, _, _, err := contract.DeployERC20Asset(auth, contractBackend, "CRYPFOX", "CFX", 18)
	if err != nil {
		return nil, nil, err
	}
	//log.Printf("Contract pending deploy: 0x%x\n", erc20Address)
	//log.Printf("Transaction waiting to be mined: 0x%x\n\n", tx.Hash())

	contractBackend.Commit()

	if err != nil {
		return nil, nil, err
	}

	d := time.Now().Add(1000 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	code, err := contractBackend.CodeAt(ctx, erc20Address, nil)
	if err != nil {
		return nil, nil, err
	}
	//log.Printf("code of sc:  %v\n", hexutil.Encode(code[:]))

	storage := make(map[common.Hash]common.Hash)
	f := func(key, val common.Hash) bool {
		decode := []byte{}
		trim := bytes.TrimLeft(val.Bytes(), "\x00")
		rlp.DecodeBytes(trim, &decode)
		storage[key] = common.BytesToHash(decode)
		//log.Printf("DecodeBytes: key %v, value %v -  decode %v\n", key.String(), val.String(), storage[key].String())
		return true
	}
	contractBackend.ForEachStorageAt(ctx, erc20Address, nil, f)

	return code, storage, nil
}
