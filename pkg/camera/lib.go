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
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"gocv.io/x/gocv"
	"golang.org/x/sync/semaphore"
	"image"
	"image/color"
	"io"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	xmlFile = "./res/docker/haarcascade_frontalface_default.xml"

	font          = gocv.FontHersheySimplex
	fontScale     = 0.75
	fontThickness = 2
	textPadding   = 5
)

var (
	cameraSemaphone = semaphore.NewWeighted(1)

	red   = color.RGBA{255, 0, 0, 0}
	green = color.RGBA{0, 255, 0, 0}
	blue  = color.RGBA{0, 0, 255, 0}
)

type FrameToken struct {
	startTS      int64
	readTS       int64
	processedTS  int64
	frame        gocv.Mat
	waitGroup    sync.WaitGroup
	overlays     []FrameOverlay
	overlayMutex sync.Mutex
}

type FrameOverlay struct {
	rect image.Rectangle
	text string
}

func (recorder *Recorder) NewFrameToken() *FrameToken {
	return &FrameToken{
		startTS: helper.UnixMilliNow(),
		frame:   gocv.NewMat(),
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
	//net	       gocv.Net
	writer *gocv.VideoWriter
	window *gocv.Window

	faceBuffer chan *FrameToken
	//vinoBuffer  chan *FrameToken
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

		faceBuffer: make(chan *FrameToken, 10),
		//vinoBuffer:  make(chan *FrameToken, 10),
		writeBuffer: make(chan *FrameToken, 10),
		waitBuffer:  make(chan *FrameToken, 10),
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
	if config.AppConfig.VideoCaptureFOURCC != "" {
		recorder.webcam.Set(gocv.VideoCaptureFOURCC, codecToFloat64(config.AppConfig.VideoCaptureFOURCC))
	}
	if recorder.width != 0 {
		recorder.webcam.Set(gocv.VideoCaptureFrameWidth, float64(recorder.width))
	}
	if recorder.height != 0 {
		recorder.webcam.Set(gocv.VideoCaptureFrameHeight, float64(recorder.height))
	}
	if recorder.fps != 0 {
		recorder.webcam.Set(gocv.VideoCaptureFPS, recorder.fps)
	}
	if config.AppConfig.VideoCaptureBufferSize != 0 {
		recorder.webcam.Set(gocv.VideoCaptureBufferSize, float64(config.AppConfig.VideoCaptureBufferSize))
	}

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	recorder.classifier = &classifier
	if !recorder.classifier.Load(xmlFile) {
		return fmt.Errorf("error reading cascade file: %v", xmlFile)
	}

	//caffeModel := "/opt/intel/openvino/models/intel/face-detection-retail-0004/INT8/face-detection-retail-0004.bin"
	//protoModel := "/opt/intel/openvino/models/intel/face-detection-retail-0004/INT8/face-detection-retail-0004.xml"
	//recorder.net = gocv.ReadNet(caffeModel, protoModel)
	//if recorder.net.Empty() {
	//	return fmt.Errorf("error reading network model %v, %v", caffeModel, protoModel)
	//}
	//
	//recorder.net.SetPreferableBackend(gocv.NetBackendType(gocv.NetBackendOpenVINO))
	//recorder.net.SetPreferableTarget(gocv.NetTargetType(gocv.NetTargetCPU))

	// skip the first frame (sometimes it takes longer to read, which affects the smoothness of the video)
	recorder.webcam.Grab(1)

	logrus.Debugf("input codec: %s", recorder.webcam.CodecString())

	logrus.Debug("Open() completed")
	return nil
}

func (recorder *Recorder) Begin() time.Time {
	go recorder.ProcessFaceQueue(recorder.done)
	//go recorder.ProcessOpenVINOQueue(recorder.done)
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

	// for debug stats
	var frameCount, processTotal, readTotal float64
	// pre-compute the x location of the avg stats
	x2 := gocv.GetTextSize("Avg Process: 99.9", font, fontScale, fontThickness).X + textPadding

	logrus.Debug("ProcessWaitQueue() goroutine started")
	for {
		select {
		case <-done:
			logrus.Debug("ProcessWaitQueue() goroutine completed")
			close(recorder.waitBuffer)
			return

		case frameToken := <-recorder.waitBuffer:
			frameToken.waitGroup.Wait()
			frameToken.processedTS = helper.UnixMilliNow()

			if recorder.liveView {
				if config.AppConfig.ShowVideoDebugStats {
					frameCount++
					processTotal += float64(frameToken.processedTS - frameToken.readTS)
					readTotal += float64(frameToken.readTS - frameToken.startTS)

					// Instant
					gocv.PutText(&frameToken.frame, "   Read: "+strconv.FormatInt(frameToken.readTS-frameToken.startTS, 10), image.Point{textPadding, 25}, font, fontScale, green, fontThickness)
					gocv.PutText(&frameToken.frame, "Process: "+strconv.FormatInt(frameToken.processedTS-frameToken.readTS, 10), image.Point{textPadding, 60}, font, fontScale, green, fontThickness)
					gocv.PutText(&frameToken.frame, "    FPS: "+strconv.FormatFloat(1.0/(float64(frameToken.readTS-frameToken.startTS)/1000.0), 'f', 1, 64), image.Point{textPadding, 95}, font, fontScale, green, fontThickness)

					// Average
					gocv.PutText(&frameToken.frame, "   Avg Read: "+strconv.FormatFloat(readTotal/frameCount, 'f', 1, 64), image.Point{x2, 25}, font, fontScale, green, fontThickness)
					gocv.PutText(&frameToken.frame, "Avg Process: "+strconv.FormatFloat(processTotal/frameCount, 'f', 1, 64), image.Point{x2, 60}, font, fontScale, green, fontThickness)
					gocv.PutText(&frameToken.frame, "    Avg FPS: "+strconv.FormatFloat(1.0/((readTotal/frameCount)/1000.0), 'f', 1, 64), image.Point{x2, 95}, font, fontScale, green, fontThickness)
				}

				frameToken.overlayMutex.Lock()
				for _, overlay := range frameToken.overlays {
					gocv.Rectangle(&frameToken.frame, overlay.rect, red, fontThickness)
					gocv.PutText(&frameToken.frame, overlay.text, image.Point{overlay.rect.Min.X, overlay.rect.Min.Y - 10}, font, 1, red, fontThickness)
				}
				frameToken.overlayMutex.Unlock()

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

				token.overlayMutex.Lock()
				for _, rect := range rects {
					token.overlays = append(token.overlays, FrameOverlay{rect, "Bacon Thief!"})
				}
				token.overlayMutex.Unlock()

				recorder.foundFace = true
			}
			//}
			token.waitGroup.Done()
		}
	}
}

//func (recorder *Recorder) ProcessOpenVINOQueue(done chan bool) {
//	defer func() {
//		if r := recover(); r != nil {
//			logrus.Errorf("recovered from panic: %+v", r)
//		}
//	}()
//
//	logrus.Debug("ProcessOpenVINOQueue() goroutine started")
//	for {
//		select {
//		case <-done:
//			logrus.Debug("ProcessOpenVINOQueue() goroutine completed")
//			close(recorder.vinoBuffer)
//			return
//
//		case token := <-recorder.vinoBuffer:
//			//blob := gocv.BlobFromImage(token.frame, 1.0, image.Point{recorder.width, recorder.height}, gocv.Scalar{}, true, false)
//			//recorder.net.SetInput(blob, "vino")
//			//prob := recorder.net.Forward("vino")
//			////minVal, maxVal, minLoc, maxLoc := gocv.MinMaxLoc(prob)
//
//			token.waitGroup.Done()
//		}
//	}
//}

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
	//safeClose(&recorder.net)
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
		token.readTS = helper.UnixMilliNow()

		if token.frame.Empty() {
			logrus.Debug("skipping empty frame from webcam")
			continue
		}

		// Add 2 (one for faceBuffer, another for writeBuffer)
		token.waitGroup.Add(fontThickness)
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
