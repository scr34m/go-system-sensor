package main

type Config struct {
	Device ConfigDevice
	Temp   ConfigTemp
	Fan    ConfigFan
}

type ConfigDevice struct {
	Identifiers  []string
	Name         string
	Manufacturer string
	Model        string
}

type ConfigTemp struct {
	Name     string
	Paths    []string
	Prefixes []string
}

type ConfigFan struct {
	Name     string
	Entities []ConfigFanEntity
}

type ConfigFanEntity struct {
	Name    string
	PathPWM string
	PathRPM string
}
