// Package guac speaks the Guacamole protocol to guacd and bridges it to a
// browser WebSocket. Element lengths are counted in Unicode code points.
package guac

import (
	"bufio"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxElementLen = 8 << 20 // guard against a hostile/broken peer

// Encode builds one instruction, e.g. Encode("select", "rdp") = "6.select,3.rdp;".
func Encode(elems ...string) string {
	var b strings.Builder
	for i, e := range elems {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(utf8.RuneCountInString(e)))
		b.WriteByte('.')
		b.WriteString(e)
	}
	b.WriteByte(';')
	return b.String()
}

// Reader reads whole instructions from guacd.
type Reader struct{ r *bufio.Reader }

func NewReader(r *bufio.Reader) *Reader { return &Reader{r: r} }

// Buffered reports bytes already read from the socket but not yet consumed.
func (rd *Reader) Buffered() int { return rd.r.Buffered() }

// Read returns the raw instruction text (including the ';') and its elements.
func (rd *Reader) Read() (raw string, elems []string, err error) {
	var b strings.Builder
	for {
		n, err := rd.readLength(&b)
		if err != nil {
			return "", nil, err
		}
		var val strings.Builder
		for i := 0; i < n; i++ {
			r, _, err := rd.r.ReadRune()
			if err != nil {
				return "", nil, err
			}
			val.WriteRune(r)
		}
		b.WriteString(val.String())
		elems = append(elems, val.String())
		term, err := rd.r.ReadByte()
		if err != nil {
			return "", nil, err
		}
		b.WriteByte(term)
		switch term {
		case ';':
			return b.String(), elems, nil
		case ',':
			continue
		default:
			return "", nil, fmt.Errorf("guac: bad terminator %q", term)
		}
	}
}

func (rd *Reader) readLength(b *strings.Builder) (int, error) {
	n := 0
	digits := 0
	for {
		c, err := rd.r.ReadByte()
		if err != nil {
			return 0, err
		}
		b.WriteByte(c)
		if c == '.' {
			if digits == 0 {
				return 0, errors.New("guac: empty length")
			}
			return n, nil
		}
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("guac: bad length byte %q", c)
		}
		n = n*10 + int(c-'0')
		digits++
		if n > maxElementLen {
			return 0, errors.New("guac: element too long")
		}
	}
}
