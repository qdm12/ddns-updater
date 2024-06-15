package mikrotik

import (
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var (
	ErrLoginChallengeNoRet = errors.New("login challenge response has no ret field")
)

func (c *client) login(username, password string) error {
	reply, err := c.Run("/login", "=name="+username, "=password="+password)
	if err != nil {
		return err
	}
	ret, ok := reply.done.mapping["ret"]
	if !ok {
		// Login method post-6.43 one stage, cleartext and no challenge
		if reply.done != nil {
			return nil
		}
		return fmt.Errorf("%w", ErrLoginChallengeNoRet)
	}

	// Login method pre-6.43 two stages, challenge
	challenge, err := hex.DecodeString(ret)
	if err != nil {
		return fmt.Errorf("hex decoding challenge response ret field: %w", err)
	}

	response := challengeResponse(challenge, password)
	_, err = c.Run("/login", "=name="+username, "=response="+response)
	if err != nil {
		return err
	}

	return nil
}

func challengeResponse(challenge []byte, password string) string {
	hasher := md5.New() //nolint:gosec
	_, _ = hasher.Write([]byte{0})
	_, _ = io.WriteString(hasher, password)
	_, _ = hasher.Write(challenge)
	return fmt.Sprintf("00%x", hasher.Sum(nil))
}
