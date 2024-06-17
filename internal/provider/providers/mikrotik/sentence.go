package mikrotik

import "fmt"

type sentence struct {
	word    string // word that begins with !
	tag     string
	pairs   []pair
	mapping map[string]string
}

type pair struct {
	key   string
	value string
}

func newSentence() *sentence {
	return &sentence{
		mapping: make(map[string]string),
	}
}

func (s *sentence) String() string {
	return fmt.Sprintf("%s @%s %#q", s.word, s.tag, s.pairs)
}
