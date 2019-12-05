/*
 * INTEL CONFIDENTIAL
 * Copyright (2019) Intel Corporation.
 *
 * The source code contained or described herein and all documents related to the source code ("Material")
 * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
 * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
 * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
 * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
 * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
 * any way without Intel/'s prior express written permission.
 * No license under any patent, copyright, trade secret or other intellectual property right is granted
 * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
 * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
 * and approved by Intel in writing.
 * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
 * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
 */

package main

import (
	"github.com/edgexfoundry/app-functions-sdk-go/pkg/transforms"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/lossprevention"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/notification"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/webserver"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/pkg/camera"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/pkg/sensor"
	"os"
	"strings"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	reporter "github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics-influxdb"
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

	go camera.RecordVideoToDisk(config.AppConfig.VideoDevice, 10, "/recordings/testing", true)

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
