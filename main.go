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
	"text/template"

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

var audioTrack *webrtc.TrackLocalStaticRTP
var videoTrack *webrtc.TrackLocalStaticRTP

func readRtpWriteTrack(port int, track *webrtc.TrackLocalStaticRTP) {
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
	for {
		n, _, err := listener.ReadFrom(inboundRTPPacket)
		if err != nil {
			panic(fmt.Sprintf("error during read: %s", err))
		}

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

	rtpAudioSender, err := peerConnection.AddTrack(audioTrack)
	if err != nil {
		panic(err)
	}
	go readRtcpFromSender(rtpAudioSender)

	rtpVideoSender, err := peerConnection.AddTrack(videoTrack)
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
	// Create audio and video tracks
	// If I were better at go, I would know how to avoid creating these temp variables and then using them to set the globals
	audioTrackTemp, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
	if err != nil {
		panic(err)
	}
	audioTrack = audioTrackTemp
	videoTrackTemp, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if err != nil {
		panic(err)
	}
	videoTrack = videoTrackTemp

	go readRtpWriteTrack(5003, audioTrack)
	go readRtpWriteTrack(5004, videoTrack)
	http.HandleFunc("/stream1", stream1)
	port := ":8090"
	fmt.Fprintf(os.Stdout, "Serving on http://localhost%s\n", port)
	http.ListenAndServe(port, nil)
}
