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

type CreateInstanceLocationError struct {
	err string
}

func (e *CreateInstanceLocationError) Error() string {
	return e.err
}

func (i *InstancesClient) List(db string) ([]Instance, error) {
	r, err := i.client.Get(i.URL(db, ""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances of %s: %s", db, err)
	}
	defer r.Body.Close()

	org := i.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return nil, notMemberErr(org)
	}

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
	url := i.URL(db, "/"+instance)
	r, err := i.client.Delete(url, nil)
	if err != nil {
		return fmt.Errorf("failed to destroy instances %s of %s: %s", instance, db, err)
	}
	defer r.Body.Close()

	org := i.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return notMemberErr(org)
	}

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

func (d *InstancesClient) Create(dbName, location string) (*Instance, error) {
	type Body struct {
		Location string
	}
	body, err := marshal(Body{location})
	if err != nil {
		return nil, fmt.Errorf("could not serialize request body: %w", err)
	}

	url := d.URL(dbName, "")
	res, err := d.client.Post(url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create new instances for %s: %s", dbName, err)
	}
	defer res.Body.Close()

	org := d.client.Org
	if isNotMemberErr(res.StatusCode, org) {
		return nil, notMemberErr(org)
	}

	if res.StatusCode >= http.StatusInternalServerError {
		return nil, &CreateInstanceLocationError{fmt.Sprintf("failed to create new instance: %s", res.Status)}
	}

	if res.StatusCode != http.StatusOK {
		return nil, parseResponseError(res)
	}

	data, err := unmarshal[struct{ Instance Instance }](res)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	return &data.Instance, nil
}

func (i *InstancesClient) Wait(db, instance string) error {
	url := i.URL(db, "/"+instance+"/wait")
	r, err := i.client.Get(url, nil)
	if err != nil {
		return fmt.Errorf("failed to wait for instance %s to of %s be ready: %s", instance, db, err)
	}
	defer r.Body.Close()

	org := i.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return notMemberErr(org)
	}

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

func (d *InstancesClient) URL(database, suffix string) string {
	prefix := "/v1"
	if d.client.Org != "" {
		prefix = "/v1/organizations/" + d.client.Org
	}
	return fmt.Sprintf("%s/databases/%s/instances%s", prefix, database, suffix)
}
