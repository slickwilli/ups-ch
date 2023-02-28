package models

import "time"

type Reading struct {
	DeviceID    uint32
	DisplayName string
	Watts       float32
	Timestamp   time.Time
}
