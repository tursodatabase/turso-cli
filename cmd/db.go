package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/briandowns/spinner"
)

type DbCmd struct {
	Create CreateCmd `cmd:"" help:"Create a database."`
}

type CreateCmd struct {
}

func (cmd *CreateCmd) Run(globals *Globals) error {
	accessToken := os.Getenv("IKU_API_TOKEN")
	if accessToken == "" {
		return fmt.Errorf("please set the `IKU_API_TOKEN` environment variable to your access token")
	}
	url := "https://api.chiseledge.com/v1/databases"
	bearer := "Bearer " + accessToken
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", bearer)
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Creating a database... "
	s.Start()
	client := &http.Client{}
	resp, err := client.Do(req)
	s.Stop()
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	m := result.(map[string]interface{})
	ipAddr := m["ipAddr"].(map[string]interface{})["address"].(string)
	pgUrl := fmt.Sprintf("postgresql://%v:5000", ipAddr)
	fmt.Printf("Database created. You can access it at:\n\n%s\n\n", pgUrl)
	fmt.Println("Starting SQL shell...")
	time.Sleep(2 * time.Second)
	pgCmd := exec.Command("psql", pgUrl)
	pgCmd.Stdout = os.Stdout
	pgCmd.Stderr = os.Stderr
	pgCmd.Stdin = os.Stdin
	err = pgCmd.Run()
	if err != nil {
		return err
	}
	return nil
}
