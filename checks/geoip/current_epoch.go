package geoip

import (
	"context"
	"log"
	"net/http"

	jsonrpc "github.com/filecoin-project/go-jsonrpc"
	lotusapi "github.com/filecoin-project/lotus/api"
)

// GetCurrentEpoch gets the current chain height from the Lotus API
func GetCurrentEpoch(ctx context.Context) (int64, error) {
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

	ts, err := api.ChainHead(ctx)
	if err != nil {
		return 0, err
	}
	height := int64(ts.Height())
	log.Printf("Chain height: %v\n", height)
	return height, nil
}
