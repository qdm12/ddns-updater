package env

import (
	"fmt"
	"os"

	"github.com/qdm12/ddns-updater/internal/database"

	"github.com/qdm12/golibs/admin"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/network"
)

// Env contains objects necessary to the main function.
// These are created at start and are needed to the top-level
// working management of the program.
type Env struct {
	stopCh chan struct{}
	Client network.Client
	Gotify admin.Gotify
	SQL    database.SQL
}

// Warn logs a message and sends a notification to the Gotify server.
func (e *Env) Warn(message interface{}) {
	s := fmt.Sprintf("%s", message)
	logging.Warn(s)
	if e.Gotify != nil {
		e.Gotify.Notify("Warning", 2, s)
	}
}

// CheckError logs an error and sends a notification to the Gotify server
// if the error is not nil.
func (e *Env) CheckError(err error) {
	if err == nil {
		return
	}
	s := err.Error()
	logging.Errorf(s)
	if e.Gotify != nil {
		e.Gotify.Notify("Error", 3, s)
	}
}

// FatalOnError calls Fatal if the error is not nil.
func (e *Env) FatalOnError(err error) {
	if err != nil {
		e.Fatal(err)
	}
}

// shutdown cleanly exits the program by closing all connections,
// databases and syncing the loggers.
func (e *Env) shutdown() (exitCode int) {
	defer logging.Sync()
	e.Client.Close()
	if e.SQL != nil {
		err := e.SQL.Close()
		if err != nil {
			logging.Err(err)
			exitCode = 1
		}
	}
	return exitCode
}

// ShutdownFromSignal logs a warning, sends a notification to Gotify and shutdowns
// the program cleanly when a OS level signal is received. It should be passed as a
// callback to a function which would catch such signal.
func (e *Env) ShutdownFromSignal(signal string) (exitCode int) {
	logging.Warnf("Program stopped with signal %s", signal)
	if e.Gotify != nil {
		e.Gotify.Notify("Program stopped", 1, "Caught OS signal "+signal)
	}
	return e.shutdown()
}

// Fatal logs an error, sends a notification to Gotify and shutdowns the program.
// It exits the program with an exit code of 1.
func (e *Env) Fatal(message interface{}) {
	s := fmt.Sprintf("%s", message)
	logging.Error(s)
	if e.Gotify != nil {
		e.Gotify.Notify("Fatal error", 4, s)
	}
	e.shutdown()
	os.Exit(1)
}
