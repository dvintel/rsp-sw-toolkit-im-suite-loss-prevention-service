/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package sensor

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/intel/rsp-sw-toolkit-im-suite-loss-prevention-service/app/config"
	"github.com/intel/rsp-sw-toolkit-im-suite-loss-prevention-service/pkg/jsonrpc"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	GetBasicInfo  = "sensor_get_basic_info"
	GetDeviceIds  = "sensor_get_device_ids"
	RspController = "rsp-controller"
)

var (
	sensors = make(map[string]*RSP)
)

// FindByAntennaAlias is a backwards lookup of an alias to the sensor (RSP) it belongs to
// Note that if more than one sensor has the same alias, it will just return the first match
func FindByAntennaAlias(alias string) *RSP {
	for _, rsp := range sensors {
		for _, a := range rsp.Aliases {
			if a == alias {
				return rsp
			}
		}
	}
	return nil
}

// GetOrQueryRSPInfo returns a pointer to an RSP if found in memory, otherwise will
// query the command service for that info
func GetOrQueryRSPInfo(deviceId string) (*RSP, error) {
	var err error
	var info *jsonrpc.SensorBasicInfo

	rsp, ok := sensors[deviceId]
	if !ok {
		rsp = NewRSP(deviceId)

		// this is a new sensor, try and obtain the actual info from the RSP Controller
		info, err = QueryBasicInfo(deviceId)
		if err != nil {
			// warn, we still want to put it in the database
			logrus.Warn(errors.Wrapf(err, "unable to query sensor basic info for device %s", deviceId))
		} else {
			rsp.Personality = Personality(info.Personality)
			rsp.Aliases = info.Aliases
			rsp.FacilityId = info.FacilityId
		}
		sensors[deviceId] = rsp
	}

	return rsp, err
}

func UpdateRSP(rsp *RSP) {
	sensors[rsp.DeviceId] = rsp
}

// QueryBasicInfoAllSensors retrieves the list of deviceIds from the RSP Controller
// and then queries the basic info for each one
func QueryBasicInfoAllSensors() error {
	var reading *models.Reading
	var err error

	// keep trying
	for err != nil || reading == nil {
		if reading, err = ExecuteSensorCommand(RspController, GetDeviceIds); err != nil {
			logrus.Errorf("Error trying to get list of device ids from controller: %v", err)
			time.Sleep(5 * time.Second)
		}
	}

	logrus.Debugf("ExecuteSensorCommand %v received: %s", GetDeviceIds, strings.ReplaceAll(strings.ReplaceAll(reading.Value, "\\", ""), "\"", "'"))

	deviceIds := new(jsonrpc.SensorDeviceIdsResponse)
	if err := jsonrpc.Decode(reading.Value, deviceIds, nil); err != nil {
		logrus.Errorf("unable to decode SensorDeviceIdsResponse")
		return err
	}

	logrus.Debugf("successfully queried list of device ids from RSP Controller")

	for _, deviceId := range *deviceIds {
		go ForceRefreshSensorInfo(deviceId)
	}

	return nil
}

func ForceRefreshSensorInfo(deviceId string) {
	var err error
	var info *jsonrpc.SensorBasicInfo

	for err != nil || info == nil {
		if info, err = QueryBasicInfo(deviceId); err != nil {
			logrus.Errorf("Error trying to get sensor info for device %v, %v", deviceId, err)
			time.Sleep(5 * time.Second)
		}
	}

	rsp := NewRSP(deviceId)
	rsp.Personality = Personality(info.Personality)
	rsp.Aliases = info.Aliases
	rsp.FacilityId = info.FacilityId
	sensors[deviceId] = rsp
	logrus.Infof("got rsp info: %+v", rsp)
}

// QueryBasicInfo makes a call to the EdgeX command service to request the RSP-Controller
// to return us more information about a given RSP sensor
func QueryBasicInfo(deviceId string) (*jsonrpc.SensorBasicInfo, error) {
	reading, err := ExecuteSensorCommand(deviceId, GetBasicInfo)
	if err != nil {
		return nil, err
	}

	sensorInfo := new(jsonrpc.SensorBasicInfo)
	if err := jsonrpc.Decode(reading.Value, sensorInfo, nil); err != nil {
		return nil, err
	}

	return sensorInfo, nil
}

// ExecuteSensorCommand makes an HTTP GET call to the EdgeX core command service to execute
// a specified command on a given RSP sensor
func ExecuteSensorCommand(deviceId string, commandName string) (*models.Reading, error) {
	url := fmt.Sprintf("%s/api/v1/device/name/%s/command/%s", config.AppConfig.CoreCommandUrl, deviceId, commandName)
	logrus.Infof("Making GET call to %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.Errorf("Http call returned status code %v: %v  for url: %s", resp.StatusCode, resp.Status, url)
	} else {
		logrus.Debugf("Http call returned status code %v: %v  for url: %s", resp.StatusCode, resp.Status, url)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	logrus.Info(strings.ReplaceAll(strings.ReplaceAll(string(body), "\\", ""), "\"", "'"))

	evt := new(models.Event)
	if err := evt.UnmarshalJSON(body); err != nil {
		return nil, err
	}

	if len(evt.Readings) == 0 {
		return nil, errors.New("response contained no reading values!")
	}

	return &evt.Readings[0], nil
}
