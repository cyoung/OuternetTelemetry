package main

import (
	"./OuternetStats"
	"fmt"
	"time"
)

func main() {
	server := OuternetStats.NewStatsReceiver()
	for {
		time.Sleep(1 * time.Second)
		fmt.Printf("hi\n")
	}
}
