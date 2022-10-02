package ikea

import (
	"fmt"
	"log"
)

func GetBlindPosition(deviceId int, position float32) (int, error) {
	device, err := GetDevice(deviceId)

	if err != nil {
		return 0, fmt.Errorf("error getting device state: %+v", err)
	}

	if device.Type != 7 {
		return 0, fmt.Errorf("device is not a blind: %+v", err)
	}

	return int(device.BlindControl[0].Position), nil
}

func SetBlindPosition(deviceId int, position float32) error {
	device, err := GetDevice(deviceId)

	if err != nil {
		return fmt.Errorf("error getting device state: %+v", err)
	}

	if device.Type != 7 {
		return fmt.Errorf("device is not a blind: %+v", err)
	}
	// Sometimes blinds errouneously report position = 0 even though it's not
	if device.BlindControl[0].Position != position || position == 0 {

		_, err := client.PutDevicePositioning(deviceId, position)
		if err != nil {
			return fmt.Errorf("error setting blind: %+v", err)
		}
		log.Printf("%v set to %.2f\n", device.Name, position)
	}
	return nil
}
