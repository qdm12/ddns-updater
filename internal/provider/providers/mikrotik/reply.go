package mikrotik

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

type reply struct {
	sentences []*sentence
	done      *sentence
}

func (r *reply) ingestSentence(sentence *sentence) (done bool, err error) {
	switch sentence.word {
	case "!re":
		r.sentences = append(r.sentences, sentence)
	case "!done":
		r.done = sentence
		return true, nil
	case "!trap", "!fatal":
		done = sentence.word == "!fatal"
		message := sentence.mapping["message"]
		if message == "" {
			err = fmt.Errorf("%w: unknown error: %s", errors.ErrUnsuccessful, sentence)
		} else {
			err = fmt.Errorf("%w: %s", errors.ErrUnsuccessful, message)
		}
		return done, err
	case "":
		// empty sentences should be ignored
	default:
		return true, fmt.Errorf("%w: word %q",
			errors.ErrUnknownResponse, sentence.word)
	}
	return false, nil
}
