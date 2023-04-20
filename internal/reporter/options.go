package reporter

import (
	"fmt"
	"net/http"
	"strings"
)

type Options func(*Messanger) error

// HTTPClient will add http client for Slack messanger
func HTTPClient(hc *http.Client) Options {
	return func(s *Messanger) error {
		if hc == nil {
			return fmt.Errorf("you must provide valid http client")
		}
		s.httpClient = hc
		return nil
	}
}

// SlackChannel will add slack channel name
func SlackChannel(sl string) Options {
	return func(s *Messanger) error {
		s.slackChannel = strings.Trim(sl, " \t")
		if len(s.slackChannel) == 0 {
			return fmt.Errorf("slack channel must be provided")
		}
		return nil
	}
}
