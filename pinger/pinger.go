package pinger

import (
	"strconv"
)

type (
	Pinger interface {
		Ping(host string, times int, delay int) Result
	}

	Latency struct {
		Min, Avg, Max float64
	}

	Result struct {
		Online     bool
		PacketLoss int64
		Latency    Latency
	}
)

/* Utilities */

func giveFloat(value string) float64 {
	result, _ := strconv.ParseFloat(value, 64)
	return result
}

func giveInt64(value string) int64 {
	result, _ := strconv.ParseInt(value, 10, 64)
	return result
}
