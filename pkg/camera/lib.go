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

package camera

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"gocv.io/x/gocv"
	"golang.org/x/sync/semaphore"
	"image"
	"image/color"
	"io"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	xmlFile = "./res/docker/haarcascade_frontalface_default.xml"
)

var (
	cameraSemaphone = semaphore.NewWeighted(1)
)

type FrameToken struct {
	frame     gocv.Mat
	waitGroup sync.WaitGroup
}

func (recorder *Recorder) NewFrameToken() *FrameToken {
	return &FrameToken{
		frame: gocv.NewMat(),
	}
}

type Recorder struct {
	videoDevice    string
	foundFace      bool
	outputFilename string
	fps            float64
	codec          string
	width          int
	height         int
	liveView       bool
	waitGroup      sync.WaitGroup
	done           chan bool

	webcam     *gocv.VideoCapture
	classifier *gocv.CascadeClassifier
	writer     *gocv.VideoWriter
	window     *gocv.Window

	faceBuffer  chan *FrameToken
	writeBuffer chan *FrameToken
	waitBuffer  chan *FrameToken
}

func NewRecorder(videoDevice string, outputFilename string) *Recorder {
	recorder := &Recorder{
		videoDevice:    videoDevice,
		outputFilename: outputFilename,
		width:          config.AppConfig.VideoResolutionWidth,
		height:         config.AppConfig.VideoResolutionHeight,
		liveView:       config.AppConfig.LiveView,
		fps:            float64(config.AppConfig.VideoOutputFps),
		codec:          config.AppConfig.VideoOutputCodec,
		window:         gocv.NewWindow(config.AppConfig.ServiceName + " - OpenVINO"),

		faceBuffer:  make(chan *FrameToken, 100),
		writeBuffer: make(chan *FrameToken, 100),
		waitBuffer:  make(chan *FrameToken, 100),
		done:        make(chan bool),
	}

	return recorder
}

// codecToFloat64 returns a float64 representation of FourCC bytes for use with `gocv.VideoCaptureFOURCC`
func codecToFloat64(codec string) float64 {
	if len(codec) != 4 {
		return -1.0
	}
	c1 := []rune(string(codec[0]))[0]
	c2 := []rune(string(codec[1]))[0]
	c3 := []rune(string(codec[2]))[0]
	c4 := []rune(string(codec[3]))[0]
	return float64((c1 & 255) + ((c2 & 255) << 8) + ((c3 & 255) << 16) + ((c4 & 255) << 24))
}

func (recorder *Recorder) Open() error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	logrus.Debug("Open()")
	var err error

	if recorder.webcam, err = gocv.OpenVideoCapture(recorder.videoDevice); err != nil {
		return errors.Wrapf(err, "Error opening video capture device: %+v", recorder.videoDevice)
	}
	// Note: setting the video capture four cc is very important for performance reasons.
	// 		 it should also be set before applying any size or fps configurations.
	recorder.webcam.Set(gocv.VideoCaptureFOURCC, codecToFloat64(config.AppConfig.VideoCaptureFOURCC))
	recorder.webcam.Set(gocv.VideoCaptureFrameWidth, float64(recorder.width))
	recorder.webcam.Set(gocv.VideoCaptureFrameHeight, float64(recorder.height))
	recorder.webcam.Set(gocv.VideoCaptureFPS, recorder.fps)
	recorder.webcam.Set(gocv.VideoCaptureBufferSize, float64(config.AppConfig.VideoCaptureBufferSize))

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	recorder.classifier = &classifier
	if !recorder.classifier.Load(xmlFile) {
		return fmt.Errorf("error reading cascade file: %v", xmlFile)
	}

	// skip the first frame (sometimes it takes longer to read, which affects the smoothness of the video)
	recorder.webcam.Grab(1)

	logrus.Debug("Open() completed")
	return nil
}

func (recorder *Recorder) Begin() time.Time {
	go recorder.ProcessFaceQueue(recorder.done)
	if recorder.outputFilename != "" {
		go recorder.ProcessWriteQueue(recorder.done)
	}
	go recorder.ProcessWaitQueue(recorder.done)
	if recorder.liveView {
		recorder.window.ResizeWindow(recorder.width, recorder.height)
	}
	return time.Now()
}

func (recorder *Recorder) Wait() time.Time {
	logrus.Debug("Wait()")
	recorder.waitGroup.Wait()
	logrus.Debug("Wait() completed")
	return time.Now()
}

func (recorder *Recorder) ProcessWaitQueue(done chan bool) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	logrus.Debug("ProcessWaitQueue() goroutine started")
	for {
		select {
		case <-done:
			logrus.Debug("ProcessWaitQueue() goroutine completed")
			close(recorder.waitBuffer)
			return

		case frameToken := <-recorder.waitBuffer:
			frameToken.waitGroup.Wait()

			if recorder.liveView {
				recorder.window.IMShow(frameToken.frame)
				recorder.window.WaitKey(1)
				//
				//if recorder.window.GetWindowProperty(gocv.WindowPropertyVisible) < 1 {
				//	logrus.Debugf("stopping video live view")
				//	recorder.liveView = false
				//	safeClose(recorder.window)
				//}
			}

			safeClose(&frameToken.frame)
			recorder.waitGroup.Done()
		}
	}
}

