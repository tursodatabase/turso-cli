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
	start := time.Now()
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
	end := time.Now()
	elapsed := end.Sub(start)
	m := result.(map[string]interface{})
	primaryIpAddr := m["primaryIpAddr"].(string)
	primaryPgUrl := fmt.Sprintf("postgresql://%v:5000", primaryIpAddr)
	replicaIpAddr := m["replicaIpAddr"].(string)
	replicaPgUrl := fmt.Sprintf("postgresql://%v:5000", replicaIpAddr)
	fmt.Printf("Database created in %d seconds.\n\n", int(elapsed.Seconds()))
	fmt.Printf("You can access the database at:\n")
	fmt.Printf("  - %s [primary]\n", primaryPgUrl)
	fmt.Printf("  - %s [replica]\n", replicaPgUrl)
	fmt.Printf("\n")
	fmt.Println("Connecting SQL shell to primary server...\n")
	pgCmd := exec.Command("psql", primaryPgUrl)
	pgCmd.Stdout = os.Stdout
	pgCmd.Stderr = os.Stderr
	pgCmd.Stdin = os.Stdin
	err = pgCmd.Run()
	if err != nil {
		return err
	}
	return nil
}
