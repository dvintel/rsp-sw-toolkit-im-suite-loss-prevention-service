/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
*/

package webserver

import "github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"

type RecordingsResponse struct {
	BaseUrl     string          `json:"base_url"`
	ThumbHeight int             `json:"thumb_height"`
	Recordings  []RecordingInfo `json:"recordings"`
}

func NewRecordingsResponse(count int) RecordingsResponse {
	return RecordingsResponse{
		BaseUrl:     config.AppConfig.VideoUrlBase,
		ThumbHeight: config.AppConfig.ThumbnailHeight,
		Recordings:  make([]RecordingInfo, count),
	}
}

type RecordingInfo struct {
	FolderName string   `json:"folder_name"`
	EPC        string   `json:"epc"`
	ProductId  string   `json:"product_id"`
	Timestamp  int64    `json:"timestamp"`
	Video      string   `json:"video"`
	Thumb      string   `json:"thumb"`
	Detections []string `json:"detections"`
}
