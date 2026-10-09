package guac

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxBatch     = 32 << 10 // bytes of instructions per WebSocket frame
	wsWriteLimit = 15 * time.Second
	idleTimeout  = 60 * time.Second // guacd sends sync ~every frame; silence = dead
)

// Bridge relays between the browser and guacd until either side closes.
// guacamole-common-js needs whole instructions per frame and expects the
// tunnel UUID as the first internal ("0.") instruction.
func Bridge(ws *websocket.Conn, conn net.Conn, rd *Reader, uuid string) {
	var wmu sync.Mutex
	write := func(msg string) error {
		wmu.Lock()
		defer wmu.Unlock()
		_ = ws.SetWriteDeadline(time.Now().Add(wsWriteLimit))
		return ws.WriteMessage(websocket.TextMessage, []byte(msg))
	}
	defer ws.Close()
	defer conn.Close()

	if err := write(Encode("", uuid)); err != nil {
		return
	}

	done := make(chan struct{})
	go func() { // guacd -> browser
		defer close(done)
		var batch strings.Builder
		for {
			_ = conn.SetReadDeadline(time.Now().Add(idleTimeout))
			raw, _, err := rd.Read()
			if err != nil {
				_ = write(Encode("error", "Remote desktop closed the connection.", "519"))
				return
			}
			batch.WriteString(raw)
			if rd.Buffered() > 0 && batch.Len() < maxBatch {
				continue
			}
			if err := write(batch.String()); err != nil {
				return
			}
			batch.Reset()
		}
	}()

	ws.SetReadLimit(1 << 20)
	go func() { // browser -> guacd
		defer conn.Close()
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			s := string(msg)
			if strings.HasPrefix(s, "0.,") { // tunnel-internal: answer pings, never forward
				if strings.HasPrefix(s, "0.,4.ping,") {
					if write(s) != nil {
						return
					}
				}
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteLimit))
			if _, err := conn.Write(msg); err != nil {
				return
			}
		}
	}()
	<-done
}