func (recorder *Recorder) ProcessWriteQueue(done chan bool) error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	var err error
	recorder.writer, err = gocv.VideoWriterFile(recorder.outputFilename, recorder.codec, recorder.fps, recorder.width, recorder.height, true)
	if err != nil {
		return errors.Wrapf(err, "error opening video writer device: %+v", recorder.outputFilename)
	}

	logrus.Debug("ProcessWriteQueue() goroutine started")
	for {
		select {
		case <-done:
			logrus.Debug("ProcessWriteQueue() goroutine completed")
			close(recorder.writeBuffer)
			return nil

		case token := <-recorder.writeBuffer:
			if err := recorder.writer.Write(token.frame); err != nil {
				logrus.Errorf("error occurred while writing video to disk: %v", err)
			}
			token.waitGroup.Done()
		}
	}
}

func (recorder *Recorder) ProcessFaceQueue(done chan bool) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	red := color.RGBA{255, 0, 0, 0}
	logrus.Debug("ProcessFaceQueue() goroutine started")
	for {
		select {
		case <-done:
			logrus.Debug("ProcessFaceQueue() goroutine completed")
			close(recorder.faceBuffer)
			return

		case token := <-recorder.faceBuffer:
			//if !recorder.foundFace {
			rects := recorder.classifier.DetectMultiScaleWithParams(token.frame, 1.1, 6, 0,
				image.Point{X: int(float64(recorder.width) * 0.1), Y: int(float64(recorder.height) * 0.1)}, image.Point{X: int(float64(recorder.width) * 0.8), Y: int(float64(recorder.height) * 0.8)})

			if len(rects) > 0 {
				logrus.Debugf("Detected %v face(s)", len(rects))
				prefix := strings.TrimSuffix(recorder.outputFilename, config.AppConfig.VideoOutputExtension)
				if !recorder.foundFace {
					gocv.IMWrite(fmt.Sprintf("%s.face.jpg", prefix), token.frame)
					for i, rect := range rects {
						gocv.IMWrite(fmt.Sprintf("%s.face.%d.jpg", prefix, i), token.frame.Region(rect))
					}
				}
				for _, rect := range rects {
					gocv.Rectangle(&token.frame, rect, red, 2)
					gocv.PutText(&token.frame, "Bacon Thief!", image.Point{rect.Min.X, rect.Min.Y - 10}, gocv.FontHersheyComplex, 1, red, 3)
				}

				recorder.foundFace = true
			}
			//}
			token.waitGroup.Done()
		}
	}
}

func (recorder *Recorder) Close() {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	logrus.Debug("Close()")

	// this will signal to stop the background tasks
	recorder.done <- true
	close(recorder.done)

	safeClose(recorder.webcam)
	safeClose(recorder.writer)
	safeClose(recorder.classifier)
	safeClose(recorder.window)

	logrus.Debug("Close() completed")
}

func safeClose(c io.Closer) {
	if c == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.Debugf("closing %v", reflect.TypeOf(c))
	}

	if err := c.Close(); err != nil {
		logrus.Errorf("error while attempting to close %v: %v", reflect.TypeOf(c), err)
	}
}

func SanityCheck() error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	if !cameraSemaphone.TryAcquire(1) {
		return fmt.Errorf("unable to acquire camera lock")
	}
	defer cameraSemaphone.Release(1)

	recorder := NewRecorder(config.AppConfig.VideoDevice, "")
	recorder.liveView = false
	if err := recorder.Open(); err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}
	recorder.Begin()
	recorder.Close()

	return nil
}

func RecordVideoToDisk(videoDevice string, seconds int, outputFilename string) error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	// only allow one recording at a time
	// also we do not want to queue up recordings because they would be at invalid times anyways
	if !cameraSemaphone.TryAcquire(1) {
		logrus.Warn("unable to acquire camera lock, we must already be recording. skipping.")
		return nil
	}
	defer cameraSemaphone.Release(1)

	recorder := NewRecorder(videoDevice, outputFilename)
	if err := recorder.Open(); err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}

	defer recorder.Close()

	begin := recorder.Begin()
	frameCount := int(recorder.fps * float64(seconds))
	for i := 0; i < frameCount; i++ {

		token := recorder.NewFrameToken()

		if ok := recorder.webcam.Read(&token.frame); !ok {
			return fmt.Errorf("unable to read from webcam. device closed: %+v", recorder.videoDevice)
		}

		if token.frame.Empty() {
			logrus.Debug("skipping empty frame from webcam")
			continue
		}

		// Add 2 (one for faceBuffer, another for writeBuffer)
		token.waitGroup.Add(2)
		recorder.faceBuffer <- token
		recorder.writeBuffer <- token

		// Add 1 for waitBuffer
		recorder.waitGroup.Add(1)
		recorder.waitBuffer <- token
	}

	end := recorder.Wait()
	logrus.Debugf("recording took %v", end.Sub(begin))

	logrus.Infof("Video url: %s%s", config.AppConfig.VideoUrlBase, url.QueryEscape(outputFilename))

	return nil
}
