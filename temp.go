package main

import (
	// "encoding/json"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"os"
	"path/filepath"
)

type tempNode struct {
	path        string
	unique_id   string
	value       float64
	state_topic string
}

var tempNodes map[string]*tempNode

func tempRead(path string) float64 {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(b))
	milliCelsius, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return float64(milliCelsius) / 1000.0
}

func tempPublishLoop() {
	for {
		for _, temp := range tempNodes {
			temp.value = tempRead(temp.path)
			token := mqttClient.Publish(temp.state_topic, 0, false, fmt.Sprintf("%.1f", temp.value))
			token.Wait()
		}
		time.Sleep(10 * time.Second)
	}
}

func tempConfig(device map[string]interface{}) {
	tempNodes = make(map[string]*tempNode)
	tempBuild("/sys/class/hwmon/hwmon1/", tempNodes)
	tempBuild("/sys/class/hwmon/hwmon2/", tempNodes)

	for label, temp := range tempNodes {
		tempNodes[label].state_topic = fmt.Sprintf("sensors/system/hp_temp/%s", temp.unique_id)

		config := map[string]interface{}{
			"name":                label,
			"unique_id":           fmt.Sprintf("hp_temp_%s", temp.unique_id),
			"state_topic":         temp.state_topic,
			"unit_of_measurement": "°C",
			"device_class":        "temperature",
			"device":              device,
		}

		payload, err := json.Marshal(config)
		if err != nil {
			log.Fatalf("Json.Marshal error: %v", err)
			continue
		}

		token := mqttClient.Publish(fmt.Sprintf("homeassistant/sensor/hp_temp/%s/config", temp.unique_id), 0, true, payload)
		token.Wait()
	}
}

func tempLabelValidate(label string) bool {
	if !strings.HasPrefix(label, "Package") && !strings.HasPrefix(label, "Core") && !strings.HasPrefix(label, "SYSTIN") && !strings.HasPrefix(label, "CPUTIN") {
		return false
	} else {
		return true
	}
}

func tempBuild(hwPath string, list map[string]*tempNode) {
	err := filepath.Walk(hwPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasPrefix(info.Name(), "temp") {
			return nil
		}

		if !strings.HasSuffix(info.Name(), "_label") {
			return nil
		}

		labelBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		label := strings.TrimSpace(string(labelBytes))
		if !tempLabelValidate(label) {
			return nil
		}

		prefix := strings.TrimSuffix(info.Name(), "_label")

		list[label] = &tempNode{
			path:      filepath.Join(hwPath, prefix+"_input"),
			unique_id: strings.ReplaceAll(strings.ToLower(label), " ", "_"),
		}

		return nil
	})

	if err != nil {
		fmt.Println("An error occured:", err)
	}
}
