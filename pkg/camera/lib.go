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
	"gocv.io/x/gocv"
	"image"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	maxFps         = 25
	videoWidth     = 1920
	videoHeight    = 1080
	codec          = "avc1"
	VideoExtension = ".mp4"
	xmlFile        = "./res/docker/haarcascade_frontalface_default.xml"
)

type FrameToken struct {
	frame     *gocv.Mat
	waitGroup *sync.WaitGroup
}

func NewFrameToken(frame *gocv.Mat, waitGroup *sync.WaitGroup) FrameToken {
	return FrameToken{
		frame:     frame,
		waitGroup: waitGroup,
	}
}

type Recorder struct {
	videoDevice    int
	foundFace      bool
	outputFilename string
	fps            float64
	codec          string
	width          int
	height         int
	waitGroup      sync.WaitGroup
	done           chan bool

	webcam     *gocv.VideoCapture
	classifier *gocv.CascadeClassifier
	writer     *gocv.VideoWriter

	writeBuffer chan FrameToken
	faceBuffer  chan FrameToken
}

func NewRecorder(videoDevice int, outputFilename string) *Recorder {
	recorder := &Recorder{
		videoDevice:    videoDevice,
		outputFilename: outputFilename,
		width:          videoWidth,
		height:         videoHeight,
		fps:            maxFps,
		codec:          codec,
		faceBuffer:     make(chan FrameToken, 25),
		done:           make(chan bool),
	}

	return recorder
}

// ToCodec returns an float64 representation of FourCC bytes
func ToCodec(codec string) float64 {
	if len(codec) != 4 {
		return -1.0
	}
	c1 := []rune(string(codec[0]))[0]
	c2 := []rune(string(codec[1]))[0]
	c3 := []rune(string(codec[2]))[0]
	c4 := []rune(string(codec[3]))[0]
	return float64((c1 & 255) + ((c2 & 255) << 8) + ((c3 & 255) << 16) + ((c4 & 255) << 24))
}

func (r *Recorder) Open() error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	logrus.Debug("Open()")
	var err error

	if r.webcam, err = gocv.OpenVideoCapture(r.videoDevice); err != nil {
		return errors.Wrapf(err, "Error opening video capture device: %+v", r.videoDevice)
	}
	// Note: setting the video capture four cc is very important for performance reasons.
	// 		 it should also be set before applying any size or fps configurations.
	r.webcam.Set(gocv.VideoCaptureFOURCC, ToCodec("MJPG"))
	r.webcam.Set(gocv.VideoCaptureFrameWidth, float64(r.width))
	r.webcam.Set(gocv.VideoCaptureFrameHeight, float64(r.height))
	r.webcam.Set(gocv.VideoCaptureBufferSize, 3)
	r.webcam.Set(gocv.VideoCaptureFPS, r.fps)

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	r.classifier = &classifier
	if !r.classifier.Load(xmlFile) {
		return fmt.Errorf("error reading cascade file: %v", xmlFile)
	}

	// skip the first frame
	r.webcam.Grab(1)

	//s := time.Now()
	//r.webcam.Grab(1)
	//d := time.Now().Sub(s)
	//computedFps := 1000 / int64(d/time.Millisecond)
	//logrus.Debugf("computedFps: %v", computedFps)
	//
	//r.fps = math.Min(float64(computedFps), maxFps)
	//logrus.Debugf("actualFps: %v", r.fps)
	////r.webcam.Set(gocv.VideoCaptureFPS, r.fps)

	r.writer, err = gocv.VideoWriterFile(r.outputFilename, r.codec, r.fps, r.width, r.height, true)
	if err != nil {
		return errors.Wrapf(err, "error opening video writer device: %+v", r.outputFilename)
	}

	logrus.Debug("Open() completed")
	return nil
}

func (r *Recorder) Next() error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	logrus.Debug("Next()")
	frame := gocv.NewMat()
	if ok := r.webcam.Read(&frame); !ok {
		return fmt.Errorf("unable to read from webcam. device closed: %+v", r.videoDevice)
	}

	//r.waitGroup.Add(1)
	//go func(wg *sync.WaitGroup) {
	//	defer func() {
	//		if r := recover(); r != nil {
	//			logrus.Errorf("recovered from panic: %+v", r)
	//		}
	//	}()
	//
	//
	//	//safeClose(&frame)
	//	//wg.Done()
	//}(&r.waitGroup)

	logrus.Debug("processing frame")
	frameWaitGroup := new(sync.WaitGroup)
	frameWaitGroup.Add(2)

	frameClone := frame.Clone()
	r.writeBuffer <- NewFrameToken(&frameClone, frameWaitGroup)
	//r.faceBuffer <- NewFrameToken(frame.Clone(), frameWaitGroup)

	//frameWaitGroup.Wait()

	logrus.Debug("processing frame done")

	logrus.Debug("Next() completed")
	return nil
}

