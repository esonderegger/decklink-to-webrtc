# decklink-to-webrtc

A little demo using Pion and FFmpeg

The code in this repo is meant to include minimal versions of the following pieces for streaming from a local source to a web client via WebRTC:

- ffmpeg scripts to stream from an input source to RTP
- a server, written in go, that reads from RTP and exposes a WHEP-compatible endpoint
- an html/css/js page that requests the WHEP endpoint and plays the WebRTC stream in an HTML5 video element

## Running this code locally

Assuming you have golang installed, run these commands to compile `decklink-to-webrtc` to your `$GOPATH/bin` directory

```
mkdir $GOPATH/src/github.com/esonderegger
cd $GOPATH/src/github.com/esonderegger
git clone git@github.com:esonderegger/decklink-to-webrtc.git
cd decklink-to-webrtc
go install
```

Next, run one of the shell scripts in this repo to stream RTP audio to port 5003 and RTP video to port 5004. For example, to loop a movie using libx264, run:

```
sh loop_file_x264.sh '/home/myuser/Videos/my_movie.mov'
```

Finally, in a new terminal, start the server by running:

```
decklink-to-webrtc
```

And open `http://localhost:8090/stream1` in a web broswer.
