package golurk

import "github.com/go-logr/logr"

var internalLogger = logr.Logger{}

func SetInternalLogger(logger logr.Logger) {
	internalLogger = logger.WithName("golurk")
}
