package main

import (
	"./OuternetTelemetry"
	"encoding/json"
	"fmt"
	"github.com/paulbellamy/ratecounter"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	DATAPOINTS        = 1440
	DATAUPLOAD_SERVER = "updates.stratux.me:9000"
)

var datarate_counter = ratecounter.NewRateCounter(1 * time.Minute) // Packets per minute.

type StatsMessage struct {
	TimeCollected time.Time // Ending timestamp for the collection period.
	PeriodSeconds int       // Number of seconds for the collection period.
	SNR_Avg       float64   // SNR average over the period.
	Packets_Total int       // Packets transferred (total: success + error) over the period.
}

/*
	statsPoster().
	 Posts stats to remote server.
*/
var statsChannel chan StatsMessage

func statsPoster(receiverLat, receiverLng float64) {
	statsChannel = make(chan StatsMessage, 1024)
	var conn *net.Conn
	msg := ""
	for {
		conn, err := net.Dial("tcp", DATAUPLOAD_SERVER)
		if err != nil {
			fmt.Printf("statsPoster(): error connecting to '%s'\n", DATAUPLOAD_SERVER)
			time.Sleep(1 * time.Second)
			continue
		}
		fmt.Printf("statsPoster(): connected to '%s'\n", DATAUPLOAD_SERVER)
		for {
			// This is here to account for a the last message received from the channel but not yet sent because of a network error.
			if len(msg) == 0 {
				// Get the new stat item.
				d := <-statsChannel // Blocks.
				json, _ := json.Marshal(&d)
				msg = json + "\n"
			}

			_, err := conn.Write([]byte(msg))
			if err != nil {
				fmt.Printf("statsPoster(): write error: %s\n", err.Error())
				time.Sleep(1 * time.Second)
				break // Reconnect, try sending message again.
			}
			msg = "" // Success, clear buffer.
		}
	}
}

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

func TelemetryWatcher() {
	secondTicker := time.NewTicker(1 * time.Second)
	minuteTicker := time.NewTicker(50 * time.Second)
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
			datarate = append(datarate, datarate_counter.Rate())
			if len(datarate) > DATAPOINTS {
				datarate = datarate[len(datarate)-DATAPOINTS:]
			}
		}
	}
}

func main() {
	go TelemetryWatcher()
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
