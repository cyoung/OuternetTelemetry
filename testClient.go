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

	v2, err := ot.GetTransfers()
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
	}
	fmt.Printf("%v\n", v2)

	v3, err := ot.GetSignaling()
	if err != nil {
		fmt.Printf("err %s\n", err.Error())
	}
	fmt.Printf("%v\n", v3)
}
