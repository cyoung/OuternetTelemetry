package main

import (
	"./OuternetTelemetry"
	"encoding/json"
	"fmt"
	"github.com/paulbellamy/ratecounter"
	"net/http"
	"strconv"
	"time"
)

const (
	DATAPOINTS = 60
)

var pps_counter = ratecounter.NewRateCounter(1 * time.Second) // Packets per second.

//http://www.jsgraphs.com/

type ChartData struct {
	Labels []string    `json:"labels"`
	Series [][]float64 `json:"series"`
}

func constructChartData(data []float64) ChartData {
	var ret ChartData
	ret.Series = make([][]float64, 1)
	for i := 0; i < DATAPOINTS; i++ {
		lbl := ""
		if i%10 == 0 {
			lbl = strconv.Itoa(i) + " sec"
		}
		ret.Labels = append(ret.Labels, lbl)
		sData := float64(0)
		if len(data) > i {
			sData = data[i]
		}
		ret.Series[0] = append(ret.Series[0], sData)
	}

	return ret
}

func handleSNRRequest(w http.ResponseWriter, r *http.Request) {
	ret := constructChartData(snr)

	json, _ := json.Marshal(&ret)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func handlePPSRequest(w http.ResponseWriter, r *http.Request) {
	var data []float64
	for i := 0; i < len(pps); i++ {
		data = append(data, float64(pps[i]))
	}
	ret := constructChartData(data)

	json, _ := json.Marshal(&ret)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func defaultServer(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./charts.html")
}

var snr []float64
var pps []int64

func TelemetryWatcher() {
	ticker := time.NewTicker(1 * time.Second)
	ot, err := OuternetTelemetry.NewClient()
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
		return
	}

	var packetCountLast int

	for {
		<-ticker.C
		v, err := ot.GetStatus()
		if err != nil {
			fmt.Printf("err: %s\n", err.Error())
			continue
		}
		snr = append(snr, v.Tuner.SNR)
		if len(snr) > DATAPOINTS {
			snr = snr[len(snr)-DATAPOINTS:]
		}

		// Count all packets, including "CRC_Err" packets.
		curPackets := v.Tuner.CRC_OK + v.Tuner.CRC_Err
		packetsSinceLast := curPackets - packetCountLast
		if packetCountLast == 0 && len(pps) == 0 {
			// This is the first count. We'll skip this measurement.
			packetCountLast = curPackets
			continue
		}
		packetCountLast = curPackets
		// Increment the counter.
		pps_counter.Incr(int64(packetsSinceLast))

		// Get counter rate and save in the data slice.
		pps = append(pps, pps_counter.Rate())
		if len(pps) > DATAPOINTS {
			pps = pps[len(pps)-DATAPOINTS:]
		}
	}
}

func main() {
	go TelemetryWatcher()
	http.HandleFunc("/snr", handleSNRRequest)
	http.HandleFunc("/pps", handlePPSRequest)
	http.HandleFunc("/", defaultServer)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("managementInterface ListenAndServe: %s\n", err.Error())
	}
	for {
		time.Sleep(1 * time.Second)
	}

}
