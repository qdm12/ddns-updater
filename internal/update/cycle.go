package update

import (
	"sync"

	"github.com/qdm12/ddns-updater/internal/models"
)

type cycler interface {
	next() models.IPMethod
}

type cyclerImpl struct {
	sync.Mutex
	counter int
	methods []models.IPMethod
}

func newCycler(methods []models.IPMethod) cycler {
	return &cyclerImpl{
		methods: methods,
	}
}

func (c *cyclerImpl) next() models.IPMethod {
	c.Lock()
	defer c.Unlock()
	method := c.methods[c.counter]
	c.counter++
	if c.counter == len(c.methods) {
		c.counter = 0
	}
	return method
}
