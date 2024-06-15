package mikrotik

// UnknownReplyError records the sentence whose Word is unknown.
type UnknownReplyError struct {
	Sentence *sentence
}

func (err *UnknownReplyError) Error() string {
	return "unknown RouterOS reply word: " + err.Sentence.word
}
