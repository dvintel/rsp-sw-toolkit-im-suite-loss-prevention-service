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
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"
)

const (
	cascadeFolder = "./res/data/haarcascades"

	font          = gocv.FontHersheySimplex
	fontScale     = 0.75
	fontThickness = 2
	textPadding   = 5

	fileMode = 0777
)

var (
	cameraSemaphone = semaphore.NewWeighted(1)

	red    = color.RGBA{255, 0, 0, 0}
	green  = color.RGBA{0, 255, 0, 0}
	blue   = color.RGBA{0, 0, 255, 0}
	orange = color.RGBA{255, 255, 0, 0}
	white  = color.RGBA{255, 255, 255, 0}
	purple = color.RGBA{255, 0, 255, 0}

	debugStatsColor = white

	cascadeFiles = []CascadeFile{
		{
			name:     "face",
			filename: "haarcascade_frontalface_default.xml",
			drawOptions: DrawOptions{
				annotation: "Bacon Thief!",
				color:      red,
				thickness:  2,
			},
			detectParams: DetectParams{
				scale:        1.4,
				minNeighbors: 4,
				flags:        0,
				minScaleX:    0.05,
				minScaleY:    0.05,
				maxScaleX:    0.8,
				maxScaleY:    0.8,
			},
		},
		{
			name:     "profile_face",
			filename: "haarcascade_profileface.xml",
			drawOptions: DrawOptions{
				annotation: "Employee",
				color:      blue,
				thickness:  2,
			},
			detectParams: DetectParams{
				scale:        1.4,
				minNeighbors: 4,
				flags:        0,
				minScaleX:    0.1,
				minScaleY:    0.1,
				maxScaleX:    0.8,
				maxScaleY:    0.8,
			},
		},
		//{
		//	name:     "eye",
		//	filename: "haarcascade_eye.xml",
		//	drawOptions: DrawOptions{
		//		color:          blue,
		//		thickness:      1,
		//		renderAsCircle: true,
		//	},
		//	detectParams: DetectParams{
		//		scale:        1.5,
		//		minNeighbors: 5,
		//		flags:        0,
		//		minScaleX:    0.01,
		//		minScaleY:    0.01,
		//		maxScaleX:    0.1,
		//		maxScaleY:    0.1,
		//	},
		//},
		{
			name:     "upper_body",
			filename: "haarcascade_upperbody.xml",
			drawOptions: DrawOptions{
				color:      white,
				thickness:  2,
				annotation: "Employee",
			},
			detectParams: DetectParams{
				scale:        1.5,
				minNeighbors: 3,
				flags:        0,
				minScaleX:    0.1,
				minScaleY:    0.1,
				maxScaleX:    0.75,
				maxScaleY:    0.75,
			},
		},
		{
			name:     "full_body",
			filename: "haarcascade_fullbody.xml",
			drawOptions: DrawOptions{
				color:      orange,
				thickness:  2,
				annotation: "Suspicious Person",
			},
			detectParams: DetectParams{
				scale:        1.4,
				minNeighbors: 2,
				flags:        0,
				minScaleX:    0.1,
				minScaleY:    0.1,
				maxScaleX:    0.6,
				maxScaleY:    0.8,
			},
		},
	}
)

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
	for _, cascadeFile := range cascadeFiles {
		classifier := gocv.NewCascadeClassifier()
		if !classifier.Load(cascadeFolder + "/" + cascadeFile.filename) {
			logrus.Errorf("error reading cascade file: %v", cascadeFile.filename)
			continue
		}

		cascadeFile.classifier = &classifier
		recorder.cascadeQueues = append(recorder.cascadeQueues, cascadeFile.CreateQueue(100))
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

	// skip the first few frames (sometimes it takes longer to read, which affects the smoothness of the video)
	recorder.webcam.Grab(config.AppConfig.VideoCaptureBufferSize)

	logrus.Debugf("input codec: %s", recorder.webcam.CodecString())

	if err = os.MkdirAll(recorder.outputFolder, fileMode); err != nil {
		return err
	}

	logrus.Debug("Open() completed")
	return nil
}

