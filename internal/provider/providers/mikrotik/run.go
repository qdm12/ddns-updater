package mikrotik

func (c *client) Run(sentence ...string) (*reply, error) {
	for _, word := range sentence {
		c.writer.writeWord(word)
	}
	err := c.writer.endSentence()
	if err != nil {
		return nil, err
	}
	return c.reader.readReply()
}
