package params

import (
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

func verifyListeningPort(listeningPort string) {
	value, err := strconv.Atoi(listeningPort)
	if err != nil {
		zap.S().Fatalf("listening port %s is not a valid integer", listeningPort)
	} else if value < 1 {
		zap.S().Fatalf("listening port %s cannot be lower than 1", listeningPort)
	} else if value < 1024 {
		if os.Geteuid() == 0 {
			zap.S().Warnf("listening port %s allowed to be in the reserved system ports range as you are running as root", listeningPort)
		} else if os.Geteuid() == -1 {
			zap.S().Warnf("listening port %s allowed to be in the reserved system ports range as you are running in Windows", listeningPort)
		} else {
			zap.S().Fatalf("listening port %s cannot be in the reserved system ports range (1 to 1023) when running without root", listeningPort)
		}
	} else if value > 65535 {
		zap.S().Fatalf("listening port %s cannot be higher than 65535", listeningPort)
	} else if value > 49151 {
		// dynamic and/or private ports.
		zap.S().Warnf("listening port %s is in the dynamic/private ports range (above 49151)", listeningPort)
	}
}

func verifyRootURL(rootURL string) {
	if strings.ContainsAny(rootURL, " .?~#") {
		zap.S().Fatalf("root URL %s contains invalid characters", rootURL)
	}
}
