/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
*/

package camera

import (
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"path/filepath"
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
}

type Cascade struct {
	name         string
	found        int
	written      int
	drawOptions  DrawOptions
	detectParams DetectParams
	classifier   *gocv.CascadeClassifier
}

func (cascadeFile CascadeFile) AsNewCascade(classifier *gocv.CascadeClassifier) *Cascade {
	return &Cascade{
		name:         cascadeFile.name,
		detectParams: cascadeFile.detectParams,
		drawOptions:  cascadeFile.drawOptions,
		classifier:   classifier,
	}
}

type FrameOverlay struct {
	rect        image.Rectangle
	drawOptions DrawOptions
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

	webcam *gocv.VideoCapture
	writer *gocv.VideoWriter
	window *gocv.Window
	//net	       gocv.Net

	frame        gocv.Mat
	processFrame gocv.Mat

	overlays []FrameOverlay
	cascades []*Cascade
}

func NewRecorder(videoDevice string, outputFolder string, liveView bool) *Recorder {
	recorder := &Recorder{
		videoDevice:    videoDevice,
		outputFolder:   outputFolder,
		outputFilename: filepath.Join(outputFolder, "video"+config.AppConfig.VideoOutputExtension),
		width:          config.AppConfig.VideoResolutionWidth,
		height:         config.AppConfig.VideoResolutionHeight,
		liveView:       liveView,
		fps:            float64(config.AppConfig.VideoOutputFps),
		codec:          config.AppConfig.VideoOutputCodec,
		window:         gocv.NewWindow(config.AppConfig.ServiceName + " - OpenVINO"),
		frame:          gocv.NewMat(),
		processFrame:   gocv.NewMat(),
	}

	return recorder
}
