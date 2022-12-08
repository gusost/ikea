package ikea

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"sort"
	"time"

	"github.com/eriklupander/tradfri-go/model"
	"github.com/eriklupander/tradfri-go/tradfri"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type IKEAGateway struct {
	ClientID string `json:"clientId"`
	Code     string `json:"code"`
	IP       string `json:"ip"`
	Mac      string `json:"mac"`
	PSK      string `json:"psk"`
	Serial   string `json:"serial"`
}

type TradfriConfig struct {
	ClientID       string `json:"client_id"`
	GatewayAddress string `json:"gateway_address"`
	GatewayIP      string `json:"gateway_ip"`
	Loglevel       string `json:"loglevel"`
	PreSharedKey   string `json:"pre_shared_key"`
	PSK            string `json:"psk"`
}

var client *tradfri.Client

func firstIntitGateway() error {
	//err := fmt.Errorf("initial key exchange with gateway not implemented yet. It needs to be done to get a valid key associated with your client ID (key erroneously called pre shared in the TradfriConfig struct)")

	ikeaKeyfile, err := ioutil.ReadFile("./ikea.json.key")
	if err != nil {
		return fmt.Errorf("cannot read key file 'ikea.json.key': %s", err)
	}

	ikeaKey := &IKEAGateway{}

	err = json.Unmarshal(ikeaKeyfile, ikeaKey)
	if err != nil {
		return err
	}
	performTokenExchange(ikeaKey.IP+":5684", ikeaKey.ClientID, ikeaKey.PSK)
	return nil
}

func IntitGateway() error {

	level, _ := logrus.ParseLevel("error")
	//level, _ := logrus.ParseLevel("info")
	//level, _ := logrus.ParseLevel("trace")
	logrus.SetLevel(level)

	ikeaKeyfile, err := ioutil.ReadFile("./ikea.config.json.key")
	if err != nil {
		err = firstIntitGateway()
		if err != nil {
			return fmt.Errorf("cannot read key file 'ikea.config.json.key': %s", err)
		}

		ikeaKeyfile, err = ioutil.ReadFile("./ikea.config.json.key")

		if err != nil {
			return fmt.Errorf("cannot read key file 'ikea.config.json.key': %s", err)
		}
	}

	tradfriConfig := &TradfriConfig{}

	err = json.Unmarshal(ikeaKeyfile, tradfriConfig)

	if err != nil {
		return err
	}

	client = tradfri.NewTradfriClient(tradfriConfig.GatewayAddress, tradfriConfig.ClientID, tradfriConfig.PSK)

	return nil
}

func ListDevices() (string, error) {
	return ListDevicesWithDead(false)
}

func ListDevicesWithDead(includeDead bool) (string, error) {
	deviceList, err := client.ListDevices()
	if err != nil {
		return "", err
	}
	// Sort by id
	sort.Slice(deviceList, func(i, j int) bool { return deviceList[i].DeviceId < deviceList[j].DeviceId })

	// Sort by type
	sort.SliceStable(deviceList, func(i, j int) bool { return deviceList[i].Type < deviceList[j].Type })

	// Sort by alive
	sort.SliceStable(deviceList, func(i, j int) bool { return deviceList[i].Alive < deviceList[j].Alive })

	list := "  ID  -  Type  - State\n"
	for _, device := range deviceList {
		// Hide dead devices
		if !includeDead && device.Alive == 0 {
			continue
		}
		list += fmt.Sprintf("%v - ", device.DeviceId)
		switch device.Type {
		case 0: // Remote
			list += fmt.Sprintf("Remote - 🔋%v", printPercent(device.Metadata.Battery))
		case 3: // Outlet
			if device.OutletControl[0].Power == 0 {
				list += fmt.Sprintf("%-15v", "Outlet - Off")
			} else {
				list += fmt.Sprintf("%-15v", "Outlet - On")
			}
		case 4: // Motion
			list += fmt.Sprintf("Motion - 🔋%v", printPercent(device.Metadata.Battery))
		case 6: // Repeater
			list += fmt.Sprintf("%-15v", "Repeat -")
		case 7: // Blind
			list += fmt.Sprintf("Blind  - 📏%v - 🔋%v", printPercent(int(device.BlindControl[0].Position)), printPercent(device.Metadata.Battery))
		default:
			list += fmt.Sprintf("%v     ", device.Type)
		}
		// Alive
		/* if device.Alive == 1 {
			list += " - Alive"
		} else {
			list += " - Dead "
		} */

		// Time since seen
		since := time.Since(time.Unix(int64(device.LastSeen), 0))
		if since > 365*24*time.Hour {
			// print days since
			list += " - 👁         "
		} else if since > 7*24*time.Hour {
			list += fmt.Sprintf(" - 👁 %-3v days", math.Round(since.Hours()/24))
		} else if since > 24*time.Hour {
			days := math.Round(since.Hours() / 24)
			list += fmt.Sprintf(" - 👁 %-2v days %-2v hours", days, math.Round((since.Hours() - days*24)))
		} else if since > 2*time.Hour {
			list += fmt.Sprintf(" - 👁 %-2v hours", math.Round(since.Hours()))
		} else {
			list += " - 👁 Recently"
		}
		// Name
		list += fmt.Sprintf(" - %v\n", device.Name)
	}

	fmt.Println(list)

	return list, err
}

