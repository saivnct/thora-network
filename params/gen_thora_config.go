package params

import (
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common/math"
	"math/big"
)

var _ = (*thoraConfigMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (t ThoraConfig) MarshalJSON() ([]byte, error) {
	type ThoraConfig struct {
		BlockReward *math.HexOrDecimal256 `json:"blockReward" gencodec:"required"` // Block reward
	}
	var enc ThoraConfig
	enc.BlockReward = (*math.HexOrDecimal256)(t.BlockReward)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *ThoraConfig) UnmarshalJSON(input []byte) error {
	type ThoraConfig struct {
		BlockReward *math.HexOrDecimal256 `json:"blockReward" gencodec:"required"` // Block reward
	}
	var dec ThoraConfig
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.BlockReward == nil {
		return errors.New("missing required field 'balance' for GenesisAccount")
	}
	t.BlockReward = (*big.Int)(dec.BlockReward)
	return nil
}
