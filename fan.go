package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type fanNode struct {
	name      string
	unique_id string
	kind      string

	state_topic              string
	percentage_command_topic string
	percentage_state_topic   string

	pathPWM string
	pathRPM string
}

var fanNodes []*fanNode

func setFanSpeed(node *fanNode, percent int64) {
	f := fmt.Sprintf("%d\n", percentToPwm(percent))
	err := ioutil.WriteFile(node.pathPWM, []byte(f), 0644)
	if err != nil {
		log.Printf("Unable to set PWM value: %v", err)
	} else {
		log.Printf("New fan speed PWM value: %d", percent)
		mqttClient.Publish(node.percentage_state_topic, 0, false, fmt.Sprintf("%d", percent))
	}
}

func fanReadPWM(path string) int64 {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("Unable to read temperature: %v", err)
		return -1
	}
	s := strings.TrimSpace(string(data))
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		log.Printf("Nem sikerült floattá konvertálni: %v", err)
		return -1
	}
	return val
}

func pwmToPercent(val int64) int64 {
	return val * 100 / 255
}

func percentToPwm(val int64) int64 {
	return val * 255 / 100
}

func onFanSet(client MQTT.Client, msg MQTT.Message) {
	s := strings.TrimSpace(string(msg.Payload()))
	percent, err := strconv.ParseInt(s, 10, 64)
	if err != nil || percent < 0 || percent > 100 {
		log.Printf("Not valid percent value: %s", s)
		return
	}

	for _, node := range fanNodes {
		if node.percentage_command_topic == msg.Topic() {
			log.Printf("New percent value: %d", percent)
			setFanSpeed(node, percent)
			break
		}
	}
}

func fanPublishLoop() {
	for _, node := range fanNodes {
		if node.percentage_command_topic != "" {
			mqttClient.Subscribe(node.percentage_command_topic, 0, onFanSet)
		}
	}

	for {
		for _, node := range fanNodes {
			if node.kind == "fan" {
				token := mqttClient.Publish(node.state_topic, 0, false, "ON")
				token.Wait()
			}

			if node.pathPWM != "" {
				value := fanReadPWM(node.pathPWM)
				token := mqttClient.Publish(node.percentage_state_topic, 0, false, fmt.Sprintf("%d", pwmToPercent(value)))
				token.Wait()
			}

			if node.pathRPM != "" {
				value := fanReadPWM(node.pathRPM)
				token := mqttClient.Publish(node.state_topic, 0, false, fmt.Sprintf("%d", value))
				token.Wait()
			}
		}

		time.Sleep(10 * time.Second)
	}
}

func fanConfig(system_name string, device map[string]interface{}, configs []ConfigFanEntity) {
	for _, configNode := range configs {

		unique_id := strings.ReplaceAll(strings.ToLower(configNode.Name), " ", "_")

		node1 := fanNode{
			name:        configNode.Name,
			unique_id:   unique_id,
			kind:        "fan",
			state_topic: fmt.Sprintf("sensors/system/%s/%s/on/state", system_name, unique_id),
			pathPWM:     configNode.PathPWM,
		}
		if configNode.PathPWM != "" {
			node1.percentage_command_topic = fmt.Sprintf("sensors/system/%s/%s/speed/percentage", system_name, unique_id)
			node1.percentage_state_topic = fmt.Sprintf("sensors/system/%s/%s/speed/percentage_state", system_name, unique_id)
		}
		fanNodes = append(fanNodes, &node1)

		node2 := fanNode{
			name:        fmt.Sprintf("%s RPM", configNode.Name),
			unique_id:   unique_id,
			kind:        "rpm",
			state_topic: fmt.Sprintf("sensors/system/%s/%s/speed/rpm", system_name, unique_id),
			pathRPM:     configNode.PathRPM,
		}
		fanNodes = append(fanNodes, &node2)
	}

	for _, node := range fanNodes {
		config := map[string]interface{}{
			"name":        node.name,
			"state_topic": node.state_topic,
			"device":      device,
		}
		var config_topic string

		switch node.kind {
		case "fan":
			config_topic = fmt.Sprintf("homeassistant/fan/%s/%s/config", system_name, node.unique_id)

			config["unique_id"] = fmt.Sprintf("%s_%s", system_name, node.unique_id)
			config["command_topic"] = "dummy"
			if node.percentage_command_topic != "" {
				config["percentage_command_topic"] = node.percentage_command_topic
			}
			if node.percentage_state_topic != "" {
				config["percentage_state_topic"] = node.percentage_state_topic
			}
			config["payload_on"] = "ON"
			config["payload_off"] = "OFF"
			config["speed_range_min"] = 1
			config["speed_range_max"] = 100

		case "rpm":
			config["unique_id"] = fmt.Sprintf("%s_%s_rpm", system_name, node.unique_id)
			config["unit_of_measurement"] = "rpm"

			config_topic = fmt.Sprintf("homeassistant/sensor/%s/%s/config", system_name, node.unique_id)
		default:
			continue
		}

		payload, err := json.Marshal(config)
		if err != nil {
			log.Fatalf("Json.Marshal error: %v", err)
		}

		token := mqttClient.Publish(config_topic, 0, true, payload)
		token.Wait()
	}
}