func ListDevicesBattery() (string, error) {
	deviceList, err := client.ListDevices()
	if err != nil {
		return "", err
	}
	// Sort by id
	sort.Slice(deviceList, func(i, j int) bool { return deviceList[i].Name < deviceList[j].Name })

	list := "Name                         -  🔋\n"
	for _, device := range deviceList {
		// Hide dead devices
		if device.Alive == 0 || (device.Type != 0 && device.Type != 4 && device.Type != 7) {
			continue
		}
		list += fmt.Sprintf("%-28v - ", device.Name)
		list += fmt.Sprintf("%v\n", printPercent(device.Metadata.Battery))
	}

	fmt.Println(list)

	return list, err
}

func printPercent(percent int) string {
	batteryString := fmt.Sprintf("%v%%", percent)
	return fmt.Sprintf("%-4v", batteryString) // Left align, padding 4
	// return fmt.Sprintf("%4v", batteryString) // Left align, padding 4
}

func GetDevice(deviceId int) (model.Device, error) {

	device, err := client.GetDevice(deviceId)

	if err != nil {
		// attempt a re-connect.
		fmt.Printf("error getting device: %+v\n", err)
		fmt.Println("Got an error, attempting to reconnect")
		IntitGateway()

		device, err = client.GetDevice(deviceId)
		if err != nil {
			return model.Device{}, fmt.Errorf("error getting device state: %+v", err)
		}
	}
	return device, nil
}

func GetDevices() ([]model.Device, error) {
	deviceList, err := client.ListDevices()

	if err != nil {
		// attempt a re-connect.
		fmt.Printf("error getting device list: %+v\n", err)
		fmt.Println("Got an error, attempting to reconnect")
		IntitGateway()

		deviceList, err = client.ListDevices()

		if err != nil {
			return []model.Device{}, fmt.Errorf("error getting device state: %+v", err)
		}
	}
	return deviceList, nil
}

func GetDevicesOfType(deviceType int) ([]model.Device, error) {
	deviceList, err := GetDevices()
	if err != nil {
		return []model.Device{}, err
	}

	deviceListOfType := make([]model.Device, 0)

	// Filter devices of the correct type.
	for i := range deviceList {
		if deviceList[i].Type == deviceType {
			deviceListOfType = append(deviceListOfType, deviceList[i])
		}
	}

	return deviceListOfType, nil
}

func performTokenExchange(gatewayAddress, clientID, psk string) {
	if len(clientID) < 1 || len(psk) < 10 {
		fail("Both clientID and psk args must be specified when performing key exchange")
	}

	done := make(chan bool)
	defer func() { done <- true }()
	go func() {
		select {
		case <-time.After(time.Second * 5):
			logrus.Info("(Please note that the key exchange may appear to be stuck at \"Connecting to peer at\" if the PSK from the bottom of your Gateway is not entered correctly.)")
		case <-done:
		}
	}()
	// Note that we hard-code "Client_identity" here before creating the DTLS client,
	// required when performing token exchange
	log.Println(gatewayAddress, psk, clientID)
	dtlsClient := tradfri.NewTradfriClient(gatewayAddress, "Client_identity", psk)

	authToken, err := dtlsClient.AuthExchange(clientID)
	if err != nil {
		log.Fatal(err)
	}
	viper.Set("client_id", clientID)
	viper.Set("gateway_address", gatewayAddress)
	viper.Set("psk", authToken.Token)
	err = viper.WriteConfigAs("ikea.config.json")
	if err != nil {
		log.Fatal(err)
	}
	os.Rename("ikea.config.json", "ikea.config.json.key")
	logrus.Info("Your configuration including the new PSK and clientID has been written to ikea.config.json.key, keep this file safe!")
}

func fail(msg string) {
	logrus.Info(msg)
	os.Exit(1)
}
