package guac

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"time"
)

// Target is everything needed to open one RDP session.
type Target struct {
	IP         string // already validated by netguard
	Port       int
	Username   string
	Password   string
	Domain     string
	Security   string // any | nla | tls | rdp
	IgnoreCert bool
	Quality    string // high | balanced | low
	Width      int
	Height     int
	DPI        int
}

// Feature flags (env FEATURE_CLIPBOARD / FEATURE_FILE_TRANSFER).
type Features struct {
	ClipboardUpload   bool   // local → remote paste allowed
	ClipboardDownload bool   // remote → local copy allowed
	FileUpload        bool   // browser → "Transfer" drive
	FileDownload      bool   // drive → browser (files dropped in Transfer\Download)
	DrivePath         string // per-session directory inside guacd; "" = no drive
	RecordingPath     string // directory for the session recording; "" = not recorded
	RecordingName     string // file name (the recording id)
	RecordingKeys     bool   // include keystrokes (off: passwords stay out)
}

func (f Features) drive() bool { return f.DrivePath != "" && (f.FileUpload || f.FileDownload) }

// Dial connects to guacd and completes the RDP handshake. On success the
// returned reader holds any bytes guacd sent after "ready".
func Dial(ctx context.Context, guacdAddr string, t Target, f Features) (net.Conn, *Reader, string, error) {
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", guacdAddr)
	if err != nil {
		return nil, nil, "", fmt.Errorf("guacd unreachable: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	rd := NewReader(bufio.NewReaderSize(conn, 64<<10))
	id, err := handshake(conn, rd, t, f)
	if err != nil {
		conn.Close()
		return nil, nil, "", err
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, rd, id, nil
}

func handshake(conn net.Conn, rd *Reader, t Target, f Features) (string, error) {
	if _, err := conn.Write([]byte(Encode("select", "rdp"))); err != nil {
		return "", err
	}
	_, args, err := rd.Read()
	if err != nil {
		return "", fmt.Errorf("reading args: %w", err)
	}
	if len(args) == 0 || args[0] != "args" {
		return "", fmt.Errorf("guacd: expected args, got %v", first(args))
	}
	params := rdpParams(t, f)
	values := []string{"connect"}
	for _, name := range args[1:] {
		if len(name) > 8 && name[:8] == "VERSION_" {
			values = append(values, "VERSION_1_5_0")
			continue
		}
		values = append(values, params[name])
	}
	pre := Encode("size", strconv.Itoa(t.Width), strconv.Itoa(t.Height), strconv.Itoa(t.DPI)) +
		Encode("audio") + Encode("video") +
		Encode("image", "image/png", "image/jpeg", "image/webp") +
		Encode("timezone", "UTC")
	if _, err := conn.Write([]byte(pre + Encode(values...))); err != nil {
		return "", err
	}
	_, ready, err := rd.Read()
	if err != nil {
		return "", fmt.Errorf("reading ready: %w", err)
	}
	switch first(ready) {
	case "ready":
		if len(ready) > 1 {
			return ready[1], nil
		}
		return "", nil
	case "error":
		return "", fmt.Errorf("guacd error: %v", ready[1:])
	default:
		return "", fmt.Errorf("guacd: expected ready, got %v", first(ready))
	}
}

func rdpParams(t Target, f Features) map[string]string {
	b := func(v bool) string {
		if v {
			return "true"
		}
		return "false"
	}
	params := map[string]string{
		"hostname":          t.IP,
		"port":              strconv.Itoa(t.Port),
		"username":          t.Username,
		"password":          t.Password,
		"domain":            t.Domain,
		"security":          t.Security,
		"ignore-cert":       b(t.IgnoreCert),
		"width":             strconv.Itoa(t.Width),
		"height":            strconv.Itoa(t.Height),
		"dpi":               strconv.Itoa(t.DPI),
		"resize-method":     "display-update",
		"disable-copy":      b(!f.ClipboardDownload),
		"disable-paste":     b(!f.ClipboardUpload),
		"enable-drive":      b(f.drive()),
		"drive-name":        "Transfer",
		"client-name":       "WebRDP", // Windows shows the drive as "Transfer on WebRDP"
		"drive-path":        f.DrivePath,
		"create-drive-path": b(f.drive()),
		"disable-audio":     "true",
		"enable-printing":   "false",
		"server-layout":     "en-us-qwerty",
		"disable-download":  b(!f.FileDownload),
		"disable-upload":    b(!f.FileUpload),
	}
	if f.RecordingPath != "" {
		params["recording-path"] = f.RecordingPath
		params["recording-name"] = f.RecordingName
		params["create-recording-path"] = "true"
		params["recording-include-keys"] = b(f.RecordingKeys)
	}
	for k, v := range qualityParams(t.Quality) {
		params[k] = v
	}
	return params
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
