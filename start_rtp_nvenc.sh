# change for other resolution/frame rate combinations
my_format='Hp29'

ffmpeg -y \
  -format_code $my_format -f decklink -i 'DeckLink SDI' \
  -map 0:a -vn \
  -f s16le \
  -c:a libopus -b:a 48000 -ssrc 1 -payload_type 111 \
  -f rtp -max_delay 0 -application lowdelay 'rtp://127.0.0.1:5002?pkt_size=1200' \
  -map 0:v -an \
  -pix_fmt yuv420p \
  -c:v h264_nvenc -b:v 4M -g 15 -preset ll -tune zerolatency \
  -f rtp -deadline realtime 'rtp://127.0.0.1:5004?pkt_size=1200'
