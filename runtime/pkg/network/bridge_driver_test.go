package network

import (
	"testing"

	"github.com/DeJeune/sudocker/runtime/config"
)

var testName = "testbridge"

func TestBridgeCreate(t *testing.T) {
	d := BridgeNetworkDriver{}
	n, err := d.Create("192.168.0.1/24", testName)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("create network :%v", n)
}

func TestBridgeConnect(t *testing.T) {
	ep := &config.Endpoint{
		Uuid: "testcontainer",
	}

	n := config.Network{
		Name: testName,
	}

	d := BridgeNetworkDriver{}
	err := d.Connect(n.Name, ep)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBridgeDisconnect(t *testing.T) {
	ep := config.Endpoint{
		Uuid: "testcontainer",
	}

	d := BridgeNetworkDriver{}
	err := d.Disconnect(ep.Uuid)
	if err != nil {
		t.Fatal(err)
	}
}
