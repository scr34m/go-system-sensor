package main

import (
	"log"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

const (
	mqttBroker = "tcp://192.168.1.21:1883"
	clientID   = "go-system-sensor"
)

var mqttClient MQTT.Client

func setupMQTT() {
	opts := MQTT.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID(clientID)
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		log.Printf("Ismeretlen üzenet: %s => %s", msg.Topic(), msg.Payload())
	})

	mqttClient = MQTT.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT kapcsolódási hiba: %v", token.Error())
	}
}

func main() {
	setupMQTT()

	device := map[string]interface{}{
		"identifiers":  []string{"hp_server"},
		"name":         "Server",
		"manufacturer": "ASUSTeK COMPUTER INC.",
		"model":        "PRIME H410M-K",
	}

	fanConfig(device)
	go fanPublishLoop()

	tempConfig(device)
	go tempPublishLoop()

	select {}
}
