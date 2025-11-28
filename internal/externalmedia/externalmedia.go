package externalmedia

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/charmbracelet/log"
	"github.com/pion/rtp"
)

// ExternalMediaParams contains parameters for external media
type ExternalMediaParams struct {
	ARIBaseURL string //ex : http://localhost:8088
	Username   string
	Password   string

	AppName string //ex : "myapp"
	HostIP  string //ex : "127.0.0.1"
	Port    int    //ex : 4002
	Format  string //ex : "slin16"
}

// ExternalMediaChannel represents the created channel + the UDP socket bound correctly
type ExternalMediaChannel struct {
	ID            string
	AsteriskRTP   *net.UDPAddr // Where to send the packets (retrieved dynamically)
	UDPConn       *net.UDPConn // Socket already bound on HostIP:Port
	SSRC          uint32
	Sequence      uint16
	Timestamp     uint32
	SamplesPerPkt uint32 // 160 bytes for ulaw/alaw, 320 bytes for slin16 (20ms)
	PayloadType   uint8
}

type ExternalMediaResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	State       string `json:"state"`
	ChannelVars struct {
		RTPAddress string `json:"UNICASTRTP_LOCAL_ADDRESS"`
		RTPPort    string `json:"UNICASTRTP_LOCAL_PORT"`
	} `json:"channelvars"`
}

// CreateExternalMedia creates the externalmedia channel on asterisk server
func CreateExternalMedia(p ExternalMediaParams) (*ExternalMediaResponse, error) {
	endpoint := fmt.Sprintf("%s/ari/channels/externalMedia", p.ARIBaseURL)
	log.Info("Creating External Media Channel", "endpoint", endpoint)

	params := url.Values{}
	params.Set("app", p.AppName)
	params.Set("external_host", fmt.Sprintf("%s:%d", p.HostIP, p.Port))
	params.Set("format", p.Format)

	reqURL := endpoint + "?" + params.Encode()
	log.Info("External Media Request URL", "url", reqURL)
	req, err := http.NewRequest("POST", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(p.Username, p.Password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("asterisk returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	//Parse JSON response
	var parsed ExternalMediaResponse
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

// CreateExternalMediaChannel creates the channel AND opens the UDP socket correctly bound
func CreateExternalMediaChannel(p ExternalMediaParams) (*ExternalMediaChannel, error) {
	// 1. Create the ExternalMedia channel on Asterisk
	resp, err := CreateExternalMedia(p)
	if err != nil {
		return nil, fmt.Errorf("failed to create external media channel: %w", err)
	}
	// 2. Open the UDP socket bound EXACTLY to HostIP:Port (CRUCIAL!)
	localAddr := &net.UDPAddr{
		IP:   net.ParseIP(p.HostIP),
		Port: p.Port,
	}
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("cannot bind UDP on %s:%d: %w", p.HostIP, p.Port, err)
	}
	log.Info("ExternalMedia channel created", "id", resp.ID, "bound", localAddr)

	//3. Prepare RTP variables
	ssrcBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.MaxUint32)))
	ssrc := uint32(ssrcBig.Int64())
	seqBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.MaxUint16)))
	seq := uint16(seqBig.Int64())
	tsBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.MaxUint32)))
	ts := uint32(tsBig.Int64())

	// According to the format
	var samplesPerPkt uint32
	var pt uint8
	switch p.Format {
	case "ulaw":
		samplesPerPkt = 160 // 20ms @ 8kHz
		pt = 0
	case "alaw":
		samplesPerPkt = 160
		pt = 8
	case "slin16":
		samplesPerPkt = 320 // 20ms @ 16kHz
		pt = 96             // dynamic, but pion/rtp accepts it
	default:
		conn.Close()
		return nil, fmt.Errorf("format %s not supported", p.Format)
	}
	return &ExternalMediaChannel{
		ID:            resp.ID,
		UDPConn:       conn,
		SSRC:          ssrc,
		Sequence:      seq,
		Timestamp:     ts,
		SamplesPerPkt: samplesPerPkt,
		PayloadType:   pt,
	}, nil

}

