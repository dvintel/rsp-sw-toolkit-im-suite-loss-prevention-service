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
)

const (
	fps     = 25
	xmlFile = "./res/docker/haarcascade_frontalface_default.xml"
)

func RecordVideoToDisk(videoDevice int, seconds int, outputFilename string) error {
	webcam, err := gocv.OpenVideoCapture(videoDevice)
	if err != nil {
		return errors.Wrapf(err, "Error opening video capture device: %+v", videoDevice)
	}
	defer webcam.Close()

	img := gocv.NewMat()
	defer img.Close()

	if ok := webcam.Read(&img); !ok {
		return fmt.Errorf("Cannot read device %+v\n", videoDevice)
	}

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()

	if !classifier.Load(xmlFile) {
		return fmt.Errorf("error reading cascade file: %v", xmlFile)
	}

	writer, err := gocv.VideoWriterFile(outputFilename, "MJPG", fps, img.Cols(), img.Rows(), true)
	if err != nil {
		return errors.Wrapf(err, "error opening video writer device: %+v", outputFilename)
	}
	defer writer.Close()

	foundFace := false

	frameCount := fps * seconds
	for i := 0; i < frameCount; i++ {
		if ok := webcam.Read(&img); !ok {
			return fmt.Errorf("unable to read from webcam. device closed: %+v", videoDevice)
		}
		if img.Empty() {
			continue
		}

		if !foundFace {
			rects := classifier.DetectMultiScale(img)
			if len(rects) > 0 {
				logrus.Debugf("Detected %v face(s)", len(rects))
				gocv.IMWrite(fmt.Sprintf("%s.face.jpg", outputFilename), img)
				for i, rect := range rects {
					gocv.IMWrite(fmt.Sprintf("%s.face.%d.jpg", outputFilename, i), img.Region(rect))
				}
				foundFace = true
			}
		}

		if err := writer.Write(img); err != nil {
			return errors.Wrapf(err, "error occurred while writing video to disk.")
		}
	}

	return nil
}
