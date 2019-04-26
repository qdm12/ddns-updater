package logging

var gLogger *Logger

func init() {
	gLogger = &Logger{
		mode:  Default,
		json:  createJSONLogger(InfoLevel, 0),
		human: createHumanLogger(InfoLevel, 0),
	}
}

// SetGlobalLoggerMode sets the mode of the global logger
func SetGlobalLoggerMode(mode Mode) {
	gLogger.m.Lock()
	gLogger.mode = mode
	gLogger.m.Unlock()
}

// SetGlobalLoggerLevel sets the level of the global logger
func SetGlobalLoggerLevel(level Level) {
	gLogger.m.Lock()
	nodeID := gLogger.json.nodeID
	gLogger.json = createJSONLogger(level, nodeID)
	gLogger.human = createHumanLogger(level, nodeID)
	gLogger.m.Unlock()
}

// SetGlobalLoggerNodeID sets the node ID of the global logger
func SetGlobalLoggerNodeID(nodeID int) {
	gLogger.m.Lock()
	level := gLogger.json.level
	gLogger.json = createJSONLogger(level, nodeID)
	gLogger.human = createHumanLogger(level, nodeID)
	gLogger.m.Unlock()
}

// Fatal logs a message and exit the program
func Fatal(message string, fargs ...interface{}) {
	gLogger.m.RLock()
	defer gLogger.m.RUnlock()
	if gLogger.mode == JSON || gLogger.mode == Default {
		gLogger.json.fatal(message, fargs...)
	} else if gLogger.mode == Human {
		gLogger.human.fatal(message, fargs...)
	}
}

// Error logs an error message if the level of the logger is set to higher or equal to ErrorLevel
func Error(message string, fargs ...interface{}) {
	gLogger.m.RLock()
	defer gLogger.m.RUnlock()
	if gLogger.mode == JSON || gLogger.mode == Default {
		gLogger.json.error(message, fargs...)
	} else if gLogger.mode == Human {
		gLogger.human.error(message, fargs...)
	}
}

// Warn logs a warning message if the level of the logger is set to higher or equal to WarningLevel
func Warn(message string, fargs ...interface{}) {
	gLogger.m.RLock()
	defer gLogger.m.RUnlock()
	if gLogger.mode == JSON || gLogger.mode == Default {
		gLogger.json.warn(message, fargs...)
	} else if gLogger.mode == Human {
		gLogger.human.warn(message, fargs...)
	}
}

// Success logs a success message if the level of the logger is set to higher or equal to SuccessLevel
func Success(message string, fargs ...interface{}) {
	gLogger.m.RLock()
	defer gLogger.m.RUnlock()
	if gLogger.mode == JSON || gLogger.mode == Default {
		gLogger.json.success(message, fargs...)
	} else if gLogger.mode == Human {
		gLogger.human.success(message, fargs...)
	}
}

// Info logs a message if the level of the logger is set to higher or equal to InfoLevel
func Info(message string, fargs ...interface{}) {
	gLogger.m.RLock()
	defer gLogger.m.RUnlock()
	if gLogger.mode == JSON || gLogger.mode == Default {
		gLogger.json.info(message, fargs...)
	} else if gLogger.mode == Human {
		gLogger.human.info(message, fargs...)
	}
}