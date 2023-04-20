package reporter

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Messanger struct {
	httpClient   *http.Client
	slackChannel string
	slackClient  *slack.Client
}

const (
	BotName         = "Kafka opbects operator"
	MsgColorOK      = "#00CC00"
	MsgColorWarning = "#F5EC1E"
	MsgColorError   = "#EE0000"
	BotLogo         = "https://upload.wikimedia.org/wikipedia/commons/3/39/Kubernetes_logo_without_workmark.svg"
)

func New(token string, options ...Options) (*Messanger, error) {
	mess := &Messanger{
		httpClient: &http.Client{},
	}
	var err error
	for _, option := range options {
		err = option(mess)
		if err != nil {
			return nil, fmt.Errorf("can't make new Messanger object: %w", err)
		}
	}
	// Set default http client timeout if not specified
	if mess.httpClient.Timeout == 0 {
		mess.httpClient.Timeout = 10 * time.Second
	}
	if len(token) != 0 {
		mess.slackClient = slack.New(token, slack.OptionHTTPClient(mess.httpClient))
	}
	return mess, nil
}

func (r *Messanger) Message(
	messages []string,
	title, color string) {
	// system logger
	reqLogger := log.FromContext(context.Background()).WithValues("reporter")
	// Send message to slack
	attachment := slack.Attachment{
		Color:      color,
		Title:      title,
		Text:       strings.Join(messages, "\n"),
		AuthorName: BotName,
		Footer:     "Migrations service notifications",
		FooterIcon: BotLogo,
	}
	_, _, err := r.slackClient.PostMessage(r.slackChannel,
		slack.MsgOptionUsername(BotName),
		slack.MsgOptionAttachments(attachment))
	if err != nil {
		reqLogger.Error(err, "can't send errors message to Slack")
	}
}
