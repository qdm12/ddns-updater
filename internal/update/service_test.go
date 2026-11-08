package update

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Service_SetPeriod(t *testing.T) {
	t.Parallel()

	service := &Service{
		period: time.Minute,
		reset:  make(chan struct{}, 1),
	}

	service.SetPeriod(30 * time.Second)
	assert.Equal(t, 30*time.Second, service.getPeriod())

	// A non positive period must be ignored.
	service.SetPeriod(0)
	assert.Equal(t, 30*time.Second, service.getPeriod())
	service.SetPeriod(-time.Second)
	assert.Equal(t, 30*time.Second, service.getPeriod())

	// A reset must have been signaled for the ticker to be reset.
	assert.Len(t, service.reset, 1)

	// Signaling a reset twice must not block, since a single pending
	// signal is enough for the run goroutine.
	service.signalReset()
	assert.Len(t, service.reset, 1)
}

func Test_Service_SetCooldown(t *testing.T) {
	t.Parallel()

	service := &Service{
		cooldown: 5 * time.Minute,
	}

	service.SetCooldown(time.Minute)
	assert.Equal(t, time.Minute, service.getCooldown())

	// A negative cooldown must be ignored.
	service.SetCooldown(-time.Second)
	assert.Equal(t, time.Minute, service.getCooldown())
}