func (recorder *Recorder) Begin() time.Time {
	for _, cascadeQueue := range recorder.cascadeQueues {
		go recorder.ProcessClassifierQueue(recorder.done, cascadeQueue)
	}

	//go recorder.ProcessOpenVINOQueue(recorder.done)

	if recorder.outputFolder != "" {
		go recorder.ProcessWriteQueue(recorder.done)
	}

	go recorder.ProcessWaitQueue(recorder.done)
	go recorder.ProcessCloseQueue(recorder.done)

	if recorder.liveView {
		//recorder.window.ResizeWindow(1920, 1080)
		recorder.window.ResizeWindow(recorder.width, recorder.height)

		if config.AppConfig.FullscreenView {
			recorder.window.SetWindowProperty(gocv.WindowPropertyFullscreen, gocv.WindowFullscreen)
		}
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
	var read, process DebugStats
	// pre-compute the x location of the avg stats
	x2 := gocv.GetTextSize("Avg Process: 99.9", font, fontScale, fontThickness).X
	x3 := gocv.GetTextSize("Min Process: 99", font, fontScale, fontThickness).X + x2 + 60
	yPadding := 35
	yStart := 0
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
					read.AddValue(float64(frameToken.readTS - frameToken.startTS))
					process.AddValue(float64(frameToken.processedTS - frameToken.readTS))

					// Instant
					gocv.PutText(&frameToken.frame, "   Read: "+strconv.FormatInt(int64(read.current), 10),
						image.Point{textPadding, yStart + (yPadding * 1)}, font, fontScale, debugStatsColor, fontThickness)
					gocv.PutText(&frameToken.frame, "Process: "+strconv.FormatInt(int64(process.current), 10),
						image.Point{textPadding, yStart + (yPadding * 2)}, font, fontScale, debugStatsColor, fontThickness)
					gocv.PutText(&frameToken.frame, "ReadFPS: "+strconv.FormatFloat(read.FPS(), 'f', 1, 64),
						image.Point{textPadding, yStart + (yPadding * 3)}, font, fontScale, debugStatsColor, fontThickness)
					gocv.PutText(&frameToken.frame, "ProcFPS: "+strconv.FormatFloat(process.FPS(), 'f', 1, 64),
						image.Point{textPadding, yStart + (yPadding * 4)}, font, fontScale, debugStatsColor, fontThickness)

					// Min / Max
					gocv.PutText(&frameToken.frame, "   Min Read: "+strconv.FormatInt(int64(read.min), 10),
						image.Point{x2, yStart + (yPadding * 1)}, font, fontScale, debugStatsColor, fontThickness)
					gocv.PutText(&frameToken.frame, "   Max Read: "+strconv.FormatInt(int64(read.max), 10),
						image.Point{x2, yStart + (yPadding * 2)}, font, fontScale, debugStatsColor, fontThickness)
					gocv.PutText(&frameToken.frame, "Min Process: "+strconv.FormatInt(int64(process.min), 10),
						image.Point{x2, yStart + (yPadding * 3)}, font, fontScale, debugStatsColor, fontThickness)
					gocv.PutText(&frameToken.frame, "Max Process: "+strconv.FormatInt(int64(process.max), 10),
						image.Point{x2, yStart + (yPadding * 4)}, font, fontScale, debugStatsColor, fontThickness)

					// Average
					gocv.PutText(&frameToken.frame, "   Avg Read: "+strconv.FormatFloat(read.Average(), 'f', 1, 64),
						image.Point{x3, yStart + (yPadding * 1)}, font, fontScale, debugStatsColor, fontThickness)
					gocv.PutText(&frameToken.frame, "Avg Process: "+strconv.FormatFloat(process.Average(), 'f', 1, 64),
						image.Point{x3, yStart + (yPadding * 2)}, font, fontScale, debugStatsColor, fontThickness)
					gocv.PutText(&frameToken.frame, "Avg ReadFPS: "+strconv.FormatFloat(read.AverageFPS(), 'f', 1, 64),
						image.Point{x3, yStart + (yPadding * 3)}, font, fontScale, debugStatsColor, fontThickness)
					gocv.PutText(&frameToken.frame, "Avg ProcFPS: "+strconv.FormatFloat(process.AverageFPS(), 'f', 1, 64),
						image.Point{x3, yStart + (yPadding * 4)}, font, fontScale, debugStatsColor, fontThickness)

				}

				frameToken.overlayMutex.Lock()
				for _, overlay := range frameToken.overlays {
					if overlay.drawOptions.renderAsCircle {
						radius := (overlay.rect.Max.X - overlay.rect.Min.X) / 2
						gocv.Circle(&frameToken.frame, image.Point{overlay.rect.Max.X - radius, overlay.rect.Max.Y - radius}, radius, overlay.drawOptions.color, overlay.drawOptions.thickness)
					} else {
						gocv.Rectangle(&frameToken.frame, overlay.rect, overlay.drawOptions.color, overlay.drawOptions.thickness)
					}
					gocv.PutText(&frameToken.frame, overlay.drawOptions.annotation, image.Point{overlay.rect.Min.X, overlay.rect.Min.Y - 10}, font, 1, overlay.drawOptions.color, fontThickness)
				}
				frameToken.overlayMutex.Unlock()

				recorder.window.IMShow(frameToken.frame)
				key := recorder.window.WaitKey(1)

				// ESC, Q, q
				if key == 27 || key == 'q' || key == 'Q' {
					logrus.Debugf("stopping video live view")
					recorder.liveView = false
					safeClose(recorder.window)
					break
				}
			}

			recorder.waitGroup.Add(1)
			recorder.closeBuffer <- frameToken

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

			switch token.index {
			case 0:
				token.writeFrame(filepath.Join(recorder.outputFolder, "frame.first.jpg"))
				token.writeThumb(filepath.Join(recorder.outputFolder, "thumb.jpg"))
			case recorder.frameCount / 2:
				token.writeFrame(filepath.Join(recorder.outputFolder, "frame.middle.jpg"))
			case recorder.frameCount - 1:
				token.writeFrame(filepath.Join(recorder.outputFolder, "frame.last.jpg"))
			default:
				break
			}

			token.waitGroup.Done()
		}
	}
}

