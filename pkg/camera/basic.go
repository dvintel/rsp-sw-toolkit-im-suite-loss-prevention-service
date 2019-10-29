package camera

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"gocv.io/x/gocv"
	"image"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
)

func RecordVideoToDiskBasic(videoDevice string, seconds int, outputFolder string) error {
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
	token := recorder.NewFrameToken(0)
	var rects []image.Rectangle
	var cascade CascadeFile
	var params DetectParams
	// for debug stats
	var read, process, total DebugStats
	var prevMillis, currentMills int64
	// pre-compute the x location of the avg stats
	x2 := gocv.GetTextSize("Avg Process: 99.9", font, fontScale, fontThickness).X
	x3 := gocv.GetTextSize("Min Process: 99", font, fontScale, fontThickness).X + x2 + 60
	yPadding := 35
	yStart := 0

	for i := 0; i < recorder.frameCount; i++ {
		token.index = i
		token.startTS = helper.UnixMilliNow()

		if ok := recorder.webcam.Read(&token.frame); !ok {
			return fmt.Errorf("unable to read from webcam. device closed: %+v", recorder.videoDevice)
		}
		token.readTS = helper.UnixMilliNow()

		if token.frame.Empty() {
			logrus.Debug("skipping empty frame from webcam")
			continue
		}

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

		// Resize smaller for use with the cascade classifiers
		gocv.Resize(token.frame, &token.procFrame, image.Point{}, 1.0/float64(config.AppConfig.ImageProcessScale), 1.0/float64(config.AppConfig.ImageProcessScale), gocv.InterpolationLinear)

		token.overlays = nil
		for _, cascadeQueue := range recorder.cascadeQueues {

			cascade = cascadeQueue.cascadeFile
			params = cascade.detectParams

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

				for _, rect := range rects {
					token.overlays = append(token.overlays, FrameOverlay{rect: transformProcessRect(rect), drawOptions: cascade.drawOptions})
				}
			}

		}

		token.processedTS = helper.UnixMilliNow()

		if recorder.liveView {
			if config.AppConfig.ShowVideoDebugStats {
				read.AddValue(float64(token.readTS - token.startTS))
				process.AddValue(float64(token.processedTS - token.readTS))
				currentMills = helper.UnixMilliNow()
				if prevMillis != 0 {
					total.AddValue(float64(currentMills - prevMillis))
				}
				prevMillis = currentMills

				// Instant
				gocv.PutText(&token.frame, "   Read: "+strconv.FormatInt(int64(read.current), 10),
					image.Point{textPadding, yStart + (yPadding * 1)}, font, fontScale, debugStatsColor, fontThickness)
				gocv.PutText(&token.frame, "Process: "+strconv.FormatInt(int64(process.current), 10),
					image.Point{textPadding, yStart + (yPadding * 2)}, font, fontScale, debugStatsColor, fontThickness)
				gocv.PutText(&token.frame, "    FPS: "+strconv.FormatFloat(total.FPS(), 'f', 1, 64),
					image.Point{textPadding, yStart + (yPadding * 3)}, font, fontScale, debugStatsColor, fontThickness)

				// Min / Max
				gocv.PutText(&token.frame, "   Min Read: "+strconv.FormatInt(int64(read.min), 10),
					image.Point{x2, yStart + (yPadding * 1)}, font, fontScale, debugStatsColor, fontThickness)
				gocv.PutText(&token.frame, "   Max Read: "+strconv.FormatInt(int64(read.max), 10),
					image.Point{x2, yStart + (yPadding * 2)}, font, fontScale, debugStatsColor, fontThickness)
				gocv.PutText(&token.frame, "Min Process: "+strconv.FormatInt(int64(process.min), 10),
					image.Point{x2, yStart + (yPadding * 3)}, font, fontScale, debugStatsColor, fontThickness)
				gocv.PutText(&token.frame, "Max Process: "+strconv.FormatInt(int64(process.max), 10),
					image.Point{x2, yStart + (yPadding * 4)}, font, fontScale, debugStatsColor, fontThickness)

				// Average
				gocv.PutText(&token.frame, "   Avg Read: "+strconv.FormatFloat(read.Average(), 'f', 1, 64),
					image.Point{x3, yStart + (yPadding * 1)}, font, fontScale, debugStatsColor, fontThickness)
				gocv.PutText(&token.frame, "Avg Process: "+strconv.FormatFloat(process.Average(), 'f', 1, 64),
					image.Point{x3, yStart + (yPadding * 2)}, font, fontScale, debugStatsColor, fontThickness)
				gocv.PutText(&token.frame, "    Avg FPS: "+strconv.FormatFloat(total.AverageFPS(), 'f', 1, 64),
					image.Point{x3, yStart + (yPadding * 3)}, font, fontScale, debugStatsColor, fontThickness)

			}

			for _, overlay := range token.overlays {
				if overlay.drawOptions.renderAsCircle {
					radius := (overlay.rect.Max.X - overlay.rect.Min.X) / 2
					gocv.Circle(&token.frame, image.Point{overlay.rect.Max.X - radius, overlay.rect.Max.Y - radius}, radius, overlay.drawOptions.color, overlay.drawOptions.thickness)
				} else {
					gocv.Rectangle(&token.frame, overlay.rect, overlay.drawOptions.color, overlay.drawOptions.thickness)
				}
				gocv.PutText(&token.frame, overlay.drawOptions.annotation, image.Point{overlay.rect.Min.X, overlay.rect.Min.Y - 10}, font, 1, overlay.drawOptions.color, fontThickness)
			}

			recorder.window.IMShow(token.frame)
			key := recorder.window.WaitKey(1)

			// ESC, Q, q
			if key == 27 || key == 'q' || key == 'Q' {
				logrus.Debugf("stopping video live view")
				recorder.liveView = false
				safeClose(recorder.window)
				break
			}
		}
	}

	end := recorder.Wait()
	logrus.Debugf("recording took %v", end.Sub(begin))

	logrus.Infof("Video url: %s%s", config.AppConfig.VideoUrlBase, url.QueryEscape(recorder.outputFilename))

	safeClose(&token.frame)
	safeClose(&token.procFrame)

	return nil
}
