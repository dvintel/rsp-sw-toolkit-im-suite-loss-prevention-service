/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
*/

package jsonrpc

type SensorDeviceIdsResponse []string

func (info *SensorDeviceIdsResponse) Validate() error {
	return nil
}
