package main

import (
	"./OuternetTelemetry"
	"fmt"
)

func main() {
	ot, err := OuternetTelemetry.NewClient()
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
	}
	v, err := ot.GetStatus()
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
	}
	fmt.Printf("%v\n", v)
}
