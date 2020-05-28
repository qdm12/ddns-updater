package update

import (
	"sync"
	"testing"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_newCycler(t *testing.T) {
	t.Parallel()
	ipMethods := []models.IPMethod{
		{Name: "a"}, {Name: "b"},
	}
	c := newCycler(ipMethods)
	require.NotNil(t, c)
	ipMethod := c.next()
	assert.Equal(t, ipMethod, models.IPMethod{Name: "a"})
}

func Test_next(t *testing.T) {
	t.Parallel()
	c := &cyclerImpl{
		methods: []models.IPMethod{
			{Name: "a"}, {Name: "b"},
		},
	}
	var m models.IPMethod
	m = c.next()
	assert.Equal(t, m, models.IPMethod{Name: "a"})
	m = c.next()
	assert.Equal(t, m, models.IPMethod{Name: "b"})
	m = c.next()
	assert.Equal(t, m, models.IPMethod{Name: "a"})
}

func Test_next_RaceCondition(t *testing.T) {
	// Run with -race flag
	t.Parallel()
	const workers = 5
	const loopSize = 101
	c := &cyclerImpl{
		methods: []models.IPMethod{
			{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"},
		},
	}
	ready := make(chan struct{})
	wg := &sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			<-ready
			for i := 0; i < loopSize; i++ {
				c.next()
			}
			wg.Done()
		}()
	}
	close(ready)
	wg.Wait()
	assert.Equal(t, (workers*loopSize)%len(c.methods), c.counter)
}
