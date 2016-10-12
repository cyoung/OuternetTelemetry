package main

import (
	"./OuternetStats"
	"./OuternetTelemetry"
	"encoding/json"
	"fmt"
	"github.com/paulbellamy/ratecounter"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	DATAPOINTS = 1440
)

var datarate_counter = ratecounter.NewRateCounter(1 * time.Minute) // Packets per minute.

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
		if i%60 == 0 {
			lbl = strconv.Itoa(i/60) + " hour"
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

func handleDataRateRequest(w http.ResponseWriter, r *http.Request) {
	var data []float64
	for i := 0; i < len(datarate); i++ {
		data = append(data, float64(datarate[i]))
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
var datarate []int64

func avg(vals []float64) float64 {
	var total float64
	for _, v := range vals {
		total += v
	}
	return total / float64(len(vals))
}

var statsPoster *OuternetStats.StatsPoster

func TelemetryWatcher(lat, lng float64) {
	secondTicker := time.NewTicker(1 * time.Second)
	minuteTicker := time.NewTicker(50 * time.Second)

	// Create a new "Stats Poster" (goroutine that connects back to the server to send stats)
	statsPoster = OuternetStats.NewStatsPoster(lat, lng)

	ot, err := OuternetTelemetry.NewClient()
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
		return
	}

	var packetCountLast int

	var curSNR []float64

	for {
		v, err := ot.GetStatus()
		if err != nil {
			fmt.Printf("err: %s\n", err.Error())
			continue
		}
		select {
		case <-secondTicker.C:
			curSNR = append(curSNR, v.Tuner.SNR)
		case <-minuteTicker.C:
			// Average one-second SNR over the minute.
			avgSNROneMinute := avg(curSNR)
			curSNR = make([]float64, 0)
			snr = append(snr, avgSNROneMinute)
			if len(snr) > DATAPOINTS {
				snr = snr[len(snr)-DATAPOINTS:]
			}

			// Calculate packet rate over the last minute.

			// Count all packets, including "CRC_Err" packets.
			curPackets := v.Tuner.CRC_OK + v.Tuner.CRC_Err
			packetsSinceLast := curPackets - packetCountLast
			if packetCountLast == 0 && len(datarate) == 0 {
				// This is the first count. We'll skip this measurement.
				packetCountLast = curPackets
				continue
			}
			packetCountLast = curPackets
			// Increment the counter.
			datarate_counter.Incr(int64(packetsSinceLast))

			// Get counter rate and save in the data slice.
			oneMinuteDataRate := datarate_counter.Rate()
			datarate = append(datarate, oneMinuteDataRate)
			if len(datarate) > DATAPOINTS {
				datarate = datarate[len(datarate)-DATAPOINTS:]
			}

			// Construct a "StatsMessage" to send back to the server.
			sm := OuternetStats.StatsMessage{
				TimeCollected: time.Now(),
				PeriodSeconds: 50,
				SNR_Avg:       avgSNROneMinute,
				Packets_Total: oneMinuteDataRate,
			}
			statsPoster.Send(sm)
		}
	}
}

func main() {
	var lat, lng float64
	if len(os.Args) >= 3 {
		f, err := strconv.ParseFloat(os.Args[1], 64)
		if err != nil {
			fmt.Printf("invalid: '%s'.\n", os.Args[1])
			return
		}
		lat = f
		f, err = strconv.ParseFloat(os.Args[2], 64)
		if err != nil {
			fmt.Printf("invalid: '%s'.\n", os.Args[2])
			return
		}
		lng = f
	}
	go TelemetryWatcher(lat, lng)
	http.HandleFunc("/snr", handleSNRRequest)
	http.HandleFunc("/datarate", handleDataRateRequest)
	http.HandleFunc("/", defaultServer)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("managementInterface ListenAndServe: %s\n", err.Error())
	}
	for {
		time.Sleep(1 * time.Second)
	}

}
