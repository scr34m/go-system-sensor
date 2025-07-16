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

func setFanSpeed(path string, percent int64) {
	f := fmt.Sprintf("%d\n", percentToPwm(percent))
	err := ioutil.WriteFile(path, []byte(f), 0644)
	if err != nil {
		log.Printf("Unable to set PWM value: %v", err)
	} else {
		log.Printf("New fan speed PWM value: %d", percent)
		mqttClient.Publish(fanCasePercentageTopic, 0, false, fmt.Sprintf("%d", percent))
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

const (
	fanCpuStateTopic    = "sensors/system/hp_fan/cpu/on/state"
	fanCpuRpmStateTopic = "sensors/system/hp_fan/cpu/speed/rpm"

	fanCaseStateTopic             = "sensors/system/hp_fan/case/on/state"
	fanCasePercentageTopic        = "sensors/system/hp_fan/case/speed/percentage_state"
	fanCasePercentageCommandTopic = "sensors/system/hp_fan/case/speed/percentage"
	fanCaseRpmStateTopic          = "sensors/system/hp_fan/case/speed/rpm"
)

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
	log.Printf("New percent value: %d", percent)
	setFanSpeed("/sys/class/hwmon/hwmon2/pwm1", percent)
}

func fanPublishLoop() {
	mqttClient.Subscribe(fanCasePercentageCommandTopic, 0, onFanSet)

	token := mqttClient.Publish(fanCpuStateTopic, 0, false, "ON")
	token.Wait()

	token = mqttClient.Publish(fanCaseStateTopic, 0, false, "ON")
	token.Wait()

	for {
		// pwm := fanReadPWM("/sys/class/hwmon/hwmon2/pwm2")
		// mqttClient.Publish(fanCpuStateTopic, 0, false, fmt.Sprintf("%d", pwmToPercent(pwm)))

		rpm := fanReadPWM("/sys/class/hwmon/hwmon2/fan2_input")
		token := mqttClient.Publish(fanCpuRpmStateTopic, 0, false, fmt.Sprintf("%d", rpm))
		token.Wait()

		pwm := fanReadPWM("/sys/class/hwmon/hwmon2/pwm1")
		token = mqttClient.Publish(fanCasePercentageTopic, 0, false, fmt.Sprintf("%d", pwmToPercent(pwm)))
		token.Wait()

		rpm = fanReadPWM("/sys/class/hwmon/hwmon2/fan1_input")
		token = mqttClient.Publish(fanCaseRpmStateTopic, 0, false, fmt.Sprintf("%d", rpm))
		token.Wait()

		time.Sleep(10 * time.Second)
	}
}

func fanConfig(device map[string]interface{}) {
	// CPU FAN
	fanConfig1 := map[string]interface{}{
		"name":            "CPU FAN",
		"unique_id":       "hp_fan_cpu",
		"command_topic":   "dummy",
		"state_topic":     fanCpuStateTopic,
		"payload_on":      "ON",
		"payload_off":     "OFF",
		"speed_range_min": 1,
		"speed_range_max": 100,
		"device":          device,
	}

	payload, err := json.Marshal(fanConfig1)
	if err != nil {
		log.Fatalf("Json.Marshal error: %v", err)
	}

	token := mqttClient.Publish("homeassistant/fan/hp_fan/cpu/config", 0, true, payload)
	token.Wait()

	fanConfig12 := map[string]interface{}{
		"name":                "CPU FAN RPM",
		"unique_id":           "hp_fan_rpm_cpu",
		"unit_of_measurement": "rpm",
		"device_class":        nil,
		"state_topic":         fanCpuRpmStateTopic,
		"device":              device,
	}

	payload, err = json.Marshal(fanConfig12)
	if err != nil {
		log.Fatalf("Json.Marshal error: %v", err)
	}

	token = mqttClient.Publish("homeassistant/sensor/hp_fan/cpu_rpm/config", 0, true, payload)
	token.Wait()

	// Case FAN
	fanConfig2 := map[string]interface{}{
		"name":                     "Case FAN",
		"unique_id":                "hp_fan_case",
		"command_topic":            "dummy",
		"percentage_command_topic": fanCasePercentageCommandTopic,
		"percentage_state_topic":   fanCasePercentageTopic,
		"state_topic":              fanCaseStateTopic,
		"payload_on":               "ON",
		"payload_off":              "OFF",
		"speed_range_min":          1,
		"speed_range_max":          100,
		"device":                   device,
	}

	payload, err = json.Marshal(fanConfig2)
	if err != nil {
		log.Fatalf("Json.Marshal error: %v", err)
	}

	token = mqttClient.Publish("homeassistant/fan/hp_fan/case/config", 0, true, payload)
	token.Wait()

	fanConfig22 := map[string]interface{}{
		"name":                "Case FAN RPM",
		"unique_id":           "hp_fan_rpm_case",
		"unit_of_measurement": "rpm",
		"device_class":        nil,
		"state_topic":         fanCaseRpmStateTopic,
		"device":              device,
	}

	payload, err = json.Marshal(fanConfig22)
	if err != nil {
		log.Fatalf("Json.Marshal error: %v", err)
	}

	token = mqttClient.Publish("homeassistant/sensor/hp_fan/case_rpm/config", 0, true, payload)
	token.Wait()
}
