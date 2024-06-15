package mikrotik

import (
	"bufio"
	"io"
)

type writer struct {
	conn *bufio.Writer
	err  error
}

func newWriter(conn io.Writer) *writer {
	return &writer{
		conn: bufio.NewWriter(conn),
	}
}

// endSentence writes the end-of-sentence marker (an empty word).
// It returns the first error that occurred on calls to methods on w.
func (w *writer) endSentence() error {
	w.writeWord("")
	w.flush()
	return w.err
}

// writeWord writes one word.
func (w *writer) writeWord(word string) {
	b := []byte(word)
	w.write(encodeLength(len(b)))
	w.write(b)
}

func (w *writer) flush() {
	if w.err != nil {
		return
	}
	err := w.conn.Flush()
	if err != nil {
		w.err = err
	}
}

func (w *writer) write(b []byte) {
	if w.err != nil {
		return
	}
	_, err := w.conn.Write(b)
	if err != nil {
		w.err = err
	}
}

//nolint:gomnd
func encodeLength(l int) []byte {
	switch {
	case l < 0x80:
		return []byte{byte(l)}
	case l < 0x4000:
		return []byte{byte(l>>8) | 0x80, byte(l)}
	case l < 0x200000:
		return []byte{byte(l>>16) | 0xC0, byte(l >> 8), byte(l)}
	case l < 0x10000000:
		return []byte{byte(l>>24) | 0xE0, byte(l >> 16), byte(l >> 8), byte(l)}
	default:
		return []byte{0xF0, byte(l >> 24), byte(l >> 16), byte(l >> 8), byte(l)}
	}
}
