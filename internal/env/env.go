package env

import (
	"os"

	"github.com/qdm12/ddns-updater/internal/data"

	"github.com/qdm12/golibs/admin"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/network"
)

type Env interface {
	SetClient(client network.Client)
	SetGotify(gotify admin.Gotify)
	SetDB(db data.Database)
	Notify(priority int, messageArgs ...interface{})
	Info(messageArgs ...interface{})
	Warn(messageArgs ...interface{})
	CheckError(err error)
	FatalOnError(err error)
	ShutdownFromSignal(signal string) (exitCode int)
	Fatal(messageArgs ...interface{})
	Shutdown() (exitCode int)
}

func NewEnv(logger logging.Logger) Env {
	return &env{logger: logger}
}

// env contains objects necessary to the main function.
// These are created at start and are needed to the top-level
// working management of the program.
type env struct {
	logger logging.Logger
	client network.Client
	gotify admin.Gotify
	db     data.Database
}

func (e *env) SetClient(client network.Client) {
	e.client = client
}

func (e *env) SetGotify(gotify admin.Gotify) {
	e.gotify = gotify
}

func (e *env) SetDB(db data.Database) {
	e.db = db
}

// Notify sends a notification to the Gotify server.
func (e *env) Notify(priority int, messageArgs ...interface{}) {
	if e.gotify == nil {
		return
	}
	if err := e.gotify.Notify("DDNS Updater", priority, messageArgs...); err != nil {
		e.logger.Error(err)
	}
}

// Info logs a message and sends a notification to the Gotify server.
func (e *env) Info(messageArgs ...interface{}) {
	e.logger.Info(messageArgs...)
	e.Notify(1, messageArgs...)
}

// Warn logs a message and sends a notification to the Gotify server.
func (e *env) Warn(messageArgs ...interface{}) {
	e.logger.Warn(messageArgs...)
	e.Notify(2, messageArgs...)
}

// CheckError logs an error and sends a notification to the Gotify server
// if the error is not nil.
func (e *env) CheckError(err error) {
	if err == nil {
		return
	}
	s := err.Error()
	e.logger.Error(s)
	if len(s) > 100 {
		s = s[:100] + "..." // trim down message for notification
	}
	e.Notify(3, s)
}

// FatalOnError calls Fatal if the error is not nil.
func (e *env) FatalOnError(err error) {
	if err != nil {
		e.Fatal(err)
	}
}

// Shutdown cleanly exits the program by closing all connections,
// databases and syncing the loggers.
func (e *env) Shutdown() (exitCode int) {
	defer func() {
		if err := e.logger.Sync(); err != nil {
			exitCode = 99
		}
	}()
	if e.client != nil {
		e.client.Close()
	}
	if e.db != nil {
		if err := e.db.Close(); err != nil {
			e.logger.Error(err)
			return 1
		}
	}
	return 0
}

// ShutdownFromSignal logs a warning, sends a notification to Gotify and shutdowns
// the program cleanly when a OS level signal is received. It should be passed as a
// callback to a function which would catch such signal.
func (e *env) ShutdownFromSignal(signal string) (exitCode int) {
	e.logger.Warn("Program stopped with signal %q", signal)
	e.Notify(1, "Caught OS signal %q", signal)
	return e.Shutdown()
}

// Fatal logs an error, sends a notification to Gotify and shutdowns the program.
// It exits the program with an exit code of 1.
func (e *env) Fatal(messageArgs ...interface{}) {
	e.logger.Error(messageArgs...)
	e.Notify(4, messageArgs...)
	_ = e.Shutdown()
	os.Exit(1)
}
