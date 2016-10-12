package OuternetStats

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net"
	"time"
)

type StatsMessage struct {
	DeviceID      string
	ReceiverLat   float64
	ReceiverLng   float64
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
	ReceiverLat  float64
	ReceiverLng  float64
	statsChannel chan StatsMessage
}

/*
	statsPoster().
	 Posts stats to remote server.
*/

func (s *StatsPoster) statsPoster() {
	s.statsChannel = make(chan StatsMessage, 1024)
	msg := ""
	for {
		conn, err := net.Dial("tcp", DATAUPLOAD_SERVER)
		if err != nil {
			fmt.Printf("statsPoster(): error connecting to '%s'\n", DATAUPLOAD_SERVER)
			time.Sleep(5 * time.Second)
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
				conn.Close()
				time.Sleep(5 * time.Second)
				break // Reconnect, try sending message again.
			}
			msg = "" // Success, clear buffer.
		}
	}
}

func (s *StatsPoster) Send(sm StatsMessage) {
	sm.ReceiverLat = s.ReceiverLat
	sm.ReceiverLng = s.ReceiverLng
	s.statsChannel <- sm // Send to stats channel.
}

func NewStatsPoster(lat, lng float64) *StatsPoster {
	s := new(StatsPoster)
	s.ReceiverLat = lat
	s.ReceiverLng = lng
	go s.statsPoster()
	return s
}

type StatsReceiver struct {
	listener net.Listener
	db       *sql.DB
	dbChan   chan StatsMessage
}

func (s *StatsReceiver) handleConnection(conn net.Conn) {
	defer fmt.Printf("closed connection: %s\n", conn.RemoteAddr().String())
	defer conn.Close()
	reader := bufio.NewReader(conn)
	fmt.Printf("new connection: %s\n", conn.RemoteAddr().String())
	for {
		m, err := reader.ReadBytes('\n')
		if err != nil {
			break // Disconnected, etc.
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

		// Processing, save in db, etc.
		fmt.Printf("got a message successfully! %v\n", msg)
		s.dbChan <- msg
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

func (s *StatsReceiver) dbWriter() {
	s.dbChan = make(chan StatsMessage, 1024)

	// Last received message from the channel, which may not have been successfully inserted into the database.
	var msg StatsMessage
	unfinishedMessage := false

	for {
		// Connect to database.
		db, err := sql.Open("mysql", "root:@/outernet")
		if err != nil {
			fmt.Printf("dbWriter(): db connect error: %s\n", err.Error())
			time.Sleep(5 * time.Second)
			continue
		}
		s.db = db

		fmt.Printf("connected to db.\n")
		for {
			// Do we have a message that was already retrieved from the channel but not yet written to the database?
			//  If no, get one.
			if !unfinishedMessage {
				msg = <-s.dbChan
				unfinishedMessage = true
			}
			_, err := s.db.Exec(`INSERT INTO stats SET DeviceID=?, ReceiverLat=?, ReceiverLng=?, TimeCollected=?, PeriodSeconds=?, SNR_Avg=?, Packets_Total=?`,
				msg.DeviceID, msg.ReceiverLat, msg.ReceiverLng, msg.TimeCollected, msg.PeriodSeconds, msg.SNR_Avg, msg.Packets_Total)
			if err != nil {
				fmt.Printf("dbWriter(): error inserting stats row to db: %s\n", err.Error())
				s.db.Close()
				time.Sleep(5 * time.Second)
				break // Reconnect, try inserting data again.
			}
			unfinishedMessage = false // Success, don't attempt to re-insert current message.
		}
	}
}

func NewStatsReceiver() *StatsReceiver {
	p := new(StatsReceiver)

	go p.dbWriter()
	go p.statsReceiver()
	return p
}
