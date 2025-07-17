package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

var mqttClient MQTT.Client

func setupMQTT(mqttBroker *string) {
	id := uuid.New()
	opts := MQTT.NewClientOptions()
	opts.AddBroker(*mqttBroker)
	opts.SetClientID(fmt.Sprintf("go-system-sensor-%s", id.String()))
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		log.Printf("Unknown message: %s => %s", msg.Topic(), msg.Payload())
	})

	mqttClient = MQTT.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect error: %v", token.Error())
	}
}

func main() {

	mqttBroker := flag.String("mqtt", "tcp://192.168.1.21:1883", "MQTT server address")
	flag.Parse()

	setupMQTT(mqttBroker)

	f := "go-system-sensor.toml"
	if _, err := os.Stat(f); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var c Config
	_, err := toml.DecodeFile(f, &c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	device := map[string]interface{}{
		"identifiers":  c.Device.Identifiers,
		"name":         c.Device.Name,
		"manufacturer": c.Device.Manufacturer,
		"model":        c.Device.Model,
	}

	fanConfig(c.Fan.Name, device, c.Fan.Entities)
	go fanPublishLoop()

	tempConfig(c.Temp.Name, device, c.Temp.Paths, c.Temp.Prefixes, c.Temp.Entities)
	go tempPublishLoop()

	select {}
}
