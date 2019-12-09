/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
*/

package webserver

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/pkg/web"
	"io/ioutil"
	"net/http"
	"os"
	"path"
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
		return err
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
			Timestamp:  ts,
			ProductId:  tokens[1],
			EPC:        tokens[2],
			Video:      "video" + config.AppConfig.VideoOutputExtension,
			Thumb:      "thumb.jpg",
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

func (handler *Handler) DeleteRecording(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	vars := mux.Vars(request)
	folder, ok := vars["foldername"]
	if !ok {
		web.Respond(ctx, writer, "Bad Request", http.StatusBadRequest)
		return fmt.Errorf("bad request")
	}

	if err := os.RemoveAll(path.Join(baseFolder, folder)); err != nil {
		logrus.Error(err)
		web.Respond(ctx, writer, "Internal Error", http.StatusInternalServerError)
		return err
	}

	web.Respond(ctx, writer, nil, http.StatusOK)
	return nil
}

func (handler *Handler) DeleteAllRecordings(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	folders, err := ioutil.ReadDir(baseFolder)
	if err != nil {
		logrus.Error(err)
		web.Respond(ctx, writer, "Internal Error", http.StatusInternalServerError)
		return err
	}

	for _, folder := range folders {
		if err := os.RemoveAll(path.Join(baseFolder, folder.Name())); err != nil {
			logrus.Error(err)
			web.Respond(ctx, writer, "Internal Error", http.StatusInternalServerError)
			return err
		}
	}

	web.Respond(ctx, writer, nil, http.StatusOK)
	return nil
}

func (handler *Handler) Options(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	web.Respond(ctx, writer, nil, http.StatusOK)
	return nil
}
