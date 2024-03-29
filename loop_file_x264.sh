ffmpeg -y \
  -vsync 0 -async 0 -re -fflags +genpts -stream_loop -1 \
  -i $1 \
  -map 0:a -vn \
  -f s16le \
  -c:a libopus -b:a 128000 -ssrc 1 -payload_type 111 \
  -f rtp -max_delay 0 -application lowdelay 'rtp://127.0.0.1:5002?pkt_size=1200' \
  -map 0:v -an \
  -pix_fmt yuv420p \
  -vf "settb=AVTB,setpts='trunc(PTS/1K)*1K+st(1,trunc(RTCTIME/1K))-1K*trunc(ld(1)/1K)',drawtext=fontfile=/usr/share/fonts/truetype/noto/NotoSansMono-Regular.ttf:fontsize=35:fontcolor=white:text=' Frame\:%{n}     Clock\:%{localtime}.%{eif\:1M*t-1K*trunc(t*1K)\:d}':x=600:y=4" \
  -vcodec libx264 -preset ultrafast \
  -profile:v main \
  -b:v 6M -maxrate 8M -bufsize 1M \
  -g 24 \
  -movflags +faststart \
  -tune zerolatency \
  -forced-idr 1 \
  -f rtp 'rtp://127.0.0.1:5004?pkt_size=1200'
