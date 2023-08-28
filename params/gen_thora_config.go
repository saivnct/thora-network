package params

import (
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"math/big"
)

var _ = (*thoraConfigMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (t ThoraConfig) MarshalJSON() ([]byte, error) {
	type ThoraConfig struct {
		Period          math.HexOrDecimal64   `json:"period"`                          // Number of seconds between blocks to enforce
		Epoch           math.HexOrDecimal64   `json:"epoch"`                           // Epoch length to reset votes and checkpoint
		BlockReward     *math.HexOrDecimal256 `json:"blockReward" gencodec:"required"` // Block reward
		RewardRecipient *common.Address       `json:"rewardRecipient,omitempty"`       //Reward Recipient, default recipients is validators if this value nil or zero address
	}
	var enc ThoraConfig
	enc.Period = math.HexOrDecimal64(t.Period)
	enc.Epoch = math.HexOrDecimal64(t.Epoch)
	enc.BlockReward = (*math.HexOrDecimal256)(t.BlockReward)
	if t.RewardRecipient != nil {
		enc.RewardRecipient = t.RewardRecipient
	}

	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *ThoraConfig) UnmarshalJSON(input []byte) error {
	type ThoraConfig struct {
		Period          *math.HexOrDecimal64  `json:"period"`                          // Number of seconds between blocks to enforce
		Epoch           *math.HexOrDecimal64  `json:"epoch"`                           // Epoch length to reset votes and checkpoint
		BlockReward     *math.HexOrDecimal256 `json:"blockReward" gencodec:"required"` // Block reward
		RewardRecipient *common.Address       `json:"rewardRecipient,omitempty"`       //Reward Recipient, default recipients is validators if this value nil or zero address
	}
	var dec ThoraConfig
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.BlockReward == nil {
		return errors.New("missing required field 'balance' for GenesisAccount")
	}
	t.BlockReward = (*big.Int)(dec.BlockReward)
	t.Period = uint64(*dec.Period)
	t.Epoch = uint64(*dec.Epoch)
	if dec.RewardRecipient != nil {
		t.RewardRecipient = dec.RewardRecipient
	}
	return nil
}
