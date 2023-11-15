package reporter

import "github.com/slack-go/slack"

//go:generate mockgen -source=slack.go -destination=./mock_slack/mock_slack.go -package=mock_slack
type Slack interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}
