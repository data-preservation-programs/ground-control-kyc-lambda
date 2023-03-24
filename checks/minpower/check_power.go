package minpower

import (
	"context"
	"log"
	"math/big"
	"net/http"

	"github.com/filecoin-project/go-address"
	jsonrpc "github.com/filecoin-project/go-jsonrpc"
	lotusapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
)

// LookupPower gets the power for the miner from the Lotus API
func LookupPower(ctx context.Context, miner string) (*lotusapi.MinerPower, error) {
	headers := http.Header{}
	api_addr := "api.chain.love"

	var api lotusapi.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(ctx,
		"wss://"+api_addr+"/rpc/v0", "Filecoin",
		[]interface{}{&api.Internal, &api.CommonStruct.Internal}, headers)
	if err != nil {
		log.Fatalf("connecting with lotus failed: %s", err)
	}
	defer closer()

	addr, err := address.NewFromString(miner)
	if err != nil {
		return nil, err
	}

	power, err := api.StateMinerPower(ctx, addr, types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	log.Printf("Miner power %s: %v\n", miner, power)
	return power, nil
}

// MinQualityPowerOk compares the power from the API for miner against a minimum
func MinQualityPowerOk(ctx context.Context, miner string, min *big.Int) (bool, error) {
	power, err := LookupPower(ctx, miner)
	if err != nil {
		return false, err
	}
	if power.MinerPower.QualityAdjPower.Cmp(min) < 0 {
		log.Printf("Insufficient power %s: %v < %v\n", miner,
			power.MinerPower.QualityAdjPower, min)
		return false, nil
	}
	return true, nil
}
