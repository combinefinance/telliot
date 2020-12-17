// Copyright (c) The Tellor Authors.
// Licensed under the MIT License.

package main

import (
	"context"
	"crypto/ecdsa"
	"os"

	"github.com/alecthomas/kong"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	tellorCommon "github.com/tellor-io/telliot/pkg/common"
	"github.com/tellor-io/telliot/pkg/config"
	"github.com/tellor-io/telliot/pkg/contracts/getter"
	"github.com/tellor-io/telliot/pkg/contracts/tellor"
	"github.com/tellor-io/telliot/pkg/db"
	"github.com/tellor-io/telliot/pkg/rpc"
)

var ctx context.Context

func setup() (rpc.ETHClient, tellorCommon.Contract, tellorCommon.Account, error) {

	cfg := config.GetConfig()

	if !cfg.EnablePoolWorker {

		// Create an rpc client
		client, err := rpc.NewClient(cfg.NodeURL)
		if err != nil {
			return nil, tellorCommon.Contract{}, tellorCommon.Account{}, errors.Wrap(err, "create rpc client instance")
		}

		// Create an instance of the tellor master contract for on-chain interactions
		contractAddress := common.HexToAddress(cfg.ContractAddress)
		contractTellorInstance, err := tellor.NewTellor(contractAddress, client)
		if err != nil {
			return nil, tellorCommon.Contract{}, tellorCommon.Account{}, errors.Wrap(err, "create tellor master instance")
		}

		contractGetterInstance, err := getter.NewTellorGetters(contractAddress, client)

		if err != nil {
			return nil, tellorCommon.Contract{}, tellorCommon.Account{}, errors.Wrap(err, "create tellor transactor instance")
		}
		// Leaving those in because are still used in some places(miner submission mostly).
		ctx := context.WithValue(context.Background(), tellorCommon.ClientContextKey, client)
		ctx = context.WithValue(ctx, tellorCommon.ContractAddress, contractAddress)
		ctx = context.WithValue(ctx, tellorCommon.ContractsTellorContextKey, contractTellorInstance)
		ctx = context.WithValue(ctx, tellorCommon.ContractsGetterContextKey, contractGetterInstance)

		privateKey, err := crypto.HexToECDSA(cfg.PrivateKey)
		if err != nil {
			return nil, tellorCommon.Contract{}, tellorCommon.Account{}, errors.Wrap(err, "getting private key to ECDSA")
		}

		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, tellorCommon.Contract{}, tellorCommon.Account{}, errors.New("casting public key to ECDSA")
		}

		publicAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

		// Issue #55, halt if client is still syncing with Ethereum network
		s, err := client.IsSyncing(ctx)
		if err != nil {
			return nil, tellorCommon.Contract{}, tellorCommon.Account{}, errors.Wrap(err, "determining if Ethereum client is syncing")
		}
		if s {
			return nil, tellorCommon.Contract{}, tellorCommon.Account{}, errors.New("ethereum node is still syncing with the network")
		}

		account := tellorCommon.Account{
			Address:    publicAddress,
			PrivateKey: privateKey,
		}
		contract := tellorCommon.Contract{
			Getter:  contractGetterInstance,
			Caller:  contractTellorInstance,
			Address: contractAddress,
		}
		return client, contract, account, nil
	}
	// Not sure why we need this case.
	return nil, tellorCommon.Contract{}, tellorCommon.Account{}, nil
}

func AddDBToCtx(remote bool) error {
	cfg := config.GetConfig()
	// Create a db instance
	os.RemoveAll(cfg.DBFile)
	DB, err := db.Open(cfg.DBFile)
	if err != nil {
		return errors.Wrapf(err, "opening DB instance")
	}

	var dataProxy db.DataServerProxy
	if remote {
		proxy, err := db.OpenRemoteDB(DB)
		if err != nil {
			return errors.Wrapf(err, "open remote DB instance")

		}
		dataProxy = proxy
	} else {
		proxy, err := db.OpenLocalProxy(DB)
		if err != nil {
			return errors.Wrapf(err, "opening local DB instance:")

		}
		dataProxy = proxy
	}
	ctx = context.WithValue(ctx, tellorCommon.DataProxyKey, dataProxy)
	ctx = context.WithValue(ctx, tellorCommon.DBContextKey, DB)
	return nil
}

// var GitTag string
// var GitHash string

// const versionMessage = `
//     The official Tellor cli tool %s (%s)
//     -----------------------------------------
// 	Website: https://tellor.io
// 	Github:  https://github.com/tellor-io/telliot
// `

var cli struct {
	Config   configPath  `required type:"existingfile" help:"path to config file"`
	Transfer transferCmd `cmd help:"Transfer tokens"`
	Approve  approveCmd  `cmd help:"Approve tokens"`
	Balance  balanceCmd  `cmd help:"Check the balance of an address"`
	Stake    stakeCmd    `cmd help:"Perform one of the stake operations"`
	Dispute  struct {
		New  newDisputeCmd `cmd help:"start a new dispute"`
		Vote voteCmd       `cmd "vote on a open dispute"`
		Show struct {
		} `cmd help:"show open disputes"`
	} `cmd help:"Perform commands related to disputes"`
	Dataserver dataserverCmd `cmd help:"launch only a dataserver instance"`
	Mine       mineCmd       `cmd help:"mine TRB and submit values"`
}

func main() {
	ctx := kong.Parse(&cli, kong.Name("Telliot"),
		kong.Description("The official Tellor cli tool"),
		kong.UsageOnError())
	err := ctx.Run(*ctx)
	ctx.FatalIfErrorf(err)
}