func (r *Recorder) Begin() {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	logrus.Debug("Begin()")
	r.ProcessFaceQueue(r.done)
	logrus.Debug("Begin() completed")
}

func (r *Recorder) Wait() {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	logrus.Debug("Wait()")
	r.waitGroup.Wait()
	logrus.Debug("Wait() completed")
}

func (r *Recorder) ProcessFaceQueue(done chan bool) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	logrus.Debug("ProcessFaceQueue()")
	for {
		select {
		case <-done:
			logrus.Debug("ProcessFaceQueue() completed")
			return

		case token := <-r.faceBuffer:
			logrus.Debug("faceBuffer token")
			if !r.foundFace {
				rects := r.classifier.DetectMultiScale(*token.frame)
				if len(rects) > 0 {
					logrus.Debugf("Detected %v face(s)", len(rects))
					gocv.IMWrite(fmt.Sprintf("%s.face.jpg", r.outputFilename), *token.frame)
					for i, rect := range rects {
						gocv.IMWrite(fmt.Sprintf("%s.face.%d.jpg", r.outputFilename, i), token.frame.Region(rect))
					}
					r.foundFace = true
				}
			}
			logrus.Debug("faceBuffer token complete")
			//token.waitGroup.Done()
		}
	}
}

func (r *Recorder) Close() {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	close(r.done)
	close(r.faceBuffer)
	close(r.writeBuffer)

	safeClose(r.webcam)
	safeClose(r.writer)
	safeClose(r.classifier)
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

	logrus.Debugf("closing %v", reflect.TypeOf(c))

	if err := c.Close(); err != nil {
		logrus.Errorf("error while attempting to close %v: %v", reflect.TypeOf(c), err)
	}
}

func RecordVideoToDisk(videoDevice int, seconds int, outputFilename string) error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	recorder := NewRecorder(videoDevice, outputFilename)
	if err := recorder.Open(); err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}

	frame := gocv.NewMat()
	begin := time.Now()
	last := time.Now()
	//recorder.Begin()
	frameCount := int64(recorder.fps * float64(seconds))
	var i int64
	for i = 0; i < frameCount; i++ {
		//if err := recorder.Next(); err != nil {
		//	logrus.Errorf("error: %v", err)
		//	return err
		//}
		s := time.Now()

		if ok := recorder.webcam.Read(&frame); !ok {
			return fmt.Errorf("unable to read from webcam. device closed: %+v", recorder.videoDevice)
		}
		logrus.Debugf("read took: %v", time.Now().Sub(s))

		if frame.Empty() {
			logrus.Debug("skipping empty frame from webcam")
			continue
		}

		logrus.Debugf("loop took: %v", time.Now().Sub(last))
		last = time.Now()

		s = time.Now()
		//recorder.webcam.Grab(1)
		//logrus.Debugf("grab took: %v", time.Now().Sub(s))

		if !recorder.foundFace {
			go func(r *Recorder, cloneFrame gocv.Mat) {
				s = time.Now()
				if !r.foundFace {
					rects := r.classifier.DetectMultiScaleWithParams(frame, 1.1, 4, 0,
						image.Point{X: int(videoWidth / 10), Y: int(videoHeight / 10)}, image.Point{X: int(videoWidth / 4), Y: int(videoHeight / 4)})
					logrus.Debugf("face detect took: %v", time.Now().Sub(s))
					if len(rects) > 0 {
						logrus.Debugf("Detected %v face(s)", len(rects))
						prefix := strings.TrimSuffix(r.outputFilename, VideoExtension)
						gocv.IMWrite(fmt.Sprintf("%s.face.jpg", prefix), cloneFrame)
						for i, rect := range rects {
							gocv.IMWrite(fmt.Sprintf("%s.face.%d.jpg", prefix, i), cloneFrame.Region(rect))
						}
						r.foundFace = true
						logrus.Debugf("face detect write took: %v", time.Now().Sub(s))
					}
				}
			}(recorder, frame.Clone())
		}

		s = time.Now()
		if err := recorder.writer.Write(frame); err != nil {
			logrus.Errorf("error occurred while writing video to disk: %v", err)
		}
		logrus.Debugf("write took: %v", time.Now().Sub(s))
	}
	//recorder.Wait()
	recorder.Close()

	logrus.Debugf("recording took %v", time.Now().Sub(begin))

	return nil
}
