package ikea

import (
	"fmt"
	"log"
)

func IsOutletOn(deviceId int) bool {
	device, err := GetDevice(deviceId)

	if err != nil {
		log.Printf("error getting outlet state: %+v\n", err)
		return false
	}
	if device.Type != 3 {
		log.Printf("device is not an outlet: %+v\n", err)
		return false
	}

	return device.OutletControl[0].Power == 1
}

func TurnOutletOn(deviceId int) error {
	return SetOutletPowerState(deviceId, 1)
}
func TurnOutletOff(deviceId int) error {
	return SetOutletPowerState(deviceId, 0)
}

func SetOutletPowerState(deviceId, powerState int) error {
	device, err := GetDevice(deviceId)

	if err != nil {
		return fmt.Errorf("error getting device state: %+v", err)
	}

	if device.Type != 3 {
		return fmt.Errorf("device is not an outlet: %+v", err)
	}

	if device.OutletControl[0].Power != powerState {

		_, err := client.PutDevicePower(deviceId, powerState)
		if err != nil {
			return fmt.Errorf("error setting device: %+v", err)
		}
		if powerState == 1 {
			log.Printf("%v turned on\n", device.Name)
		} else {
			log.Printf("%v turned off\n", device.Name)
		}
	}
	return nil
}
