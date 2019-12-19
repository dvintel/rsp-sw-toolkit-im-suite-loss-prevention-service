/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"github.com/edgexfoundry/app-functions-sdk-go/pkg/transforms"
	"github.com/intel/rsp-sw-toolkit-im-suite-loss-prevention-service/app/config"
	"github.com/intel/rsp-sw-toolkit-im-suite-loss-prevention-service/app/lossprevention"
	"github.com/intel/rsp-sw-toolkit-im-suite-loss-prevention-service/app/notification"
	"github.com/intel/rsp-sw-toolkit-im-suite-loss-prevention-service/app/webserver"
	"github.com/intel/rsp-sw-toolkit-im-suite-loss-prevention-service/pkg/camera"
	"github.com/intel/rsp-sw-toolkit-im-suite-loss-prevention-service/pkg/jsonrpc"
	"github.com/intel/rsp-sw-toolkit-im-suite-loss-prevention-service/pkg/sensor"
	"os"
	"strings"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/go-metrics"
	reporter "github.com/intel/rsp-sw-toolkit-im-suite-utilities/go-metrics-influxdb"
)

const (
	serviceKey               = "loss-prevention-service"
	inventoryEvent           = "inventory_event"
	sensorConfigNotification = "sensor_config_notification"
)

var (
	// Filter data by value descriptors (aka device resource name)
	valueDescriptors = []string{
		inventoryEvent,
		sensorConfigNotification,
	}
)

func main() {
	mConfigurationError := metrics.GetOrRegisterGauge("loss-prevention-service.Main.ConfigurationError", nil)

	// Load config variables
	err := config.InitConfig()
	fatalErrorHandler("unable to load configuration variables", err, &mConfigurationError)

	// Initialize metrics reporting
	initMetrics()

	setLoggingLevel(config.AppConfig.LoggingLevel)

	log.WithFields(log.Fields{
		"Method": "main",
		"Action": "Start",
	}).Info("Starting Loss Prevention Service...")

	go registerSubscribers()

	go sensor.QueryBasicInfoAllSensors()

	// Connect to EdgeX zeroMQ bus
	go receiveZMQEvents()

	if _, err := camera.SanityCheck(); err != nil {
		logrus.Errorf("error running camera sanity check: %v", err)
		logrus.Error("service will now exit...")
		os.Exit(-1)
	} else {
		logrus.Info("Camera sanity check was successful")
	}

	go camera.RecordVideoToDisk(config.AppConfig.VideoDevice, 10, "/tmp/testing", true)

	webserver.StartWebServer(config.AppConfig.Port)

	log.WithField("Method", "main").Info("Completed.")
}

func registerSubscribers() {
	// Register a subscriber to EdgeX notification service
	if config.AppConfig.EmailSubscribers == "" {
		return
	}

	emails := strings.Split(config.AppConfig.EmailSubscribers, ",")
	if err := notification.RegisterSubscriber(emails); err != nil {
		log.Fatalf("Unable to register subscriber in EdgeX: %s", err)
	}
}

func initMetrics() {
	// setup metrics reporting
	if config.AppConfig.TelemetryEndpoint != "" {
		go reporter.InfluxDBWithTags(
			metrics.DefaultRegistry,
			time.Second*10, //cfg.ReportingInterval,
			config.AppConfig.TelemetryEndpoint,
			config.AppConfig.TelemetryDataStoreName,
			"",
			"",
			nil,
		)
	}
}

func receiveZMQEvents() {
	//Initialized EdgeX apps functionSDK
	edgexSdk := &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
	if err := edgexSdk.Initialize(); err != nil {
		logrus.Errorf("SDK initialization failed: %v", err)
		os.Exit(-1)
	}

	if err := edgexSdk.SetFunctionsPipeline(
		transforms.NewFilter(valueDescriptors).FilterByValueDescriptor,
		processEvents,
	); err != nil {
		logrus.Errorf("SDK SetPipeline failed: %v\n", err)
		os.Exit(-1)
	}

	if err := edgexSdk.MakeItRun(); err != nil {
		logrus.Errorf("MakeItRun returned error: %v", err)
		os.Exit(-1)
	}
}

func processEvents(edgexcontext *appcontext.Context, params ...interface{}) (bool, interface{}) {
	if len(params) < 1 {
		return false, nil
	}

	event := params[0].(models.Event)
	if len(event.Readings) < 1 {
		return false, nil
	}

	for _, reading := range event.Readings {
		switch reading.Name {

		case inventoryEvent:
			logrus.Debugf("inventoryEvent data received: %s", strings.ReplaceAll(strings.ReplaceAll(reading.Value, "\\", ""), "\"", "'"))

			payload := new(lossprevention.DataPayload)
			if err := jsonrpc.Decode(reading.Value, payload, nil); err != nil {
				return false, err
			}

			if err := lossprevention.HandleDataPayload(edgexcontext, payload); err != nil {
				return false, err
			}

		case sensorConfigNotification:
			logrus.Debugf("Received sensor config notification:\n%s", strings.ReplaceAll(strings.ReplaceAll(reading.Value, "\\", ""), "\"", "'"))

			notif := new(jsonrpc.SensorConfigNotification)
			if err := jsonrpc.Decode(reading.Value, notif, nil); err != nil {
				return false, err
			}

			rsp := sensor.NewRSPFromConfigNotification(notif)
			sensor.UpdateRSP(rsp)

		default:
			logrus.Warnf("received unsupported event: %s", reading.Name)
		}
	}

	return false, nil
}