func (token *FrameToken) writeThumb(filename string) {
	logrus.Debugf("writing thumbnail image: %s", filename)
	thumb := gocv.NewMat()
	// compute the width based on the aspect ratio
	width := int(float64(config.AppConfig.ThumbnailHeight) * (float64(config.AppConfig.VideoResolutionWidth) / float64(config.AppConfig.VideoResolutionHeight)))
	gocv.Resize(token.frame, &thumb, image.Point{width, config.AppConfig.ThumbnailHeight}, 0, 0, gocv.InterpolationLinear)
	go func() {
		gocv.IMWrite(filename, thumb)
		safeClose(&thumb)
	}()
}

func (token *FrameToken) writeFrame(filename string) {
	logrus.Debugf("writing image: %s", filename)
	gocv.IMWrite(filename, token.frame)
}

func (token *FrameToken) writeFrameRegion(filename string, region image.Rectangle) {
	logrus.Debugf("writing image region: %s (%+v)", filename, region)
	gocv.IMWrite(filename, token.frame.Region(region))
}

func (recorder *Recorder) ProcessClassifierQueue(done chan bool, cascadeQueue *CascadeQueue) {
	cascade := cascadeQueue.cascadeFile
	params := cascade.detectParams

	logrus.Debugf("ProcessClassifierQueue(classifier: %s) goroutine started", cascade.name)
	for {
		select {
		case <-done:
			logrus.Debugf("ProcessClassifierQueue(classifier: %s) goroutine completed", cascade.name)
			close(cascadeQueue.buffer)
			return

		case token := <-cascadeQueue.buffer:
			var rects []image.Rectangle
			if reflect.DeepEqual(params, DetectParams{}) {
				rects = cascade.classifier.DetectMultiScale(token.procFrame)
			} else {
				rects = cascade.classifier.DetectMultiScaleWithParams(token.procFrame, params.scale, params.minNeighbors, params.flags,
					image.Point{X: int(float64(recorder.width) * params.minScaleX), Y: int(float64(recorder.height) * params.minScaleY)},
					image.Point{X: int(float64(recorder.width) * params.maxScaleX), Y: int(float64(recorder.height) * params.maxScaleY)})
			}

			if len(rects) > 0 {
				logrus.Debugf("Detected %v %s(s)", len(rects), cascade.name)

				if config.AppConfig.SaveCascadeDetectionsToDisk {
					if cascadeQueue.found < len(rects) {
						// todo: should this not overwrite existing rects and just add, so if 1 is found
						// 		 and then 2 is found later on, we have a total of 3, not just the most recent 2?
						cascadeQueue.found = len(rects)
						for i, rect := range rects {
							token.writeFrameRegion(filepath.Join(recorder.outputFolder, fmt.Sprintf("%s.%d.jpg", cascade.name, i)), transformProcessRect(rect))
						}
					}
				}

				token.overlayMutex.Lock()
				for _, rect := range rects {
					token.overlays = append(token.overlays, FrameOverlay{rect: transformProcessRect(rect), drawOptions: cascade.drawOptions})
				}
				token.overlayMutex.Unlock()
			}
			token.waitGroup.Done()
		}
	}
}

