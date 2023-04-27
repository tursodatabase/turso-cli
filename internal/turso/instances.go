package turso

import (
	"errors"
	"fmt"
	"net/http"
)

type Instance struct {
	Uuid     string
	Name     string
	Type     string
	Region   string
	Hostname string
}

type InstancesClient client

func (i *InstancesClient) List(db string) ([]Instance, error) {
	r, err := i.client.Get(i.URL(db, ""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances of %s: %s", db, err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response with status code %d", r.StatusCode)
	}

	type ListResponse struct{ Instances []Instance }
	resp, err := unmarshal[ListResponse](r)
	if err != nil {
		return nil, err
	}

	return resp.Instances, nil
}

func (i *InstancesClient) Delete(db, instance string) error {
	url := i.URL(db, "/instances/"+instance)
	r, err := i.client.Delete(url, nil)
	if err != nil {
		return fmt.Errorf("failed to destroy instances %s of %s: %s", instance, db, err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusBadRequest {
		body, _ := unmarshal[struct{ Error string }](r)
		return errors.New(body.Error)
	}

	if r.StatusCode == http.StatusNotFound {
		body, _ := unmarshal[struct{ Error string }](r)
		return errors.New(body.Error)
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("response with status code %d", r.StatusCode)
	}

	return nil
}

func (d *InstancesClient) Create(dbName, instanceName, password, region, image string) (*Instance, error) {
	type Body struct {
		Password, Region, Image string
		InstanceName            string `json:"instance_name,omitempty"`
	}
	body, err := marshal(Body{password, region, image, instanceName})
	if err != nil {
		return nil, fmt.Errorf("could not serialize request body: %w", err)
	}

	url := d.URL(dbName, "")
	res, err := d.client.Post(url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create new instances for %s: %s", dbName, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, parseResponseError(res)
	}

	data, err := unmarshal[struct{ Instance Instance }](res)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	return &data.Instance, nil
}

func (d *InstancesClient) URL(database, suffix string) string {
	prefix := "/v1"
	if d.client.org != "" {
		prefix = "/v1/organizations/" + d.client.org
	}
	return fmt.Sprintf("%s/databases/%s/instances", prefix, database)
}
