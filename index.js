const videoEl = document.getElementById("remote-video");
const muteToggleButton = document.getElementById("mute-toggle");

const pc = new RTCPeerConnection({
  iceServers: [
    {
      urls: "stun:stun.l.google.com:19302",
    },
  ],
});

async function startStream(localSdp) {
  const { pathname } = window.location;
  const response = await fetch(pathname, {
    method: "POST",
    headers: {
      "Content-Type": "application/sdp",
    },
    body: localSdp,
  });
  const remoteSdp = await response.text();
  try {
    const remoteDescription = { type: "answer", sdp: remoteSdp };
    pc.setRemoteDescription(new RTCSessionDescription(remoteDescription));
  } catch (e) {
    alert(e);
  }
}

pc.ontrack = function (event) {
  if (event.track.kind === "video") {
    videoEl.srcObject = event.streams[0];
  }
};

pc.oniceconnectionstatechange = (e) => console.log(pc.iceConnectionState, e);
pc.onicecandidate = (event) => {
  if (event.candidate === null) {
    const { sdp } = pc.localDescription;
    startStream(sdp);
  }
};

// Offer to receive both audio and video
pc.addTransceiver("audio", { direction: "recvonly" });
pc.addTransceiver("video", { direction: "recvonly" });
pc.createOffer()
  .then((d) => pc.setLocalDescription(d))
  .catch((e) => console.log(e));

function toggleMute() {
  if (videoEl.muted) {
    videoEl.muted = false;
    muteToggleButton.textContent = "Mute";
  } else {
    videoEl.muted = true;
    muteToggleButton.textContent = "Un-mute";
  }
}

muteToggleButton.addEventListener("click", toggleMute, false);
