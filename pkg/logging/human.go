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
	log.Print(makeLine(LEVELFATAL, logger.nodeID, message, fargs...))
	os.Exit(1)
}

func (logger *humanLogger) error(message string, fargs ...interface{}) {
	if logger.level >= LEVELERROR {
		log.Print(makeLine(LEVELERROR, logger.nodeID, message, fargs...))
	}
}

func (logger *humanLogger) warn(message string, fargs ...interface{}) {
	if logger.level >= LEVELWARNING {
		log.Print(makeLine(LEVELWARNING, logger.nodeID, message, fargs...))
	}
}

func (logger *humanLogger) success(message string, fargs ...interface{}) {
	if logger.level >= LEVELSUCCESS {
		log.Print(makeLine(LEVELSUCCESS, logger.nodeID, message, fargs...))
	}
}

func (logger *humanLogger) info(message string, fargs ...interface{}) {
	if logger.level >= LEVELINFO {
		log.Print(makeLine(LEVELINFO, logger.nodeID, message, fargs...))
	}
}
