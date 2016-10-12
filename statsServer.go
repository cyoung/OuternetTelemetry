package main

import (
	"./OuternetStats"
	"time"
)

func main() {
	OuternetStats.NewStatsReceiver()
	for {
		time.Sleep(1 * time.Second)
	}
}
