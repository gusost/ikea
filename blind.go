package ikea

import (
	"fmt"
	"log"
)

func SetBlindPosition(deviceId int, position float32) error {
	device, err := GetDevice(deviceId)

	if err != nil {
		return fmt.Errorf("error getting device state: %+v", err)
	}

	if device.Type != 7 {
		return fmt.Errorf("device is not a blind: %+v", err)
	}

	if device.BlindControl[0].Position != position {

		_, err := client.PutDevicePositioning(deviceId, position)
		if err != nil {
			return fmt.Errorf("error setting blind: %+v", err)
		}
		log.Printf("%v set to %.2f\n", device.Name, position)
	}
	return nil
}
