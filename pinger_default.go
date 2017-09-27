// +build !windows

package main

import (
	"os/exec"
	"strings"
	"time"

	"github.com/paulstuart/ping"
)

type LinuxPdPinger struct {
}

func NewPinger() PdPinger {
	return &LinuxPdPinger{}
}

func (p *LinuxPdPinger) RunPing(hostname string) PingResult {
	result := PingResult{}
	result.Latency.Min = 1000.0

	var latencySum float64

	pings := 4
	errors := 0
	for i := 0; i < pings; i++ {
		now := time.Now()
		err := ping.Pinger(hostname, 1)
		end := time.Now()

		latencyValue := end.Sub(now).Seconds() * 1000.0
		latencySum += latencyValue
		if latencyValue > result.Latency.Max {
			result.Latency.Max = latencyValue
		}
		if latencyValue < result.Latency.Min {
			result.Latency.Min = latencyValue
		}
		if err != nil {
			errors++
			continue
		}
		time.Sleep(time.Second)
	}
	result.PacketLoss = int64(errors) * 25
	result.Latency.Avg = latencySum / 4.0
	result.Online = true
	if result.PacketLoss == 100 {
		result.Online = false
	}
	return result
}

type LinuxFallbackPdPinger struct {
}

func NewLinuxFallbackPdPinger() *LinuxFallbackPdPinger {
	return &LinuxFallbackPdPinger{}
}

func (p *LinuxFallbackPdPinger) RunPing(hostname string) PingResult {
	cmd := exec.Command("/bin/ping", hostname, "-c", "4")
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	result := PingResult{}
	result.Online = !strings.Contains(outputStr, "100% packet loss")

	if result.Online {
		outputLines := strings.Split(outputStr, "\n")
		for _, line := range outputLines {
			if strings.Contains(line, "packet loss") {
				markers := strings.Split(line, ", ")
				markers = strings.Split(markers[2], "%")
				result.PacketLoss = giveInt64(markers[0])
			}
			if strings.Contains(line, "round-trip") {
				markers := strings.Split(line, " = ")
				markers = strings.Split(markers[1], "/")
				result.Latency = Latency{
					giveFloat(markers[0]),
					giveFloat(markers[1]),
					giveFloat(markers[2]),
				}
				break
			}
		}
	}
	return result
}
