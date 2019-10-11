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

package lossprevention

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/pkg/camera"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/pkg/sensor"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

const (
	moved = "moved"

	seconds          = 15
	videoFilePattern = "/recordings/recording_%s_%v%s"
)

func HandleDataPayload(payload *DataPayload) error {

	for _, tag := range payload.TagEvent {
		if tag.Event != moved || len(tag.LocationHistory) < 2 {
			//logrus.Debugf("skipping: %+v", tag)
			continue
		}

		logrus.Debugf("location  current:  %+v", tag.LocationHistory[0])
		logrus.Debugf("location previous:  %+v", tag.LocationHistory[1])

		rsp := sensor.FindByAntennaAlias(tag.LocationHistory[0].Location)
		logrus.Debugf("rsp  current: %+v", rsp)
		if rsp != nil && rsp.IsExitSensor() {
			rsp2 := sensor.FindByAntennaAlias(tag.LocationHistory[1].Location)
			logrus.Debugf("rsp previous: %+v", rsp2)
			if rsp2 != nil && !rsp2.IsExitSensor() {
				// return so we do not keep checking
				return triggerRecord(tag.ProductID)
			}
		}

	}

	return nil
}

func triggerRecord(productId string) error {

	filename := fmt.Sprintf(videoFilePattern, productId, helper.UnixMilliNow(), config.AppConfig.VideoOutputExtension)
	logrus.Debugf("recording filename: %s", filename)
	go camera.RecordVideoToDisk(config.AppConfig.VideoDevice, seconds, filename)

	return nil
}
