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
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"gocv.io/x/gocv"
	"io"
	"math"
	"reflect"
	"sync"
)

const (
	fps     = 25
	xmlFile = "./res/docker/haarcascade_frontalface_default.xml"
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
		width:          640,
		height:         480,
		fps:            fps,
		codec:          "MJPG",
		writeBuffer:    make(chan FrameToken, 100),
		faceBuffer:     make(chan FrameToken, 100),
		done:           make(chan bool),
	}

	return recorder
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

	//r.webcam.Set(gocv.VideoCaptureFrameWidth, float64(r.width))
	//r.webcam.Set(gocv.VideoCaptureFrameHeight, float64(r.height))
	//r.webcam.Set(gocv.VideoCaptureFPS, r.fps)

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	r.classifier = &classifier
	if !r.classifier.Load(xmlFile) {
		return fmt.Errorf("error reading cascade file: %v", xmlFile)
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
	go r.ProcessWriteQueue(r.done)
	go r.ProcessFaceQueue(r.done)
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

func (r *Recorder) ProcessWriteQueue(done chan bool) error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered from panic: %+v", r)
		}
	}()

	var err error
	r.writer, err = gocv.VideoWriterFile(r.outputFilename, r.codec, r.fps, r.width, r.height, true)
	if err != nil {
		return errors.Wrapf(err, "error opening video writer device: %+v", r.outputFilename)
	}

	logrus.Debug("ProcessWriteQueue()")
	for {
		select {
		case <-done:
			logrus.Debug("ProcessWriteQueue() completed")
			return nil

		case token := <-r.writeBuffer:
			logrus.Debug("writeBuffer token")
			if err := r.writer.Write(*token.frame); err != nil {
				logrus.Errorf("error occurred while writing video to disk: %v", err)
			}
			logrus.Debug("writeBuffer token complete")
			//token.waitGroup.Done()
		}
	}
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
		case <- done:
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

	begin := helper.UnixMilliNow()

	recorder.Begin()
	frameCount := int(math.Round(recorder.fps * float64(seconds)))
	for i := 0; i < frameCount; i++ {
		if err := recorder.Next(); err != nil {
			logrus.Errorf("error: %v", err)
			return err
		}
	}
	recorder.Wait()
	recorder.Close()

	diff := helper.UnixMilliNow() - begin
	logrus.Debugf("recording took %v mills", diff)

	return nil
}
