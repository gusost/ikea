package ikea

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

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

func firstIntitGateway(keyPath string) error {
	//err := fmt.Errorf("initial key exchange with gateway not implemented yet. It needs to be done to get a valid key associated with your client ID (key erroneously called pre shared in the TradfriConfig struct)")

	ikeaKeyfile, err := os.ReadFile(keyPath)
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

var presentedConfigKeyPath string = ""
var presentedKeyPath string = ""

func IntitGateway(configKeyPath, keyPath string) error {
	presentedConfigKeyPath = configKeyPath
	presentedKeyPath = keyPath
	level, _ := logrus.ParseLevel("error")
	//level, _ := logrus.ParseLevel("info")
	//level, _ := logrus.ParseLevel("trace")
	logrus.SetLevel(level)

	ikeaKeyfile, err := os.ReadFile(configKeyPath)
	if err != nil {
		err = firstIntitGateway(keyPath)
		if err != nil {
			return fmt.Errorf("cannot read key file 'ikea.config.json.key': %s", err)
		}

		ikeaKeyfile, err = os.ReadFile(configKeyPath)

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

	var propertyMatrix [][]string = [][]string{
		{"ID", "👁  Seen", "State", "🔋", "Name"},
	}

	for _, device := range deviceList {
		// Hide dead devices
		if !includeDead && device.Alive == 0 {
			continue
		}

		d := MyDevice(device)

		var propertyStrings []string
		propertyStrings = append(propertyStrings, fmt.Sprintf("%v", d.DeviceId))
		//propertyStrings = append(propertyStrings, fmt.Sprintf(" - %v", d.PrintType()))
		propertyStrings = append(propertyStrings, fmt.Sprintf("%v", d.PrintLastSeen(0)))
		propertyStrings = append(propertyStrings, fmt.Sprintf("%v", d.PrintState(0)))
		propertyStrings = append(propertyStrings, fmt.Sprintf("%v", d.PrintBattery(0)))
		propertyStrings = append(propertyStrings, fmt.Sprintf("%v", d.Name))

		// Alive
		/* if device.Alive == 1 {
		list += " - Alive"
		} else {
			list += " - Dead "
		} */

		propertyMatrix = append(propertyMatrix, propertyStrings)
	}

	// Transpose a slice of string rows into a slice of string columns
	columnStringSlice := transpose(propertyMatrix)
	columnLengthSlice := []int{}

	// Find the visibly longest string in each column. Emojis take up 2 spaces. What they count as is a mystery.
	for _, col := range columnStringSlice {
		columnLengthSlice = append(columnLengthSlice, longestVisibleString(col))
	}

	deviceTable := ""
	// Print header
	// Top line
	deviceTable += frameBuilder(propertyMatrix, columnLengthSlice, '╔', '╦', '╗')
	// Title line
	deviceTable += dataLineBuilder(propertyMatrix[0], columnLengthSlice)
	// Title bottom line
	deviceTable += frameBuilder(propertyMatrix, columnLengthSlice, '╠', '╬', '╣')

	// Data rows
	for i, row := range propertyMatrix {
		// Skip header
		if i == 0 {
			continue
		}
		deviceTable += dataLineBuilder(row, columnLengthSlice)
	}

	// Bottom line
	deviceTable += frameBuilder(propertyMatrix, columnLengthSlice, '╚', '╩', '╝')

	println(deviceTable)

	return deviceTable, err
}

func dataLineBuilder(row []string, columnLengthSlice []int) string {
	s := ""
	for i, title := range row {
		offset := 1
		// A table with offset values for the emojis would be nice
		if strings.Contains(title, "📏") || strings.Contains(title, "🔋") {
			// counts as 3 only takes up 2
			offset -= 1
		}
		if strings.Contains(title, "0️⃣") || strings.Contains(title, "1️⃣") {
			// counts as 0 actually takes up 2
			offset += 2
		}
		s += fmt.Sprintf("║ %-*v", columnLengthSlice[i]+offset, title)
	}
	s += "║\n"
	return s
}

func frameBuilder(propertyStringsSlice [][]string, columnLengthSlice []int, start rune, spacing rune, end rune) string {
	s := ""
	for i := range propertyStringsSlice[0] {
		if i == 0 {
			s += fmt.Sprintf("%v%v", string(start), strings.Repeat("═", columnLengthSlice[0]+2))
		} else if i != len(propertyStringsSlice[0])-1 {
			s += fmt.Sprintf("%v%v", string(spacing), strings.Repeat("═", columnLengthSlice[i]+2))
		} else {
			s += fmt.Sprintf("%v%v%v\n", string(spacing), strings.Repeat("═", columnLengthSlice[i]+2), string(end))
		}
	}
	return s
}

// Transpoeses a sting matrix
func transpose(slice [][]string) [][]string {
	xl := len(slice[0])
	yl := len(slice)
	result := make([][]string, xl)
	for i := range result {
		result[i] = make([]string, yl)
	}
	for i := 0; i < xl; i++ {
		for j := 0; j < yl; j++ {
			result[i][j] = slice[j][i]
		}
	}
	return result
}

// Find the length of the longest string in a slice of strings
func longestVisibleString(stringSlice []string) int {
	//fmt.Printf("For %v:\n", stringSlice)
	longest := 0
	for _, str := range stringSlice {
		visibleLength := utf8.RuneCountInString(str)
		// Their VISIBLE length on screen is 2. What lenght count as in a string varies. Messy
		if strings.Contains(str, "🔋") || strings.Contains(str, "📏") || strings.Contains(str, "0️⃣") || strings.Contains(str, "1️⃣") {
			visibleLength++
		}
		if visibleLength > longest {
			longest = visibleLength
			//fmt.Printf("\"%v\" is the longest string with %v runes\n", v, longest)
		}
	}
	return longest
}

func ListDevicesBattery() (string, error) {
	deviceList, err := client.ListDevices()
	if err != nil {
		return "", err
	}
	// Sort by id
	sort.Slice(deviceList, func(i, j int) bool { return deviceList[i].Name < deviceList[j].Name })

	var propertyMatrix [][]string = [][]string{
		{"Name", "🔋"},
	}

	for _, device := range deviceList {
		// Hide dead devices
		if device.Alive == 0 || device.Metadata.Battery == 0 {
			continue
		}
		d := MyDevice(device)

		var propertyStrings []string
		propertyStrings = append(propertyStrings, fmt.Sprintf("%v", d.Name))
		propertyStrings = append(propertyStrings, fmt.Sprintf("%v", d.PrintBattery(0)))

		propertyMatrix = append(propertyMatrix, propertyStrings)
	}

	// Transpose a slice of string rows into a slice of string columns
	columnStringSlice := transpose(propertyMatrix)
	columnLengthSlice := []int{}

	// Find the visibly longest string in each column. Emojis take up 2 spaces. What they count as is a mystery.
	for _, col := range columnStringSlice {
		columnLengthSlice = append(columnLengthSlice, longestVisibleString(col))
	}

	deviceTable := ""
	// Print header
	// Top line
	deviceTable += frameBuilder(propertyMatrix, columnLengthSlice, '╔', '╦', '╗')
	// Title line
	deviceTable += dataLineBuilder(propertyMatrix[0], columnLengthSlice)
	// Title bottom line
	deviceTable += frameBuilder(propertyMatrix, columnLengthSlice, '╠', '╬', '╣')

	// Data rows
	for i, row := range propertyMatrix {
		// Skip header
		if i == 0 {
			continue
		}
		deviceTable += dataLineBuilder(row, columnLengthSlice)
	}

	// Bottom line
	deviceTable += frameBuilder(propertyMatrix, columnLengthSlice, '╚', '╩', '╝')

	println(deviceTable)

	return deviceTable, err
}

type MyDevice model.Device

func (device *MyDevice) PrintBattery(length int) string {
	if device.Metadata.Battery == 0 {
		return fmt.Sprintf("%-*v", length, " ")
	}
	batteryString := fmt.Sprintf("%v%%", device.Metadata.Battery)
	return fmt.Sprintf("🔋%-*v", length-2, batteryString) // Left align, padding 3
}

func (device *MyDevice) PrintLastSeen(length int) string {
	// Time since seen
	since := time.Since(time.Unix(int64(device.LastSeen), 0))

	// This is a hack. Some devices (only remotes?) seem to not report a proper UNIX epoch but instead the number of seconds since last seen.
	if since > 365*24*time.Hour && device.LastSeen < 1e7 { // 1e7 is ~116 days
		since = time.Since(time.Unix(time.Now().Unix()-int64(device.LastSeen), 0))
	}
	s := ""
	d := math.Round(since.Hours() / 24)
	df := math.Floor(since.Hours() / 24)
	h := math.Round(since.Hours())
	hf := math.Floor(since.Hours())
	m := math.Round(since.Minutes() - hf*60)
	if since > 365*24*time.Hour {
		s += fmt.Sprintf("%v", time.Unix(int64(device.CreatedAt), 0).Format("2006-01-02"))
	} else if since > 7*24*time.Hour {
		s += fmt.Sprintf("%v days", d)
	} else if since > 24*time.Hour {
		s += fmt.Sprintf("%v days %v hours", df, math.Round(h-df*24))
	} else if since > 10*time.Hour {
		s += fmt.Sprintf("%v hours", h)
	} else {
		s += fmt.Sprintf("%vh %02vm", hf, m)
	}
	return fmt.Sprintf("%-*v", length, s) // Left align, padding
}

func (device *MyDevice) PrintType() string {
	switch device.Type {
	case 0: // Remote
		return "Remote"
	case 3: // Outlet
		return "Outlet"
	case 4: // Motion
		return "Motion"
	case 6: // Repeater
		return "Repeat"
	case 7: // Blind
		return "Blind "
	case 1: // Unknown
	case 2:
	case 5:
	default:
	}
	return "Unknow"
}

func (device *MyDevice) PrintState(length int) string {
	s := ""
	switch device.Type {
	case 0: // Remote
	case 3: // Outlet
		length += 2
		if device.OutletControl[0].Power == 0 {
			s += "0️⃣ "
		} else {
			s += "1️⃣ "
		}
	case 4: // Motion
	case 6: // Repeater
	case 7: // Blind
		length -= 1
		s += fmt.Sprintf("📏%v%%", int(device.BlindControl[0].Position))
	case 1: // Unknown
	case 2:
	case 5:
	default:
	}
	return fmt.Sprintf("%-*v", length, s) // Left align
}

func GetDevice(deviceId int) (model.Device, error) {

	device, err := client.GetDevice(deviceId)

	if err != nil {
		fmt.Printf("error getting device: %+v\n", err)
		// if err starts with 'invalid character' return the error. It's probably a non existing device.
		if strings.HasPrefix(err.Error(), "invalid character") {
			return model.Device{}, err
		}
		// attempt a re-connect.
		fmt.Println("Got an error, attempting to reconnect")
		IntitGateway(presentedConfigKeyPath, presentedKeyPath)

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
		IntitGateway(presentedConfigKeyPath, presentedKeyPath)

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
