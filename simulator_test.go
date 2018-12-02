package main

import (
	"github.com/fsouza/go-dockerclient"
	"net"
	"testing"
	"time"
)

type fakeApi struct{}

func (fakeApi) GetDockerInfo() (*docker.DockerInfo, error) {
	panic("implement me")
}

func (fakeApi) GetClientIP(string) (*string, error) {
	panic("implement me")
}

func (fakeApi) GetClientEnode(string) (*string, error) {
	panic("implement me")
}

func (fakeApi) GetClientTypes() ([]string, error) {
	return []string{"fakeclient", "fakeclient2"}, nil
}

func (fakeApi) StartNewNode(map[string]string) (string, net.IP, error) {
	return "123123", net.ParseIP("127.1.1.1s"), nil
}

func (fakeApi) Log(string) error {
	panic("implement me")
}

func (fakeApi) AddResults(success bool, nodeID string, name string, errMsg string, duration time.Duration) error {
	return nil
}

func (fakeApi) KillNode(string) error {
	panic("implement me")
}

func xTestSimulator(t *testing.T) {

	b := BlocktestExecutor{api: fakeApi{}, root: "./"}
	b.walkTests()
	// todo, implemnent me
}