// transformProcessRect takes a smaller scaled rectangle produced by a processing function and transforms it
// into a rectangle relative to the full original image size
func transformProcessRect(rect image.Rectangle) image.Rectangle {
	return image.Rectangle{
		Min: image.Point{
			X: rect.Min.X * config.AppConfig.ImageProcessScale,
			Y: rect.Min.Y * config.AppConfig.ImageProcessScale,
		},
		Max: image.Point{
			X: rect.Max.X * config.AppConfig.ImageProcessScale,
			Y: rect.Max.Y * config.AppConfig.ImageProcessScale,
		},
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

func (recorder *Recorder) ProcessCloseQueue(done chan bool) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	logrus.Debug("ProcessCloseQueue() goroutine started")
	for {
		select {
		case <-done:
			logrus.Debug("ProcessCloseQueue() goroutine completed")
			close(recorder.closeBuffer)

		case token := <-recorder.closeBuffer:
			go safeClose(&token.frame)
			go safeClose(&token.procFrame)
			recorder.waitGroup.Done()
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

	safeClose(recorder.webcam)
	safeClose(recorder.writer)
	for _, cascadeQueue := range recorder.cascadeQueues {
		safeClose(cascadeQueue.cascadeFile.classifier)
	}
	//safeClose(&recorder.net)
	if recorder.liveView {
		safeClose(recorder.window)
	}

	close(recorder.done)

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

	recorder := NewRecorder(config.AppConfig.VideoDevice, "/tmp")
	recorder.liveView = false
	if err := recorder.Open(); err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}
	recorder.Begin()
	// set the writer to nil to prevent calling close on it, as we do not open it
	recorder.writer = nil
	recorder.Close()

	return nil
}

func RecordVideoToDisk(videoDevice string, seconds int, outputFolder string) error {
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

	recorder := NewRecorder(videoDevice, outputFolder)
	if err := recorder.Open(); err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}

	defer recorder.Close()

	begin := recorder.Begin()
	recorder.frameCount = int(recorder.fps * float64(seconds))

	for i := 0; i < recorder.frameCount; i++ {

		token := recorder.NewFrameToken(i)
		if ok := recorder.webcam.Read(&token.frame); !ok {
			return fmt.Errorf("unable to read from webcam. device closed: %+v", recorder.videoDevice)
		}
		token.readTS = helper.UnixMilliNow()

		if token.frame.Empty() {
			logrus.Debug("skipping empty frame from webcam")
			continue
		}

		// Add 1 for for writeBuffer and 1 PER classifierQueue
		token.waitGroup.Add(1)
		recorder.writeBuffer <- token

		// Resize smaller for use with the cascade classifiers
		gocv.Resize(token.frame, &token.procFrame, image.Point{}, 1.0/float64(config.AppConfig.ImageProcessScale), 1.0/float64(config.AppConfig.ImageProcessScale), gocv.InterpolationLinear)

		for _, cascadeQueue := range recorder.cascadeQueues {
			token.waitGroup.Add(1)
			cascadeQueue.buffer <- token
		}

		// Add 1 for waitBuffer
		recorder.waitGroup.Add(1)
		recorder.waitBuffer <- token
	}

	end := recorder.Wait()
	logrus.Debugf("recording took %v", end.Sub(begin))

	logrus.Infof("Video url: %s%s", config.AppConfig.VideoUrlBase, url.QueryEscape(recorder.outputFilename))

	return nil
}
