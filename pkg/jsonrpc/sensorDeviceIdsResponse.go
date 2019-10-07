package jsonrpc

type SensorDeviceIdsResponse []string

func (info *SensorDeviceIdsResponse) Validate() error {
	return nil
}
