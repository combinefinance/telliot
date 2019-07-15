package tracker

import (
	"context"
	"fmt"
	"time"

	"github.com/tellor-io/TellorMiner/config"
	"github.com/tellor-io/TellorMiner/rpc"
)

//Runner will execute all configured trackers
type Runner struct {
	client rpc.ETHClient
	//db db.DBClient
}

//NewRunner will create a new runner instance
func NewRunner(client rpc.ETHClient) (*Runner, error) {
	return &Runner{client: client}, nil
}

//Start will kick off the runner until the given exit channel selects.
func (r *Runner) Start(ctx context.Context, exitCh chan int) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}
	sleep := cfg.TrackerSleepCycle
	trackerNames := cfg.Trackers
	trackers := make([]Tracker, len(trackerNames))
	for i := 0; i < len(trackers); i++ {
		t, err := createTracker(trackerNames[i])
		if err != nil {
			fmt.Printf("Problem creating tracker: %s\n", err.Error())
		}
		trackers[i] = t
	}

	ticker := time.NewTicker(time.Duration(sleep) * time.Second)
	go func() {
		for {
			select {
			case _ = <-exitCh:
				{
					fmt.Println("Exiting run loop")
					ticker.Stop()
					return
				}
			case _ = <-ticker.C:
				{
					fmt.Println("Will run tracker queries", time.Now())
					c := context.WithValue(ctx, "client", r.client)
					for _, t := range trackers {
						err := t.Exec(c)
						if err != nil {
							fmt.Println("Problem in tracker", err)
						}
					}

				}
			}
		}
	}()

	return nil

}