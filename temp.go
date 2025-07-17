package main

import (
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

func tempConfig(system_name string, device map[string]interface{}, dirs []string, prefixes []string, configs []ConfigTempEntity) {

	tempNodes = make(map[string]*tempNode)
	for _, dir := range dirs {
		tempBuild(dir, prefixes, tempNodes)
	}

	for _, config := range configs {
		label := config.Name
		tempNodes[label] = &tempNode{
			path:      config.Path,
			unique_id: strings.ReplaceAll(strings.ToLower(label), " ", "_"),
		}
	}

	for label, temp := range tempNodes {
		tempNodes[label].state_topic = fmt.Sprintf("sensors/system/%s/%s", system_name, temp.unique_id)

		config := map[string]interface{}{
			"name":                label,
			"unique_id":           fmt.Sprintf("%s_%s", system_name, temp.unique_id),
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

		token := mqttClient.Publish(fmt.Sprintf("homeassistant/sensor/%s/%s/config", system_name, temp.unique_id), 0, true, payload)
		token.Wait()
	}
}

func tempLabelValidate(label string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(label, prefix) {
			return true
		}
	}
	return false
}

func tempBuild(hwPath string, prefixes []string, list map[string]*tempNode) {
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
		if !tempLabelValidate(label, prefixes) {
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
