package main

type PdPinger interface {
	RunPing(host string) PingResult
}
