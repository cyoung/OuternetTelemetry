package OuternetTelemetry

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"net"
)

const (
	IPC_SOCKPATH = "/var/run/ondd.ctrl"
)

type OuternetTelemetry struct {
	conn   net.Conn
	reader *bufio.Reader
}

/*
<?xml version="1.0" encoding="UTF-8"?>
<response code="200">
	<tuner>
		<lock>yes</lock>
		<freq>1539.87</freq>
		<freq_offset>39.81</freq_offset>
		<set_rs>4200</set_rs>
		<rssi>-106.99</rssi>
		<snr>3.10</snr>
		<ser>0.04</ser>
		<crc_ok>175558</crc_ok>
		<crc_err>727</crc_err>
		<alg_pk_mn>12.80</alg_pk_mn>
		<state>4</state>
	</tuner>
	<streams><stream>
	<pid>8191</pid>
	<bitrate>1800</bitrate>
	<ident>odc2</ident>
	</stream></streams>
</response>
*/

type OuternetStatusTuner struct {
	Lock        string  `xml:"lock"` // "yes".
	Freq        float64 `xml:"freq"`
	FreqOffset  float64 `xml:"freq_offset"`
	SetRS       int     `xml:"set_rs"`
	RSSI        float64 `xml:"rssi"`
	SNR         float64 `xml:"snr"`
	SER         float64 `xml:"ser"`
	CRC_OK      int     `xml:"crc_ok"`
	CRC_Err     int     `xml:"crc_err"`
	AlgPeakMean float64 `xml:"alg_pk_mn"`
	State       int     `xml:"state"`
}
type OuternetStatus struct {
	Tuner OuternetStatusTuner `xml:"tuner"`
	// Ignore everything else.
}

func NewClient() (*OuternetTelemetry, error) {
	conn, err := net.Dial("unix", IPC_SOCKPATH)
	if err != nil {
		return nil, fmt.Errorf("NewClient(): %s", err.Error())
	}
	return &OuternetTelemetry{conn: conn, reader: bufio.NewReader(conn)}, nil
}

func (o *OuternetTelemetry) sendCommand(cmd string) (int, error) {
	c := fmt.Sprintf("<get uri=\"%s\" />\x00", cmd)
	return o.conn.Write([]byte(c))
}

func (o *OuternetTelemetry) GetStatus() (OuternetStatus, error) {
	var ret OuternetStatus

	// Request status.
	_, err := o.sendCommand("/status")
	if err != nil {
		return ret, err
	}

	// Wait for response.
	m, err := o.reader.ReadBytes('\x00')
	if err != nil {
		return ret, err
	}

	// Parse XML.
	err = xml.Unmarshal(m, &ret)
	if err != nil {
		return ret, fmt.Errorf("GetStatus(): XML parse error.")
	}

	// Success.
	return ret, nil

}
