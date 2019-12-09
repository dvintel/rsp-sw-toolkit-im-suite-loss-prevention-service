/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
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

// Validate implements the jsonrpc.Message interface
func (dataPayload *DataPayload) Validate() error {
	// todo
	return nil
}
