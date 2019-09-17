package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tellor-io/TellorMiner/cli"
	tellorCommon "github.com/tellor-io/TellorMiner/common"
	"github.com/tellor-io/TellorMiner/config"
	"github.com/tellor-io/TellorMiner/contracts"
	"github.com/tellor-io/TellorMiner/contracts1"
	"github.com/tellor-io/TellorMiner/db"
	"github.com/tellor-io/TellorMiner/ops"
	"github.com/tellor-io/TellorMiner/rpc"
	"github.com/tellor-io/TellorMiner/util"
)

var mainLog = util.NewLogger("main", "Main")

func main() {
	//create os kill sig listener
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	//global config
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	//see what args are passed in
	cli := cli.GetFlags()

	//create a db instance
	db, err := db.Open(cfg.DBFile)
	if err != nil {
		log.Fatal(err)
	}

	//create an rpc client
	client, err := rpc.NewClient(cfg.NodeURL)
	if err != nil {
		log.Fatal(err)
	}

	//create an instance of the tellor master contract for on-chain interactions
	contractAddress := common.HexToAddress(cfg.ContractAddress)
	masterInstance, err := contracts.NewTellorMaster(contractAddress, client)
	transactorInstance, err := contracts1.NewTellorTransactor(contractAddress, client)
	if err != nil {
		log.Fatal(err)
	}

	var ds *ops.DataServerOps
	//var miner *ops.MinerOps
	var miner *ops.MiningMgr

	//create a context to use for ops
	ctx := context.WithValue(context.Background(), tellorCommon.DBContextKey, db)
	ctx = context.WithValue(ctx, tellorCommon.ClientContextKey, client)
	ctx = context.WithValue(ctx, tellorCommon.MasterContractContextKey, masterInstance)
	ctx = context.WithValue(ctx, tellorCommon.TransactorContractContextKey, transactorInstance)

	exitChannels := make([]*chan os.Signal, 0)

	if cli.Transfer {
		ops.Transfer(cli.ToAddress, cli.Amount, ctx)
	}else if cli.Deposit {
		ops.Deposit(ctx)
	}else if cli.Approve{
		ops.Approve(cli.ToAddress,cli.Amount,ctx)
	}else if cli.Dispute{
		ops.Dispute(cli.RequestId,cli.Timestamp,cli.MinerIndex,ctx)
	}else if cli.RequestStakingWithdraw{
		ops.RequestStakingWithdraw(ctx)
	}else if cli.WithdrawStake{
		ops.WithdrawStake(ctx)
	}else if cli.Vote{
		ops.Vote(cli.DisputeId,cli.SupportsDispute,ctx)
	}else {
		if cli.DataServer {
			ch := make(chan os.Signal)
			exitChannels = append(exitChannels, &ch)
			ds, err = ops.CreateDataServerOps(ctx, ch)
			if err != nil {
				log.Fatal(err)
			}
		}

		if cli.Miner {
			ch := make(chan os.Signal)
			exitChannels = append(exitChannels, &ch)
			//miner, err = ops.CreateMinerOps(ctx, ch)
			miner, err = ops.CreateMiningManager(ctx, ch, ops.NewSubmitter())
			if err != nil {
				log.Fatal(err)
			}
		}

		if ds != nil {
			//start the data server
			ds.Start(ctx)
		}

		if miner != nil {
			//start the miner after at least one cycle from the data server, if it's running
			if ds != nil {
				<-ds.Ready()
			}

			miner.Start(ctx)
		}

		//now we wait for kill sig
		<-c
		//and then notify exit channels
		for _, ch := range exitChannels {
			*ch <- os.Interrupt
		}
		cnt := 0
		for {
			cnt++
			start := time.Now()
			dsStopped := false
			minerStopped := false

			if ds != nil {
				dsStopped = !ds.Running
			} else {
				dsStopped = true
			}

			if miner != nil {
				minerStopped = !miner.Running
			} else {
				minerStopped = true
			}

			if !dsStopped && !minerStopped && cnt > 60 {
				mainLog.Warn("Taking longer than expected to operations. Waited %v so far", time.Now().Sub(start))
			} else if dsStopped && minerStopped {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
	mainLog.Info("Main shutdown complete")
}
