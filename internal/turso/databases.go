package turso

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type DatabasesService service

type ListDatabasesResponse struct {
	Databases []Database `json:"databases"`
}

type Database struct {
	Name     string
	Type     string
	Region   string
	Hostname string
}

func (s *DatabasesService) List() ([]Database, error) {
	url := fmt.Sprintf("/v1/databases")
	req, err := s.client.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	response := make(map[string]interface{})
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, err
	}
	databases := response["databases"].([]interface{})
	result := make([]Database, 0)
	for _, db := range databases {
		d := db.(map[string]interface{})
		result = append(result, Database{
			Name:     d["Name"].(string),
			Type:     d["Type"].(string),
			Region:   d["Region"].(string),
			Hostname: d["Hostname"].(string),
		})
	}
	return result, nil
}
