package turso

import (
	"fmt"
	"net/http"
)

type FeedbackClient client

func (d *FeedbackClient) Submit(summary, feedback string) error {
	body := struct{ Summary, Feedback string }{summary, feedback}
	reader, err := marshal(body)
	if err != nil {
		return fmt.Errorf("could not marshal feedback: %w", err)
	}

	r, err := d.client.Post("/v1/feedback", reader)
	if err != nil {
		return fmt.Errorf("failed to post feedback: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to post feedback: %w", parseResponseError(r))
	}

	return nil
}
