package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type jsonLogger struct {
	level  Level
	nodeID int
}

func createJSONLogger(level Level, nodeID int) jsonLogger {
	return jsonLogger{
		level:  level,
		nodeID: nodeID,
	}
}

func makeJSON(level Level, nodeID int, message string, fargs ...interface{}) string {
	type jsonPayload struct {
		Level   string `json:"level"`
		Message string `json:"message"`
		Time    time.Time `json:"time"` // generated on the fly
		NodeID  int `json:"node"`      // constant on the instance
	}
	payload := jsonPayload{
		Level:   level.string(),
		Message: fmt.Sprintf(message, fargs...),
		Time:    time.Now(),
		NodeID:  nodeID,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		b, _ := json.Marshal(&jsonPayload{
			Level:   ErrorLevel.string(),
			Message: fmt.Sprintf("cannot make JSON (%s) for payload: %v", err, payload),
			Time:    time.Now(),
			NodeID:  nodeID,
		})
		return string(b)
	}
	return string(b)
}

func (logger *jsonLogger) fatal(message string, fargs ...interface{}) {
	fmt.Println(makeJSON(FatalLevel, logger.nodeID, message, fargs...))
	os.Exit(1)
}

func (logger *jsonLogger) error(message string, fargs ...interface{}) {
	if logger.level >= ErrorLevel {
		fmt.Println(makeJSON(ErrorLevel, logger.nodeID, message, fargs...))
	}
}

func (logger *jsonLogger) warn(message string, fargs ...interface{}) {
	if logger.level >= WarningLevel {
		fmt.Println(makeJSON(WarningLevel, logger.nodeID, message, fargs...))
	}
}

func (logger *jsonLogger) success(message string, fargs ...interface{}) {
	if logger.level >= SuccessLevel {
		fmt.Println(makeJSON(SuccessLevel, logger.nodeID, message, fargs...))
	}
}

func (logger *jsonLogger) info(message string, fargs ...interface{}) {
	if logger.level >= InfoLevel {
		fmt.Println(makeJSON(InfoLevel, logger.nodeID, message, fargs...))
	}
}
