package logging

import (
	"sync"
)

// Logger is the structure for a logger instance and can contain multiple loggers
type Logger struct {
	mode Mode
	m sync.RWMutex
	json  jsonLogger
	human humanLogger
}

// CreateLogger returns the pointer to a Logger which can act as human 
// readable logger or a JSON formatted logger
func CreateLogger(mode Mode, level Level, nodeID int) *Logger {
	return &Logger{
		mode:  mode,
		json:  createJSONLogger(level, nodeID),
		human: createHumanLogger(level, nodeID),
	}
}

// ChangeMode changes the logging mode of the logger
func (logger *Logger) ChangeMode(mode Mode) {
	logger.m.Lock()
	logger.mode = mode
	logger.m.Unlock()
}

// Fatal logs a message and exit the program
func (logger *Logger) Fatal(message string, fargs ...interface{}) {
	logger.m.RLock()
	defer logger.m.RUnlock()
	if logger.mode == MODEJSON || logger.mode == MODEDEFAULT {
		logger.json.fatal(message, fargs...)
	} else if logger.mode == MODEHUMAN {
		logger.human.fatal(message, fargs...)
	}
}

// Error logs an error message if the level of the logger is set to higher or equal to ErrorLevel
func (logger *Logger) Error(message string, fargs ...interface{}) {
	logger.m.RLock()
	defer logger.m.RUnlock()
	if logger.mode == MODEJSON || logger.mode == MODEDEFAULT {
		logger.json.error(message, fargs...)
	} else if logger.mode == MODEHUMAN {
		logger.human.error(message, fargs...)
	}
}

// Warn logs a warning message if the level of the logger is set to higher or equal to WarningLevel
func (logger *Logger) Warn(message string, fargs ...interface{}) {
	logger.m.RLock()
	defer logger.m.RUnlock()
	if logger.mode == MODEJSON || logger.mode == MODEDEFAULT {
		logger.json.warn(message, fargs...)
	} else if logger.mode == MODEHUMAN {
		logger.human.warn(message, fargs...)
	}
}

// Success logs a success message if the level of the logger is set to higher or equal to SuccessLevel
func (logger *Logger) Success(message string, fargs ...interface{}) {
	logger.m.RLock()
	defer logger.m.RUnlock()
	if logger.mode == MODEJSON || logger.mode == MODEDEFAULT {
		logger.json.success(message, fargs...)
	} else if logger.mode == MODEHUMAN {
		logger.human.success(message, fargs...)
	}
}

// Info logs a message if the level of the logger is set to higher or equal to InfoLevel
func (logger *Logger) Info(message string, fargs ...interface{}) {
	logger.m.RLock()
	defer logger.m.RUnlock()
	if logger.mode == MODEJSON || logger.mode == MODEDEFAULT {
		logger.json.info(message, fargs...)
	} else if logger.mode == MODEHUMAN {
		logger.human.info(message, fargs...)
	}
}
