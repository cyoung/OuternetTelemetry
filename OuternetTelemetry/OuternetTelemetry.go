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
		/transfers response

<?xml version="1.0" encoding="UTF-8"?>
<response code="200">
	<streams>
		<stream>
			<pid>8191</pid>
			<transfers>
				<transfer>
					<carousel_id>2</carousel_id>
					<path>opaks/Michael_Jackson.html.tgz</path>
					<hash>632d336af4c0840b3e195786ac81b30d95d05f3b215eace71458a24f773e562f</hash>
					<block_count>2364</block_count>
					<block_received>867</block_received>
					<complete>no</complete>
				</transfer>
			</transfers>
		</stream>
	</streams>
</response>
*/

/*

		/status response

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

// /status response
type OuternetStatusResponse struct {
	Tuner OuternetStatusTuner `xml:"tuner"`
	// Ignore everything else.
}

// /transfers response
type OuternetTransferResponse struct {
	Streams OuternetStreams `xml:"streams"`
}

// /signaling/ response
type OuternetSignalingResponse struct {
	Streams OuternetStreams `xml:"streams"`
}

type OuternetStreams struct {
	Stream []OuternetStream `xml:"stream"`
}

type OuternetStream struct {
	PId       int               `xml:"pid"`
	Transfers OuternetTransfers `xml:"transfers"`
	Files     OuternetFiles     `xml:"files"`
}

type OuternetTransfers struct {
	Transfer []OuternetTransfer `xml:"transfer"`
}

type OuternetFiles struct {
	File []OuternetFile `xml:"file"`
}

/*
	<carousel_id>2</carousel_id>
	<path>opaks/Michael_Jackson.html.tgz</path>
	<hash>632d336af4c0840b3e195786ac81b30d95d05f3b215eace71458a24f773e562f</hash>
	<block_count>2364</block_count>
	<block_received>867</block_received>
	<complete>no</complete>

*/
type OuternetTransfer struct {
	CarouselID    int    `xml:"carousel_id"`
	Path          string `xml:"path"`
	Hash          string `xml:"hash"`
	BlockCount    int    `xml:"block_count"`
	BlockReceived int    `xml:"block_received"`
	Complete      string `xml:"complete"` // "yes" / "no"
}

/*
	<carousel_id>2</carousel_id>
	<path>opaks/Hillary_Clinton.html.tgz</path>
	<hash>ea4181e69f3648daa535adfd9a3b3a0cf48e88cdccfd05a28fb74ff6b535d334</hash>
	<size>0</size>
	<fec>ldpc:k=3675,n=4410,N1=2,seed=1000</fec>
*/
type OuternetFile struct {
	CarouselID int    `xml:"carousel_id"`
	Path       string `xml: "path"`
	Hash       string `xml:"hash"`
	Size       int    `xml:"size"`
	FEC        string `xml:"fec"`
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

func (o *OuternetTelemetry) getCommandResponse(cmd string, i interface{}) error {
	// Request data.
	_, err := o.sendCommand(cmd)
	if err != nil {
		return err
	}

	// Wait for response.
	m, err := o.reader.ReadBytes('\x00')
	if err != nil {
		return err
	}

	// Parse XML.
	err = xml.Unmarshal(m, i)
	if err != nil {
		return fmt.Errorf("GetStatus(): XML parse error.")
	}

	// Success.
	return nil
}

func (o *OuternetTelemetry) GetStatus() (OuternetStatusResponse, error) {
	var ret OuternetStatusResponse
	err := o.getCommandResponse("/status", &ret)
	return ret, err
}

func (o *OuternetTelemetry) GetTransfers() (OuternetTransferResponse, error) {
	var ret OuternetTransferResponse
	err := o.getCommandResponse("/transfers", &ret)
	return ret, err
}

func (o *OuternetTelemetry) GetSignaling() (OuternetSignalingResponse, error) {
	var ret OuternetSignalingResponse
	err := o.getCommandResponse("/signaling/", &ret)
	return ret, err
}
