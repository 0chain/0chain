package faucetsc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"0chain.net/pkg/tokens"

	"0chain.net/core/common"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -v

type periodicResponse struct {
	Used    int64     `json:"tokens_poured"`
	Start   time.Time `json:"start_time"`
	Restart string    `json:"time_left"`
	Allowed int64     `json:"tokens_allowed"`
}

type GlobalNode struct {
	*FaucetConfig `json:"faucet_config"`
	ID            string    `json:"id"`
	Used          int64     `json:"used"`
	StartTime     time.Time `json:"start_time"`
}

func (gn *GlobalNode) GetKey() datastore.Key {
	return datastore.Key(gn.ID + gn.ID)
}

func (gn *GlobalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *GlobalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}

func (gn *GlobalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *GlobalNode) Decode(input []byte) error {
	err := json.Unmarshal(input, gn)
	return err
}

func (gn *GlobalNode) updateConfig(fields map[string]string) error {
	for key, value := range fields {
		switch key {
		case Settings[PourAmount]:
			fAmount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert zcn %v to sas", key, value)
			}
			gn.PourAmount = tokens.ZCNToSAS(fAmount)
		case Settings[MaxPourAmount]:
			fAmount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert zcn %v to sas", key, value)
			}
			gn.MaxPourAmount = tokens.ZCNToSAS(fAmount)
		case Settings[PeriodicLimit]:
			fAmount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert zcn %v to sas", key, value)
			}
			gn.PeriodicLimit = tokens.ZCNToSAS(fAmount)
		case Settings[GlobalLimit]:
			fAmount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert zcn %v to sas", key, value)
			}
			gn.GlobalLimit = tokens.ZCNToSAS(fAmount)
		case Settings[IndividualReset]:
			ir, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to time.duration", key, value)
			}

			gn.IndividualReset = ir
		case Settings[GlobalReset]:
			gr, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to time.duration", key, value)
			}

			gn.GlobalReset = gr

		case Settings[OwnerId]:
			_, err := hex.DecodeString(value)
			if err != nil {
				return fmt.Errorf("key %s, %v should be valid hex string", key, value)
			}
			gn.OwnerId = value

		default:
			return gn.setCostValue(key, value)
		}
	}
	return nil
}

func (gn *GlobalNode) setCostValue(key, value string) error {
	if !strings.HasPrefix(key, Settings[Cost]) {
		return fmt.Errorf("key %s not recognised as setting", key)
	}
	costKey := strings.ToLower(strings.TrimPrefix(key, Settings[Cost]+"."))
	for _, costFunction := range costFunctions {
		if costKey != strings.ToLower(costFunction) {
			continue
		}
		costValue, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("key %s, unable to convert %v to integer", key, value)
		}

		if costValue < 0 {
			return fmt.Errorf("cost.%s contains invalid value %s", key, value)
		}

		gn.Cost[costKey] = costValue

		return nil
	}

	return fmt.Errorf("cost config setting %s not found", costKey)
}

func (gn *GlobalNode) validate() error {
	switch {
	case gn.PourAmount < 1:
		return common.NewError("failed to validate global node", fmt.Sprintf("pour amount(%v) is less than 1", gn.PourAmount))
	case gn.PourAmount > gn.MaxPourAmount:
		return common.NewError("failed to validate global node", fmt.Sprintf("max pour amount(%v) is less than pour amount(%v)", gn.MaxPourAmount, gn.PourAmount))
	case gn.MaxPourAmount > gn.PeriodicLimit:
		return common.NewError("failed to validate global node", fmt.Sprintf("periodic limit(%v) is less than max pour amount(%v)", gn.PeriodicLimit, gn.MaxPourAmount))
	case gn.PeriodicLimit > gn.GlobalLimit:
		return common.NewError("failed to validate global node", fmt.Sprintf("global periodic limit(%v) is less than periodic limit(%v)", gn.GlobalLimit, gn.PeriodicLimit))
	case toSeconds(gn.IndividualReset) < 1:
		return common.NewError("failed to validate global node", fmt.Sprintf("individual reset(%v) is too short", gn.IndividualReset))
	case gn.GlobalReset < gn.IndividualReset:
		return common.NewError("failed to validate global node", fmt.Sprintf("global reset(%v) is less than individual reset(%v)", gn.GlobalReset, gn.IndividualReset))
	}

	return nil
}

type UserNode struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"start_time"`
	Used      int64     `json:"used"`
}

func (un *UserNode) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + un.ID)
}

func (un *UserNode) GetHash() string {
	return util.ToHex(un.GetHashBytes())
}

func (un *UserNode) GetHashBytes() []byte {
	return encryption.RawHash(un.Encode())
}

func (un *UserNode) Encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *UserNode) Decode(input []byte) error {
	err := json.Unmarshal(input, un)
	return err
}
