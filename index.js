const audioEl = document.getElementById("remote-audio");
const videoEl = document.getElementById("remote-video");
const startButton = document.getElementById("start");
let remoteSessionDescription = "";

const pc = new RTCPeerConnection({
  iceServers: [
    {
      urls: "stun:stun.l.google.com:19302",
    },
  ],
});

async function startStream(localSdp) {
  const response = await fetch("/start", {
    method: "POST",
    body: localSdp,
  });
  const respText = await response.text();
  remoteSessionDescription = respText;
}

function startSession() {
  audioEl.muted = false;
  if (remoteSessionDescription === "") {
    return alert("Session Description must not be empty");
  }

  try {
    const remoteDescription = JSON.parse(atob(remoteSessionDescription));
    pc.setRemoteDescription(new RTCSessionDescription(remoteDescription));
  } catch (e) {
    alert(e);
  }
}

pc.ontrack = function (event) {
  if (event.track.kind === "audio") {
    audioEl.srcObject = event.streams[0];
  }
  if (event.track.kind === "video") {
    videoEl.srcObject = event.streams[0];
  }
};

pc.oniceconnectionstatechange = (e) => console.log(pc.iceConnectionState, e);
pc.onicecandidate = (event) => {
  if (event.candidate === null) {
    const localSdp = btoa(JSON.stringify(pc.localDescription));
    startStream(localSdp);
  }
};

// Offer to receive 1 audio, and 1 video tracks
pc.addTransceiver("audio", { direction: "recvonly" });
pc.addTransceiver("video", { direction: "recvonly" });
pc.createOffer()
  .then((d) => pc.setLocalDescription(d))
  .catch((e) => console.log(e));

startButton.addEventListener("click", startSession, false);
