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
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"path/filepath"
	"sync"
)

type DebugStats struct {
	min     float64
	max     float64
	current float64
	total   float64
	count   float64
}

func (stats *DebugStats) AddValue(val float64) {
	stats.current = val
	stats.total += val
	stats.count++

	if val > stats.max {
		stats.max = val
	}
	if stats.min == 0 || val < stats.min {
		stats.min = val
	}
}

func (stats *DebugStats) Average() float64 {
	if stats.count == 0 {
		return 0
	}
	return stats.total / stats.count
}

func (stats *DebugStats) AverageFPS() float64 {
	if stats.Average() == 0 {
		return 0
	}
	return 1.0 / (stats.Average() / 1000.0)
}

func (stats *DebugStats) FPS() float64 {
	if stats.current == 0 {
		return 0
	}
	return 1.0 / (stats.current / 1000.0)
}

type DrawOptions struct {
	annotation     string
	color          color.RGBA
	thickness      int
	renderAsCircle bool
}

type DetectParams struct {
	scale        float64
	minNeighbors int
	flags        int
	minScaleX    float64
	minScaleY    float64
	maxScaleX    float64
	maxScaleY    float64
}

type CascadeFile struct {
	name         string
	filename     string
	drawOptions  DrawOptions
	detectParams DetectParams
	classifier   *gocv.CascadeClassifier
}

type CascadeQueue struct {
	cascadeFile CascadeFile
	found       int
	buffer      chan *FrameToken
}

func (cascadeFile CascadeFile) CreateQueue(bufferSize int) *CascadeQueue {
	return &CascadeQueue{
		cascadeFile: cascadeFile,
		buffer:      make(chan *FrameToken, bufferSize),
	}
}

type FrameToken struct {
	startTS      int64
	readTS       int64
	processedTS  int64
	frame        gocv.Mat
	procFrame    gocv.Mat
	waitGroup    sync.WaitGroup
	overlays     []FrameOverlay
	overlayMutex sync.Mutex
	index        int
}

type FrameOverlay struct {
	rect        image.Rectangle
	drawOptions DrawOptions
}

func (recorder *Recorder) NewFrameToken(index int) *FrameToken {
	return &FrameToken{
		startTS:   helper.UnixMilliNow(),
		frame:     gocv.NewMat(),
		procFrame: gocv.NewMat(),
		index:     index,
	}
}

type Recorder struct {
	videoDevice    string
	outputFolder   string
	outputFilename string
	fps            float64
	frameCount     int
	codec          string
	width          int
	height         int
	liveView       bool
	waitGroup      sync.WaitGroup
	done           chan bool

	webcam        *gocv.VideoCapture
	cascadeQueues []*CascadeQueue
	writer        *gocv.VideoWriter
	window        *gocv.Window
	//net	       gocv.Net

	writeBuffer chan *FrameToken
	waitBuffer  chan *FrameToken
	//vinoBuffer  chan *FrameToken
	closeBuffer chan *FrameToken
}

func NewRecorder(videoDevice string, outputFolder string) *Recorder {
	recorder := &Recorder{
		videoDevice:    videoDevice,
		outputFolder:   outputFolder,
		outputFilename: filepath.Join(outputFolder, "video"+config.AppConfig.VideoOutputExtension),
		width:          config.AppConfig.VideoResolutionWidth,
		height:         config.AppConfig.VideoResolutionHeight,
		liveView:       config.AppConfig.LiveView,
		fps:            float64(config.AppConfig.VideoOutputFps),
		codec:          config.AppConfig.VideoOutputCodec,
		window:         gocv.NewWindow(config.AppConfig.ServiceName + " - OpenVINO"),

		writeBuffer: make(chan *FrameToken, 25),
		waitBuffer:  make(chan *FrameToken, config.AppConfig.VideoOutputFps*config.AppConfig.RecordingDuration),
		closeBuffer: make(chan *FrameToken, config.AppConfig.VideoOutputFps*config.AppConfig.RecordingDuration),
		done:        make(chan bool),
	}

	return recorder
}
