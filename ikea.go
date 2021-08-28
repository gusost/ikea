package ikea

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

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
	deviceList, err := client.ListDevices()

	if err != nil {
		return "", err
	}

	list := ""
	for _, device := range deviceList {
		list += fmt.Sprintf("%v - %v - %v\n", device.Name, device.Type, device.DeviceId)
	}

	fmt.Println(list)

	return list, err
}

func TurnOutletOn(deviceId int) error {
	return SetOutletPowerState(deviceId, 1)
}
func TurnOutletOff(deviceId int) error {
	return SetOutletPowerState(deviceId, 0)
}

func SetOutletPowerState(deviceId, powerState int) error {
	device, err := client.GetDevice(deviceId)

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
			fmt.Println("Outlet turned on")
		} else {
			fmt.Println("Outlet turned off")
		}
	}
	return nil
}

func IsOutletOn(deviceId int) bool {
	device, err := client.GetDevice(deviceId)

	if err != nil {
		fmt.Printf("error getting outlet state: %+v\n", err)
		return false
	}
	if device.Type != 3 {
		fmt.Printf("device is not an outlet: %+v\n", err)
		return false
	}

	return device.OutletControl[0].Power == 1
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
