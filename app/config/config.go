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

package config

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/configuration"
	"regexp"
	"strconv"
	"strings"
)

type (
	variables struct {
		ServiceName, LoggingLevel, Port               string
		TelemetryEndpoint, TelemetryDataStoreName     string
		VideoUrlBase, CoreCommandUrl                  string
		VideoDevice                                   string
		LiveView, FullscreenView, ShowVideoDebugStats bool
		RecordingDuration                             int
		VideoResolutionWidth, VideoResolutionHeight   int
		VideoOutputFps                                int
		VideoOutputCodec, VideoOutputExtension        string
		VideoCaptureFOURCC                            string
		VideoCaptureBufferSize                        int
		EPCFilter, SKUFilter                          string
		EPCFilterRegex, SKUFilterRegex                *regexp.Regexp
		ImageProcessScale                             int
		SaveCascadeDetectionsToDisk                   bool
		ThumbnailHeight                               int
		EnableCORS                                    bool
		CORSOrigin                                    string
	}
)

// AppConfig exports all config variables
var AppConfig variables

// InitConfig loads application variables
// nolint :gocyclo
func InitConfig() error {
	AppConfig = variables{}

	config, err := configuration.NewConfiguration()
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ServiceName = getOrDefaultString(config, "serviceName", "Loss Prevention Example App")
	AppConfig.LoggingLevel = getOrDefaultString(config, "loggingLevel", "info")

	AppConfig.TelemetryEndpoint = getOrDefaultString(config, "telemetryEndpoint", "")
	AppConfig.TelemetryDataStoreName = getOrDefaultString(config, "telemetryDataStoreName", "")

	AppConfig.Port = getOrDefaultString(config, "port", "8080")
	AppConfig.CoreCommandUrl = getOrDefaultString(config, "coreCommandUrl", "http://edgex-core-command:48082")

	AppConfig.LiveView = getOrDefaultBool(config, "liveView", true)
	AppConfig.FullscreenView = getOrDefaultBool(config, "fullscreenView", false)
	AppConfig.ShowVideoDebugStats = getOrDefaultBool(config, "showVideoDebugStats", false)
	AppConfig.SaveCascadeDetectionsToDisk = getOrDefaultBool(config, "saveCascadeDetectionsToDisk", true)
	AppConfig.RecordingDuration = getOrDefaultInt(config, "recordingDuration", 15)
	AppConfig.VideoResolutionWidth = getOrDefaultInt(config, "videoResolutionWidth", 1280)
	AppConfig.VideoResolutionHeight = getOrDefaultInt(config, "videoResolutionHeight", 720)
	AppConfig.ImageProcessScale = getOrDefaultInt(config, "imageProcessScale", 2)
	AppConfig.VideoOutputFps = getOrDefaultInt(config, "videoOutputFps", 25)
	AppConfig.VideoOutputCodec = getOrDefaultString(config, "videoOutputCodec", "avc1")
	AppConfig.VideoOutputExtension = getOrDefaultString(config, "videoOutputExtension", ".mp4")
	if !strings.HasPrefix(AppConfig.VideoOutputExtension, ".") {
		return fmt.Errorf("videoOutputExtension must start with a period '.'")
	}
	AppConfig.VideoCaptureFOURCC = getOrDefaultString(config, "videoCaptureFOURCC", "MJPG")
	if len(AppConfig.VideoCaptureFOURCC) != 4 && AppConfig.VideoCaptureFOURCC != "" {
		return fmt.Errorf("videoCaptureFOURCC must be a four-letter string such as 'MJPG', or an empty-string to disable setting this property: \"\"")
	}
	AppConfig.VideoCaptureBufferSize = getOrDefaultInt(config, "videoCaptureBufferSize", 3)
	if AppConfig.VideoCaptureBufferSize < 1 {
		return fmt.Errorf("videoCaptureBufferSize must be a value greater than 0")
	}

	if AppConfig.VideoUrlBase, err = config.GetString("videoUrlBase"); err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	if AppConfig.VideoDevice, err = config.GetString("ipCameraStreamUrl"); err != nil {
		if device, err := config.GetInt("usbCameraDeviceIndex"); err == nil && device >= 0 {
			AppConfig.VideoDevice = strconv.Itoa(device)
		} else {
			return errors.Wrapf(err, "Unable to load config variables: %v", err)
		}
	}
	AppConfig.ThumbnailHeight = getOrDefaultInt(config, "thumbnailHeight", 200)
	AppConfig.EnableCORS = getOrDefaultBool(config, "enableCORS", true)
	AppConfig.CORSOrigin = getOrDefaultString(config, "corsOrigin", "*")

	AppConfig.EPCFilter = getOrDefaultString(config, "epcFilter", "*")
	if AppConfig.EPCFilterRegex, err = regexp.Compile(filterToRegexPattern(AppConfig.EPCFilter)); err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %v", err)
	}

	AppConfig.SKUFilter = getOrDefaultString(config, "skuFilter", "*")
	if AppConfig.SKUFilterRegex, err = regexp.Compile(filterToRegexPattern(AppConfig.SKUFilter)); err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %v", err)
	}

	return nil
}

func filterToRegexPattern(filter string) string {
	return "^" + strings.ReplaceAll(filter, "*", ".*") + "$"
}

func getOrDefaultInt(config *configuration.Configuration, path string, defaultValue int) int {
	value, err := config.GetInt(path)
	if err != nil {
		logrus.Debugf("%s was missing from configuration, setting to default value of %s", path, defaultValue)
		return defaultValue
	}
	return value
}

func getOrDefaultBool(config *configuration.Configuration, path string, defaultValue bool) bool {
	value, err := config.GetBool(path)
	if err != nil {
		logrus.Debugf("%s was missing from configuration, setting to default value of %s", path, defaultValue)
		return defaultValue
	}
	return value
}

func getOrDefaultString(config *configuration.Configuration, path string, defaultValue string) string {
	value, err := config.GetString(path)
	if err != nil {
		logrus.Debugf("%s was missing from configuration, setting to default value of %s", path, defaultValue)
		return defaultValue
	}
	return value
}
