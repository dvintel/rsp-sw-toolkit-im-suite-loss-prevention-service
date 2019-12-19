/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/go-metrics"
	golog "log"
	"strings"
)

func fatalErrorHandler(message string, err error, errorGauge *metrics.Gauge) {
	if err != nil {
		if errorGauge != nil {
			(*errorGauge).Update(1)
		}
		log.WithFields(log.Fields{
			"Method": "main",
			"Error":  fmt.Sprintf("%+v", err),
		}).Fatal(message)
	}
}

func setLoggingLevel(loggingLevel string) {
	switch strings.ToLower(loggingLevel) {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "trace":
		log.SetLevel(log.TraceLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	// Not using filtered func (Info, etc ) so that message is always logged
	golog.Printf("Logging level set to %s\n", loggingLevel)
}
