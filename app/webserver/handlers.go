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
package webserver

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/pkg/web"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	baseFolder = "/recordings"
)

// Handler represents the User API method handler set.
type Handler struct {
	ServiceName string
}

// Index is used for Docker Healthcheck commands to indicate
// whether the http server is up and running to take requests
//nolint:unparam
func (handler *Handler) Index(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	web.Respond(ctx, writer, handler.ServiceName, http.StatusOK)
	return nil
}

// ListRecordings will return a json array of recording filenames
//nolint:unparam
func (handler *Handler) ListRecordings(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	// todo: limit recording history, or use pagination
	folders, err := ioutil.ReadDir(baseFolder)
	if err != nil {
		logrus.Error(err)
		web.Respond(ctx, writer, "Internal Error", http.StatusInternalServerError)
	}

	resp := NewRecordingsResponse(len(folders))
	for i, folder := range folders {
		tokens := strings.Split(folder.Name(), "_")
		if len(tokens) != 3 {
			logrus.Warnf("folder: %s does not appear to match expected format! skipping.", folder.Name())
			continue
		}
		ts, err := strconv.ParseInt(tokens[0], 10, 64)
		if err != nil {
			logrus.Warnf("unable to parse timestamp from folder name: %v", err)
			continue
		}
		files, err := ioutil.ReadDir(filepath.Join(baseFolder, folder.Name()))
		if err != nil {
			logrus.Warnf("unable to read recording directory %s: %v", folder.Name(), err)
			continue
		}

		info := RecordingInfo{
			FolderName: folder.Name(),
			Timestamp: ts,
			ProductId: tokens[1],
			EPC: tokens[2],
			Video: "video" + config.AppConfig.VideoOutputExtension,
			Thumb: "thumb.jpg",
		}
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".jpg") && file.Name() != "thumb.jpg" && !strings.HasPrefix(file.Name(), "frame.") {
				info.Detections = append(info.Detections, file.Name())
			}
		}

		resp.Recordings[i] = info
	}
	logrus.Tracef("%+v", resp)
	web.Respond(ctx, writer, resp, http.StatusOK)
	return nil
}