// SendRawPCMAsRTP sends a linear16 PCM (raw) buffer to addr (ip:port) in 20ms RTP packets.
// pcm: bytes (little-endian signed 16-bit). sampleRate: e.g., 16000
// addr: "192.168.122.113:19324" (the address/port returned by Asterisk in UNICASTRTP_LOCAL_*)
// ctx: allows cancelling the stream (cancelling ctx stops the send).
//
//	func SendRawPCMAsRTP(ctx context.Context, pcm []byte, addr string, sampleRate int) error {
//		if sampleRate <= 0 {
//			return errors.New("sampleRate must be > 0")
//		}
//		//samples /frame for 20ms
//		samplesPerFrame := sampleRate * 20 / 1000
//		if samplesPerFrame <= 0 {
//			return fmt.Errorf("invalid samplesPerFrame computed : %d ", samplesPerFrame)
//		}
//		bytesPerFrame := samplesPerFrame * 2 // 2 bytes per sample (16-bit)
//		if bytesPerFrame <= 0 {
//			return fmt.Errorf("invalid bytesPerFrame computed : %d ", bytesPerFrame)
//		}
//
//		//Count frames (Ceil)
//		numFrames := int(math.Ceil(float64(len(pcm))) / float64(bytesPerFrame))
//
//		//Resolve UDP addr
//		updAddr, err := net.ResolveUDPAddr("udp", addr)
//		if err != nil {
//			return fmt.Errorf("error resolving UDP addr %s: %w", addr, err)
//		}
//		conn, err := net.DialUDP("udp", nil, updAddr)
//		if err != nil {
//			return fmt.Errorf("error dialing UDP addr %s: %w", addr, err)
//		}
//		defer conn.Close()
//
//		//Create RTP stream params
//		//SSRC random
//		ssrcBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.MaxUint32)))
//		ssrc := uint32(ssrcBig.Int64())
//
//		//sequence start random
//		seqStartBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.MaxUint16)))
//		sequence := uint16(seqStartBig.Int64())
//
//		//timestamp start random
//		tsBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.MaxUint32)))
//		timestamp := uint32(tsBig.Int64())
//
//		// dynamic payload type (common usage), Asterisk will interpret the bytes as slin16
//		payloadType := uint8(96)
//
//		// Ticker for 20ms cadence
//		ticker := time.NewTicker(20 * time.Millisecond)
//		defer ticker.Stop()
//		// send loop
//		offset := 0
//		for frame := 0; frame < numFrames; frame++ {
//			//respect ctx cancellation
//			select {
//			case <-ctx.Done():
//				return ctx.Err()
//			default:
//
//			}
//			//build the payload
//			end := offset + bytesPerFrame
//			var payload []byte
//			if end <= len(pcm) {
//				payload = pcm[offset:end]
//			} else {
//				// last chunk: we pad with zeros
//				payload = make([]byte, bytesPerFrame)
//				copy(payload, pcm[offset:len(pcm)])
//			}
//			//Send RTP packet
//			packet := &rtp.Packet{
//				Header: rtp.Header{
//					Version:        2,
//					Padding:        false,
//					Extension:      false,
//					Marker:         false,
//					PayloadType:    payloadType,
//					SequenceNumber: sequence,
//					Timestamp:      timestamp,
//					SSRC:           ssrc,
//				},
//				Payload: payload,
//			}
//
//			buf, err := packet.Marshal()
//			if err != nil {
//				return fmt.Errorf("error marshalling RTP packet: %w", err)
//			}
//
//			// Send it
//			intVal, err := conn.Write(buf)
//			if err != nil {
//				return fmt.Errorf("error sending RTP packet: %w", err)
//			}
//			log.Debug("Sent RTP packet", "bytes", intVal, "seq", sequence, "ts", timestamp)
//
//			//increment for next packet
//			sequence++
//			timestamp += uint32(samplesPerFrame)
//			offset += bytesPerFrame
//			// wait for the next tick, while supporting ctx cancelation
//			select {
//			case <-ctx.Done():
//				return ctx.Err()
//			case <-ticker.C:
//			}
//
//		}
//
//		// Optional: send a short final silence to avoid abrupt cutoff (send 50ms of silence)
//		// We send 2 silence packets (40ms) for smoothing
//		silencePackets := 2
//		zeroPayload := make([]byte, bytesPerFrame)
//		for i := 0; i < silencePackets; i++ {
//			packet := &rtp.Packet{
//				Header: rtp.Header{
//					Version:        2,
//					Padding:        false,
//					Extension:      false,
//					Marker:         false,
//					PayloadType:    payloadType,
//					SequenceNumber: sequence,
//					Timestamp:      timestamp,
//					SSRC:           ssrc,
//				},
//				Payload: zeroPayload,
//			}
//			buf, _ := packet.Marshal()
//			_, err = conn.Write(buf)
//			if err != nil {
//				// we try to continue but return the error
//				return fmt.Errorf("udp write silence: %w", err)
//			}
//			sequence++
//			timestamp += uint32(samplesPerFrame)
//			select {
//			case <-ctx.Done():
//				return ctx.Err()
//			case <-time.After(20 * time.Millisecond):
//			}
//		}
//
//		return nil
//	}
func (ch *ExternalMediaChannel) SendPCM(ctx context.Context, pcm []byte) error {
	if ch.AsteriskRTP == nil {
		return errors.New("AsteriskRTP address not known yet, call WaitForAsteriskRTP first")
	}

	samplesPerFrame := int(ch.SamplesPerPkt)
	bytesPerFrame := samplesPerFrame * 2 // 16-bit

	numFrames := int(math.Ceil(float64(len(pcm)) / float64(bytesPerFrame)))
	offset := 0
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for i := 0; i < numFrames; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		end := offset + bytesPerFrame
		var payload []byte
		if end <= len(pcm) {
			payload = pcm[offset:end]
		} else {
			payload = make([]byte, bytesPerFrame)
			copy(payload, pcm[offset:])
		}

		packet := &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				PayloadType:    ch.PayloadType,
				SequenceNumber: ch.Sequence,
				Timestamp:      ch.Timestamp,
				SSRC:           ch.SSRC,
			},
			Payload: payload,
		}

		data, _ := packet.Marshal()
		_, err := ch.UDPConn.WriteToUDP(data, ch.AsteriskRTP)
		if err != nil {
			return err
		}

		ch.Sequence++
		ch.Timestamp += ch.SamplesPerPkt
		offset += bytesPerFrame
	}
	// Short final silence to avoid abrupt cutoff
	for i := 0; i < 3; i++ {
		time.Sleep(20 * time.Millisecond)
		silence := make([]byte, bytesPerFrame)
		packet := &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				PayloadType:    ch.PayloadType,
				SequenceNumber: ch.Sequence,
				Timestamp:      ch.Timestamp,
				SSRC:           ch.SSRC,
			},
			Payload: silence,
		}
		data, _ := packet.Marshal()
		ch.UDPConn.WriteToUDP(data, ch.AsteriskRTP)
		ch.Sequence++
		ch.Timestamp += ch.SamplesPerPkt
	}

	return nil
}

// Close releases the socket
func (ch *ExternalMediaChannel) Close() {
	if ch.UDPConn != nil {
		ch.UDPConn.Close()
	}
}

func (ch *ExternalMediaChannel) WaitForAsteriskRTP(timeout time.Duration) error {
	buf := make([]byte, 1500)
	ch.UDPConn.SetReadDeadline(time.Now().Add(timeout))

	n, remote, err := ch.UDPConn.ReadFromUDP(buf)
	if err != nil {
		return fmt.Errorf("timeout or error while waiting for Asterisk: %w", err)
	}
	log.Info("First RTP packet received from Asterisk", "from", remote, "bytes", n)
	ch.AsteriskRTP = remote
	return nil
}
