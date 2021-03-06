// Copyright (c) The Tellor Authors.
// Licensed under the MIT License.

package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/tellor-io/telliot/pkg/common"
	tellorCommon "github.com/tellor-io/telliot/pkg/common"
	"github.com/tellor-io/telliot/pkg/db"
	"github.com/tellor-io/telliot/pkg/rpc"
)

// GasTracker is the struct that maintains the latest gasprices.
// note the prices are actually stored in the DB.
type GasTracker struct {
	logger log.Logger
}

// GasPriceModel is what ETHGasStation returns from queries. Not all fields are filled in.
type GasPriceModel struct {
	Fast    float32 `json:"fast"`
	Fastest float32 `json:"fastest"`
	Average float32 `json:"average"`
}

func (b *GasTracker) String() string {
	return "GasTracker"
}

func NewGasTracker(logger log.Logger) *GasTracker {
	return &GasTracker{
		logger: log.With(logger, "component", "gas tracker"),
	}

}

func (b *GasTracker) Exec(ctx context.Context) error {
	client := ctx.Value(tellorCommon.ClientContextKey).(rpc.ETHClient)
	DB := ctx.Value(tellorCommon.DBContextKey).(db.DB)

	netID, err := client.NetworkID(context.Background())
	if err != nil {
		fmt.Println(err)
		return err
	}

	var gasPrice *big.Int

	if big.NewInt(1).Cmp(netID) == 0 {
		url := "https://ethgasstation.info/json/ethgasAPI.json"
		req := &FetchRequest{queryURL: url, timeout: time.Duration(15 * time.Second)}
		payload, err := fetchWithRetries(req)
		if err != nil {
			gasPrice, err = client.SuggestGasPrice(context.Background())
			if err != nil {
				level.Warn(b.logger).Log("msg", "couldn't get suggested gas price", "err", err)
			}
		} else {
			gpModel := GasPriceModel{}
			err = json.Unmarshal(payload, &gpModel)
			if err != nil {
				level.Warn(b.logger).Log("msg", "eth gas station json", "err", err)
				gasPrice, err = client.SuggestGasPrice(context.Background())
				if err != nil {
					level.Warn(b.logger).Log("msg", "getting suggested gas price", "err", err)
				}
			} else {
				gasPrice = big.NewInt(int64(gpModel.Fast / 10))
				gasPrice = gasPrice.Mul(gasPrice, big.NewInt(common.GWEI))
				level.Info(b.logger).Log("msg", "using ETHGasStation fast price", "price", gasPrice)
			}
		}
	} else {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			level.Warn(b.logger).Log("msg", "getting suggested gas price", "err", err)
		}
	}

	enc := hexutil.EncodeBig(gasPrice)
	return DB.Put(db.GasKey, []byte(enc))
}
