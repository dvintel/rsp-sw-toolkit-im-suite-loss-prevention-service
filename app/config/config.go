/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
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
		ServiceName, LoggingLevel, Port                             string
		TelemetryEndpoint, TelemetryDataStoreName                   string
		VideoUrlBase, CoreCommandUrl                                string
		VideoDevice                                                 string
		LiveView, FullscreenView, ShowVideoDebugStats               bool
		RecordingDuration                                           int
		VideoResolutionWidth, VideoResolutionHeight                 int
		VideoOutputFps                                              int
		VideoOutputCodec, VideoOutputExtension                      string
		VideoCaptureFOURCC                                          string
		VideoCaptureBufferSize                                      int
		EPCFilter, SKUFilter                                        string
		EPCFilterRegex, SKUFilterRegex                              *regexp.Regexp
		ImageProcessScale                                           int
		SaveObjectDetectionsToDisk                                  bool
		ThumbnailHeight                                             int
		EnableCORS                                                  bool
		CORSOrigin                                                  string
		EnableFaceDetection                                         bool
		FaceDetectionColor                                          float64
		FaceDetectionXmlFile, FaceDetectionAnnotation               string
		EnableProfileFaceDetection                                  bool
		ProfileFaceDetectionColor                                   float64
		ProfileFaceDetectionXmlFile, ProfileFaceDetectionAnnotation string
		EnableUpperBodyDetection                                    bool
		UpperBodyDetectionColor                                     float64
		UpperBodyDetectionXmlFile, UpperBodyDetectionAnnotation     string
		EnableFullBodyDetection                                     bool
		FullBodyDetectionColor                                      float64
		FullBodyDetectionXmlFile, FullBodyDetectionAnnotation       string
		EnableEyeDetection                                          bool
		EyeDetectionColor                                           float64
		EyeDetectionXmlFile, EyeDetectionAnnotation                 string
		NotificationServiceURL, EmailSubscribers                    string
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
	AppConfig.SaveObjectDetectionsToDisk = getOrDefaultBool(config, "saveObjectDetectionsToDisk", true)
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
	AppConfig.VideoCaptureBufferSize = getOrDefaultInt(config, "videoCaptureBufferSize", 1)
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

	AppConfig.EnableFaceDetection = getOrDefaultBool(config, "enableFaceDetection", true)
	AppConfig.FaceDetectionColor = getOrDefaultFloat64(config, "faceDetectionColor", 0)
	AppConfig.FaceDetectionXmlFile = getOrDefaultString(config, "faceDetectionXmlFile", "haarcascade_frontalface_default.xml")
	AppConfig.FaceDetectionAnnotation = getOrDefaultString(config, "faceDetectionAnnotation", "")

	AppConfig.EnableProfileFaceDetection = getOrDefaultBool(config, "enableProfileFaceDetection", true)
	AppConfig.ProfileFaceDetectionColor = getOrDefaultFloat64(config, "profileFaceDetectionColor", 0)
	AppConfig.ProfileFaceDetectionXmlFile = getOrDefaultString(config, "profileFaceDetectionXmlFile", "haarcascade_profileface.xml")
	AppConfig.ProfileFaceDetectionAnnotation = getOrDefaultString(config, "profileFaceDetectionAnnotation", "")

	AppConfig.EnableUpperBodyDetection = getOrDefaultBool(config, "enableUpperBodyDetection", true)
	AppConfig.UpperBodyDetectionColor = getOrDefaultFloat64(config, "upperBodyDetectionColor", 0)
	AppConfig.UpperBodyDetectionXmlFile = getOrDefaultString(config, "upperBodyDetectionXmlFile", "haarcascade_upperbody.xml")
	AppConfig.UpperBodyDetectionAnnotation = getOrDefaultString(config, "upperBodyDetectionAnnotation", "")

	AppConfig.EnableFullBodyDetection = getOrDefaultBool(config, "enableFullBodyDetection", true)
	AppConfig.FullBodyDetectionColor = getOrDefaultFloat64(config, "fullBodyDetectionColor", 0)
	AppConfig.FullBodyDetectionXmlFile = getOrDefaultString(config, "fullBodyDetectionXmlFile", "haarcascade_fullbody.xml")
	AppConfig.FullBodyDetectionAnnotation = getOrDefaultString(config, "fullBodyDetectionAnnotation", "")

	AppConfig.EnableEyeDetection = getOrDefaultBool(config, "enableEyeDetection", false)
	AppConfig.EyeDetectionColor = getOrDefaultFloat64(config, "eyeDetectionColor", 0)
	AppConfig.EyeDetectionXmlFile = getOrDefaultString(config, "eyeDetectionXmlFile", "haarcascade_eye.xml")
	AppConfig.EyeDetectionAnnotation = getOrDefaultString(config, "eyeDetectionAnnotation", "")

	AppConfig.NotificationServiceURL = getOrDefaultString(config, "notificationServiceURL", "http://edgex-support-notifications:48060")
	AppConfig.EmailSubscribers = getOrDefaultString(config, "emailSubscribers", "")

	return nil
}

func filterToRegexPattern(filter string) string {
	return "^" + strings.ReplaceAll(filter, "*", ".*") + "$"
}

func getOrDefaultFloat64(config *configuration.Configuration, path string, defaultValue float64) float64 {
	value, err := config.GetFloat(path)
	if err != nil {
		logrus.Debugf("%s was missing from configuration, setting to default value of %v", path, defaultValue)
		return defaultValue
	}
	return value
}

func getOrDefaultInt(config *configuration.Configuration, path string, defaultValue int) int {
	value, err := config.GetInt(path)
	if err != nil {
		logrus.Debugf("%s was missing from configuration, setting to default value of %v", path, defaultValue)
		return defaultValue
	}
	return value
}

func getOrDefaultBool(config *configuration.Configuration, path string, defaultValue bool) bool {
	value, err := config.GetBool(path)
	if err != nil {
		logrus.Debugf("%s was missing from configuration, setting to default value of %v", path, defaultValue)
		return defaultValue
	}
	return value
}

func getOrDefaultString(config *configuration.Configuration, path string, defaultValue string) string {
	value, err := config.GetString(path)
	if err != nil {
		logrus.Debugf("%s was missing from configuration, setting to default value of %v", path, defaultValue)
		return defaultValue
	}
	return value
}
