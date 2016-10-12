package OuternetStats

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type StatsMessage struct {
	DeviceID      string
	TimeCollected time.Time // Ending timestamp for the collection period.
	PeriodSeconds int       // Number of seconds for the collection period.
	SNR_Avg       float64   // SNR average over the period.
	Packets_Total int64     // Packets transferred (total: success + error) over the period.
}

const (
	DATAUPLOAD_SERVER = "updates.stratux.me:9000"
	DATAUPLOAD_LISTEN = ":9000"
)

type StatsPoster struct {
	statsChannel chan StatsMessage
}

/*
	statsPoster().
	 Posts stats to remote server.
*/

func (s *StatsPoster) statsPoster(receiverLat, receiverLng float64) {
	s.statsChannel = make(chan StatsMessage, 1024)
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
				d := <-s.statsChannel // Blocks.
				json, _ := json.Marshal(&d)
				msg = string(json) + "\n"
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

func (s *StatsPoster) Send(sm StatsMessage) {
	s.statsChannel <- sm // Send to stats channel.
}

func NewStatsPoster() *StatsPoster {
	p := new(StatsPoster)
	go p.statsPoster(0, 0)
	return p
}

type StatsReceiver struct {
	listener net.Listener
}

func (s *StatsReceiver) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		m, err := reader.ReadBytes('\n')
		if err != nil {
			return // Disconnected, etc.
		}
		if len(m) == 0 {
			continue // Empty message.
		}

		// Message is ideally a JSON-encoded "StatsMessage."
		var msg StatsMessage
		err = json.Unmarshal(m, &msg)
		if err != nil {
			fmt.Printf("handleConnection(): invalid data '%s'\n", string(m))
			continue
		}

		//TODO:Processing, save in db.
		fmt.Printf("got a message successfully! %v\n", msg)
	}

}

/*
	statsReceiver().
	 Receive stats from Outernet receiver.
*/

func (s *StatsReceiver) statsReceiver() {
	ln, err := net.Listen("tcp", DATAUPLOAD_LISTEN)
	if err != nil {
		fmt.Printf("statsReceiver(): can't listen on '%s': %s\n", DATAUPLOAD_LISTEN, err.Error())
		return
	}
	s.listener = ln
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Printf("statsReceiver(): error accepting connection.\n")
			continue
		}
		go s.handleConnection(conn)
	}
}

func NewStatsReceiver() *StatsReceiver {
	p := new(StatsReceiver)
	go p.statsReceiver()
	return p
}
