package mikrotik

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

type reader struct {
	conn *bufio.Reader
}

func newReader(conn io.Reader) *reader {
	return &reader{
		conn: bufio.NewReader(conn),
	}
}

func (r *reader) readReply() (rep *reply, err error) {
	rep = &reply{}

	var lastErr error
	for {
		sentence, err := r.readSentence()
		if err != nil {
			return nil, fmt.Errorf("reading sentence: %w", err)
		}
		done, err := rep.ingestSentence(sentence)
		if err != nil {
			err = fmt.Errorf("ingesting sentence: %w", err)
			lastErr = err
		}
		if done {
			return rep, lastErr
		}
	}
}

func (r *reader) readSentence() (*sentence, error) {
	sentence := newSentence()
	for {
		b, err := r.readWord()
		if err != nil {
			return nil, err
		} else if len(b) == 0 {
			return sentence, nil
		}
		// Ex.: !re, !done
		if sentence.word == "" {
			sentence.word = string(b)
			continue
		}
		// Command tag
		if bytes.HasPrefix(b, []byte(".tag=")) {
			sentence.tag = string(b[5:])
			continue
		}
		// Ex.: =key=value, =key
		if bytes.HasPrefix(b, []byte("=")) {
			t := bytes.SplitN(b[1:], []byte("="), 2) //nolint:gomnd
			if len(t) == 1 {
				t = append(t, []byte{})
			}
			p := pair{string(t[0]), string(t[1])}
			sentence.pairs = append(sentence.pairs, p)
			sentence.mapping[p.key] = p.value
			continue
		}
		return nil, fmt.Errorf("%w: word %#q",
			errors.ErrUnknownResponse, b)
	}
}

func (r *reader) readNumber(size int) (int64, error) {
	b := make([]byte, size)
	_, err := io.ReadFull(r.conn, b)
	if err != nil {
		return -1, err
	}
	var num int64
	for _, ch := range b {
		num = num<<8 | int64(ch) //nolint:gomnd
	}
	return num, nil
}

//nolint:gomnd
func (r *reader) readLength() (int64, error) {
	l, err := r.readNumber(1)
	if err != nil {
		return -1, err
	}
	var n int64
	switch {
	case l&0x80 == 0x00:
	case (l & 0xC0) == 0x80:
		n, err = r.readNumber(1)
		l = l & ^0xC0 << 8 | n
	case l&0xE0 == 0xC0:
		n, err = r.readNumber(2)
		l = l & ^0xE0 << 16 | n
	case l&0xF0 == 0xE0:
		n, err = r.readNumber(3)
		l = l & ^0xF0 << 24 | n
	case l&0xF8 == 0xF0:
		l, err = r.readNumber(4)
	}
	if err != nil {
		return -1, err
	}
	return l, nil
}

func (r *reader) readWord() ([]byte, error) {
	l, err := r.readLength()
	if err != nil {
		return nil, err
	}
	b := make([]byte, l)
	_, err = io.ReadFull(r.conn, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
