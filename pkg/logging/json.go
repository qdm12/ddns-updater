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
			Level:   LEVELERROR.string(),
			Message: fmt.Sprintf("cannot make JSON (%s) for payload: %v", err, payload),
			Time:    time.Now(),
			NodeID:  nodeID,
		})
		return string(b)
	}
	return string(b)
}

func (logger *jsonLogger) fatal(message string, fargs ...interface{}) {
	fmt.Println(makeJSON(LEVELFATAL, logger.nodeID, message, fargs...))
	os.Exit(1)
}

func (logger *jsonLogger) error(message string, fargs ...interface{}) {
	if logger.level >= LEVELERROR {
		fmt.Println(makeJSON(LEVELERROR, logger.nodeID, message, fargs...))
	}
}

func (logger *jsonLogger) warn(message string, fargs ...interface{}) {
	if logger.level >= LEVELWARNING {
		fmt.Println(makeJSON(LEVELWARNING, logger.nodeID, message, fargs...))
	}
}

func (logger *jsonLogger) success(message string, fargs ...interface{}) {
	if logger.level >= LEVELSUCCESS {
		fmt.Println(makeJSON(LEVELSUCCESS, logger.nodeID, message, fargs...))
	}
}

func (logger *jsonLogger) info(message string, fargs ...interface{}) {
	if logger.level >= LEVELINFO {
		fmt.Println(makeJSON(LEVELINFO, logger.nodeID, message, fargs...))
	}
}
