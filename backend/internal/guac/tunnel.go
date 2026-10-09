package guac

import (
	"bufio"
	"context"
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

// BridgeOpts tunes one session. Ctx cancelled with a cause (context.Cause)
// ends the session and shows the cause to the user.
type BridgeOpts struct {
	Ctx    context.Context
	MaxDur time.Duration                // hard ceiling on session length; 0 = none
	OnFile func(direction, name string) // "upload" / "download", for the audit trail
}

// Bridge relays between the browser and guacd until either side closes.
// guacamole-common-js needs whole instructions per frame and expects the
// tunnel UUID as the first internal ("0.") instruction.
func Bridge(ws *websocket.Conn, conn net.Conn, rd *Reader, uuid string, o BridgeOpts) {
	var wmu sync.Mutex
	write := func(msg string) error {
		wmu.Lock()
		defer wmu.Unlock()
		_ = ws.SetWriteDeadline(time.Now().Add(wsWriteLimit))
		return ws.WriteMessage(websocket.TextMessage, []byte(msg))
	}
	defer ws.Close()
	defer conn.Close()
	end := func(msg, code string) {
		_ = write(Encode("error", msg, code))
		conn.Close()
	}
	if o.MaxDur > 0 {
		t := time.AfterFunc(o.MaxDur, func() { end("Session time limit reached.", "776") })
		defer t.Stop()
	}
	if o.Ctx != nil {
		stop := context.AfterFunc(o.Ctx, func() { end(context.Cause(o.Ctx).Error(), "776") })
		defer stop()
	}
	onFile := o.OnFile
	if onFile == nil {
		onFile = func(string, string) {}
	}

	if err := write(Encode("", uuid)); err != nil {
		return
	}

	done := make(chan struct{})
	go func() { // guacd -> browser
		defer close(done)
		var batch strings.Builder
		for {
			_ = conn.SetReadDeadline(time.Now().Add(idleTimeout))
			raw, elems, err := rd.Read()
			if err != nil {
				_ = write(Encode("error", "Remote desktop closed the connection.", "519"))
				return
			}
			if len(elems) >= 4 && elems[0] == "file" {
				onFile("download", elems[3])
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
			if strings.Contains(s, "4.file,") {
				for _, name := range uploadNames(s) {
					onFile("upload", name)
				}
			}
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteLimit))
			if _, err := conn.Write(msg); err != nil {
				return
			}
		}
	}()
	<-done
}

// uploadNames extracts filenames from "file" instructions in a client frame.
func uploadNames(frame string) []string {
	var out []string
	rd := NewReader(bufio.NewReader(strings.NewReader(frame)))
	for {
		_, elems, err := rd.Read()
		if err != nil {
			return out
		}
		if len(elems) >= 4 && elems[0] == "file" {
			out = append(out, elems[3])
		}
	}
}
