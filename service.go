package service

import (
	"context"
	"time"

	"log"

	consul "github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

type ServiceClient interface {
	Register() error
	UnRegister() error
}

type consulClient struct {
	Context context.Context
	Cancel  context.CancelFunc
	ID      string
	Name    string
	Port    int
	client  *consul.Client
}

func NewServiceClient(name string, port int, consulAddr string) (ServiceClient, error) {
	return newConsulClient(name, port, consulAddr)
}

func newConsulClient(name string, port int, consulAddr string) (*consulClient, error) {
	config := consul.DefaultConfig()
	config.Address = consulAddr
	c, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.TODO())
	id := uuid.NewV4().String()

	return &consulClient{
		Context: ctx,
		Cancel:  cancel,
		ID:      id,
		Name:    name,
		Port:    port,
		client:  c}, nil
}

func (c *consulClient) Register() error {
	reg := &consul.AgentServiceRegistration{
		ID:   c.ID,
		Name: c.Name,
		Port: c.Port,
		Check: &consul.AgentServiceCheck{
			TTL:    "15s",
			Status: consul.HealthPassing,
		},
	}

	ticker := time.NewTicker(10 * time.Second)

	regErr := c.client.Agent().ServiceRegister(reg)
	go func() {
		for {
			select {
			case <-c.Context.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				err := c.client.Agent().UpdateTTL("service:"+c.ID, "user_service", consul.HealthPassing)
				if err != nil {
					log.Println(err.Error())
				}
			}
		}
	}()

	return errors.Wrap(regErr, "Cannot register service")
}

func (c *consulClient) UnRegister() error {
	c.Cancel()
	return c.client.Agent().ServiceDeregister(c.ID)
}
