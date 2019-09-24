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

import "time"

type DataPayload struct {
	ControllerId       string `json:"device_id"` //backend expects this to be "device_id" instead of controller_id
	SentOn             int64  `json:"sent_on"`
	TotalEventSegments int    `json:"total_event_segments"`
	EventSegmentNumber int    `json:"event_segment_number"`
	TagEvent           []Tag  `json:"data"`
}

// Tag is the model containing items for a Tag
//swagger:model Tag
type Tag struct {
	// URI string representation of tag
	URI string `json:"uri"`
	// SGTIN EPC code
	Epc string `json:"epc"`
	// ProductID
	ProductID string `json:"product_id" bson:"product_id"`
	// Part of EPC, denotes packaging level of the item
	FilterValue int64 `json:"filter_value" bson:"filter_value"`
	// Tag manufacturer ID
	Tid string `json:"tid"`
	// TBD
	EpcEncodeFormat string `json:"encode_format" bson:"encode_format"`
	// Facility ID
	FacilityID string `json:"facility_id" bson:"facility_id"`
	// Last event recorded for tag
	Event string `json:"event"`
	// Arrival time in milliseconds epoch
	Arrived int64 `json:"arrived"`
	// Tag last read time in milliseconds epoch
	LastRead int64 `json:"last_read" bson:"last_read"`
	// Where tags were read from (fixed or handheld)
	Source string `json:"source"`
	// Array of objects showing history of the tag's location
	LocationHistory []LocationHistory `json:"location_history" bson:"location_history"`
	// Current state of tag, either ’present’ or ’departed’
	EpcState string `json:"epc_state" bson:"epc_state"`
	// Customer defined state
	QualifiedState string `json:"qualified_state" bson:"qualified_state"`
	// Time to live, used for db purging - always in sync with last read
	TTL time.Time `json:"ttl"`
	// Customer defined context
	EpcContext string `json:"epc_context" bson:"epc_context"`
	// Probability item is actually present
	Confidence float64 `json:"confidence,omitempty"` //omitempty - confidence is not stored in the db
	// Cycle Count indicator
	CycleCount bool `json:"-"`
}

// LocationHistory is the model to record the whereabouts history of a tag
type LocationHistory struct {
	Location  string `json:"location"`
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source"`
}
