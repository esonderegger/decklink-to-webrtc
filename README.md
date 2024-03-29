# decklink-to-webrtc

A little application using [Pion](https://pion.ly/) and [FFmpeg](https://ffmpeg.org/) to demonstrate an end-to-end workflow for sendng video from a server to a browser via [WebRTC](https://webrtc.org/)

The code in this repo is meant to include minimal versions of the following pieces for streaming from a local source to a web client via WebRTC:

- ffmpeg scripts to stream from an input source to RTP
- a server, written in go, that reads from RTP and exposes a [WHEP](https://www.ietf.org/id/draft-murillo-whep-01.html)-compatible endpoint
- an html/css/js page that requests the WHEP endpoint and plays the WebRTC stream in an HTML5 video element

Note: This repository is a bit of a misnomer. It was originally intended to only demonstrate [Decklink](https://www.blackmagicdesign.com/products/decklink) functionality, but it now includes scripts for looping over a file as well. The go server doesn't care what the original source of the video is - only that it can read RTP from the specified UDP ports.

## Running the server locally (using Docker)

To build the container image, run:

```
docker build --tag decklink-to-webrtc .
```

Then run it via:

```
docker run --network host -e OPUS_PORT=5002 -e H264_PORT=5004 -e WHEP_PORT=8080 decklink-to-webrtc
```

Note the `--network host` part. This is so the go server can listen to UDP ports on the host's network. If anyone has a better way of doing this, I'd love a suggestion for how to improve this.

## Running the server locally (natively)

Assuming you have golang installed, run these commands to compile `decklink-to-webrtc` to your `$GOPATH/bin` directory

```
mkdir $GOPATH/src/github.com/esonderegger
cd $GOPATH/src/github.com/esonderegger
git clone git@github.com:esonderegger/decklink-to-webrtc.git
cd decklink-to-webrtc
go install
```

Then start the server via:

```
OPUS_PORT=5002 H264_PORT=5004 WHEP_PORT=8080 decklink-to-webrtc
```

## Stream from a source to UPD ports using RTP

Next, run one of the shell scripts in this repo to stream RTP audio to port 5002 and RTP video to port 5004. For example, to loop a movie using libx264, run:

```
sh loop_file_x264.sh '/home/myuser/Videos/my_movie.mov'
```

## Testing the stream in a web browser

If all goes well, you should be able to open `http://localhost:8080/stream` in a web broswer and see the video source with minimal latency.
