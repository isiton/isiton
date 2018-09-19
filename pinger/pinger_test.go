package pinger

import (
	"log"
	"testing"
)

func TestSomething(t *testing.T) {
	// test stuff here...
	pdp := NewPinger()
	log.Println("Ping 192.1.1.1")
	result := pdp.Ping("192.1.1.1")
	log.Println("Finished ping")
	if result.Online {
		t.Error("Ping 192.1.1.1 shoulw result in online=false")
	}
	log.Printf("result:\n%#v\n", result)
}
