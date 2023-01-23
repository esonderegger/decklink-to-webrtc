package main

import (
	_ "embed"

	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"text/template"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type HtmlData struct {
	Css string
	Js  string
}

//go:embed index.html.tmpl
var indexHtml string

//go:embed index.css
var indexCss string

//go:embed index.js
var indexJs string

var opusTrack *webrtc.TrackLocalStaticRTP
var h264Track *webrtc.TrackLocalStaticRTP

func getPortFromEnv(key string, fallback int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	intVar, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return intVar
}

func readRtpWriteTrack(port int, track *webrtc.TrackLocalStaticRTP) {
	var packetCounter uint16 = 0
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = listener.Close(); err != nil {
			panic(err)
		}
	}()
	inboundRTPPacket := make([]byte, 1600) // UDP MTU
	rtpPacket := &rtp.Packet{}
	for {
		n, _, err := listener.ReadFrom(inboundRTPPacket)
		if err != nil {
			panic(fmt.Sprintf("error during read: %s", err))
		}

		if err = rtpPacket.Unmarshal(inboundRTPPacket[:n]); err != nil {
			panic(err)
		}

		if rtpPacket.SequenceNumber != packetCounter+1 {
			fmt.Printf("unexpected sequence number on port %d - got: %d, expected: %d\n", port, rtpPacket.SequenceNumber, packetCounter+1)
		}
		packetCounter = rtpPacket.SequenceNumber

		if _, err = track.Write(inboundRTPPacket[:n]); err != nil {
			if errors.Is(err, io.ErrClosedPipe) {
				// The peerConnection has been closed.
				return
			}
			panic(err)
		}
	}
}

// Read incoming RTCP packets
// Before these packets are returned they are processed by interceptors. For things
// like NACK this needs to be called.
func readRtcpFromSender(sender *webrtc.RTPSender) {
	rtcpBuf := make([]byte, 1500)
	for {
		if _, _, rtcpErr := sender.Read(rtcpBuf); rtcpErr != nil {
			return
		}
	}
}

func getStream1(w http.ResponseWriter, req *http.Request) {
	hd := HtmlData{indexCss, indexJs}
	t := template.Must(template.New("index").Parse(indexHtml))
	err := t.Execute(w, hd)
	if err != nil {
		panic(err)
	}
}

func postStream1(w http.ResponseWriter, req *http.Request) {
	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Problem reading request body", http.StatusBadRequest)
		return
	}

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  string(reqBody),
	}

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	rtpAudioSender, err := peerConnection.AddTrack(opusTrack)
	if err != nil {
		panic(err)
	}
	go readRtcpFromSender(rtpAudioSender)

	rtpVideoSender, err := peerConnection.AddTrack(h264Track)
	if err != nil {
		panic(err)
	}
	go readRtcpFromSender(rtpVideoSender)

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())

		if connectionState == webrtc.ICEConnectionStateFailed {
			if closeErr := peerConnection.Close(); closeErr != nil {
				panic(closeErr)
			}
		}
	})

	// Set the remote SessionDescription
	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	// Send the SDP to the client in the HTTP response
	fmt.Fprintf(w, peerConnection.LocalDescription().SDP)
}

func stream1(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		getStream1(w, req)
		return
	case "POST":
		postStream1(w, req)
		return
	default:
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}
}

func main() {
	opusPort := getPortFromEnv("OPUS_PORT", 5002)
	h264Port := getPortFromEnv("H264_PORT", 5004)
	whepPort := getPortFromEnv("WHEP_PORT", 8080)

	var err error

	opusTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
	if err != nil {
		panic(err)
	}

	h264Track, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if err != nil {
		panic(err)
	}

	go readRtpWriteTrack(opusPort, opusTrack)
	go readRtpWriteTrack(h264Port, h264Track)
	http.HandleFunc("/stream", stream1)
	port := ":" + strconv.Itoa(whepPort)
	fmt.Fprintf(os.Stdout, "Serving on http://localhost%s\n", port)
	http.ListenAndServe(port, nil)
}
