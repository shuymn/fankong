package main

import (
	"fmt"
	"time"
)

type Config struct {
	Display     string
	Xauthority  string
	Interval    time.Duration
	TargetTemp  int
	MinFanSpeed uint
	MaxFanSpeed uint
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Validate() error {
	if c.Display == "" {
		return fmt.Errorf("display is required")
	}

	if c.Xauthority == "" {
		return fmt.Errorf("xauthority is required")
	}

	if c.Interval <= 0 {
		return fmt.Errorf("incorrect value to interval")
	}

	if c.TargetTemp <= 0 {
		return fmt.Errorf("incorrect value to target temp")
	}

	if c.MinFanSpeed <= 0 || c.MinFanSpeed > 100 {
		return fmt.Errorf("incorrect value to min fan speed")
	}

	if c.MaxFanSpeed <= 0 || c.MaxFanSpeed > 100 {
		return fmt.Errorf("incorrect value to max fan speed")
	}

	if c.MinFanSpeed > c.MaxFanSpeed {
		return fmt.Errorf("max fan speed must be greater than min fan speed")
	}

	return nil
}
