package sensor

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/pkg/jsonrpc"
	"io/ioutil"
	"net/http"
	"strings"
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

// GetOrCreateRSP returns a pointer to an RSP if found in the DB, and if
// not found in the DB, a record will be created and added, then returned to the caller
// error is only non-nil when there is an issue communicating with the DB
func GetOrCreateRSP(deviceId string) (*RSP, error) {
	rsp, ok := sensors[deviceId]
	if !ok {
		rsp = NewRSP(deviceId)

		// this is a new sensor, try and obtain the actual info from the RSP Controller
		info, err := QueryBasicInfo(deviceId)
		if err != nil {
			// just warn, we still want to put it in the database
			logrus.Warn(errors.Wrapf(err, "unable to query sensor basic info for device %s", deviceId))
		} else {
			// update the info before upserting
			rsp.Personality = Personality(info.Personality)
			rsp.Aliases = info.Aliases
			rsp.FacilityId = info.FacilityId
		}
		sensors[deviceId] = rsp
	}

	return rsp, nil
}

func UpdateRSP(rsp *RSP) {
	sensors[rsp.DeviceId] = rsp
}

// QueryBasicInfoAllSensors retrieves the list of deviceIds from the RSP Controller
// and then queries the basic info for each one
func QueryBasicInfoAllSensors() error {
	reading, err := ExecuteSensorCommand(RspController, GetDeviceIds)
	if err != nil {
		logrus.Error(err)
		return err
	}

	logrus.Debugf("received: %s", strings.ReplaceAll(strings.ReplaceAll(reading.Value, "\\", ""), "\"", "'"))

	deviceIds := new(jsonrpc.SensorDeviceIdsResponse)
	if err := jsonrpc.Decode(reading.Value, deviceIds, nil); err != nil {
		return err
	}

	for _, deviceId := range *deviceIds {
		rsp, _ := GetOrCreateRSP(deviceId)
		if rsp != nil {
			logrus.Debugf("rsp: %+v", rsp)
		}
	}

	return nil
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
