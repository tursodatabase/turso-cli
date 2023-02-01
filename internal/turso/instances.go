package turso

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/chiselstrike/iku-turso-cli/internal/clients"
)

type Instance struct {
	Uuid   string
	Name   string
	Type   string
	Region string
}

type instances struct {
	c *clients.Client
}

var Instances = &instances{Client}

func NewInstances(base *url.URL, token string) *instances {
	return &instances{NewTurso(base, token)}
}

func (i *instances) List(db string) ([]Instance, error) {
	r, err := i.c.Get(fmt.Sprintf("v2/databases/%s/instances", db), nil)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response with status code %d", r.StatusCode)
	}

	type ListResponse struct{ Instances []Instance }
	resp, err := Unmarshall[ListResponse](r)
	if err != nil {
		return nil, err
	}

	return resp.Instances, nil
}

func (i *instances) Delete(db, instance string) error {
	r, err := i.c.Delete(fmt.Sprintf("v2/databases/%s/instances/%s", db, instance), nil)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusBadRequest {
		body, _ := Unmarshall[struct{ Error string }](r)
		return errors.New(body.Error)
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("response with status code %d", r.StatusCode)
	}

	return nil
}
