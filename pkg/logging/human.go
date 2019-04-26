package logging

import (
	"fmt"
	"log"
	"os"
)

type humanLogger struct {
	level  Level
	nodeID int
}

func createHumanLogger(level Level, nodeID int) humanLogger {
	return humanLogger{
		level:  level,
		nodeID: nodeID,
	}
}

func makeLine(level Level, nodeID int, message string, fargs ...interface{}) string {
	builtMessage := fmt.Sprintf("Node %d: %s", nodeID, fmt.Sprintf(message, fargs...))
	return level.formatHuman(builtMessage)
}

func (logger *humanLogger) fatal(message string, fargs ...interface{}) {
	log.Print(makeLine(logger.level, logger.nodeID, message, fargs...))
	os.Exit(1)
}

func (logger *humanLogger) error(message string, fargs ...interface{}) {
	if logger.level >= ErrorLevel {
		log.Print(makeLine(logger.level, logger.nodeID, message, fargs...))
	}
}

func (logger *humanLogger) warn(message string, fargs ...interface{}) {
	if logger.level >= WarningLevel {
		log.Print(makeLine(logger.level, logger.nodeID, message, fargs...))
	}
}

func (logger *humanLogger) success(message string, fargs ...interface{}) {
	if logger.level >= SuccessLevel {
		log.Print(makeLine(logger.level, logger.nodeID, message, fargs...))
	}
}

func (logger *humanLogger) info(message string, fargs ...interface{}) {
	if logger.level >= InfoLevel {
		log.Print(makeLine(logger.level, logger.nodeID, message, fargs...))
	}
}
