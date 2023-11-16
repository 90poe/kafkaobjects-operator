package reporter

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Messenger struct {
	tickInterval time.Duration
	httpClient   *http.Client
	slackChannel string
	slackClient  Slack
	slackChan    chan Message
}

const (
	FlushInterval   = 30 * time.Second
	BotName         = "Kafka objects operator"
	MsgColorOK      = "#00CC00"
	MsgColorWarning = "#F5EC1E"
	MsgColorError   = "#EE0000"
	BotLogo         = "https://90poe-tools-infrastructure.s3.eu-west-1.amazonaws.com/images/k8s-logo.png"
)

func New(token string, options ...Options) (*Messenger, error) {
	mess := &Messenger{
		tickInterval: FlushInterval,
		httpClient:   &http.Client{},
		slackChan:    make(chan Message),
	}
	var err error
	for _, option := range options {
		err = option(mess)
		if err != nil {
			return nil, fmt.Errorf("can't make new Messenger object: %w", err)
		}
	}
	// Set default http client timeout if not specified
	if mess.httpClient.Timeout == 0 {
		mess.httpClient.Timeout = 10 * time.Second
	}
	// We are checking if we have slack client from mocks
	inter := reflect.TypeOf((*Slack)(nil)).Elem()
	if mess.slackClient == nil || !reflect.TypeOf(mess.slackClient).Implements(inter) {
		// we don't have mock client - create a real one
		if len(token) != 0 {
			mess.slackClient = slack.New(token, slack.OptionHTTPClient(mess.httpClient))
		}
	}
	// Run messenger
	go mess.run()
	return mess, nil
}

// Send will send message to slack
func (m *Messenger) Send(msg string, msgType MessageType) {
	m.slackChan <- Message{
		time:    time.Now(),
		message: msg,
		msgType: msgType,
	}
}

// run will send all messages from channel to slack
// We will close if close channel is closed
// We also flush every FlushInterval seconds
func (m *Messenger) run() {
	messages := []Message{}
	ticker := time.NewTicker(m.tickInterval)
	defer func() {
		close(m.slackChan)
		m.flush(messages)
	}()
	for {
		select {
		case msg := <-m.slackChan:
			messages = append(messages, msg)
			if len(messages) >= 10 {
				m.flush(messages)
				messages = []Message{}
			}
		case <-ticker.C:
			m.flush(messages)
			messages = []Message{}
		}
	}
}

// flush will prepare and sort all messages to be sent to slack
func (m *Messenger) flush(messages []Message) {
	if len(messages) == 0 {
		return
	}
	// divide messages by type
	okMessages := []string{}
	warningMessages := []string{}
	errorMessages := []string{}
	for _, msg := range messages {
		switch msg.MsgType() {
		case OKMessage:
			okMessages = append(okMessages, msg.String())
		case WarnMessage:
			warningMessages = append(warningMessages, msg.String())
		default:
			errorMessages = append(errorMessages, msg.String())
		}
	}
	// send messages to slack
	if len(okMessages) != 0 {
		m.send(strings.Join(okMessages, "\n"), "OK", MsgColorOK)
	}
	if len(warningMessages) != 0 {
		m.send(strings.Join(warningMessages, "\n"), "Warning", MsgColorWarning)
	}
	if len(errorMessages) != 0 {
		m.send(strings.Join(errorMessages, "\n"), "OK", MsgColorError)
	}
}

// send will send message to slack or to log
func (m *Messenger) send(msg, title, color string) {
	// system logger
	reqLogger := log.FromContext(context.Background()).WithValues("reporter", "slack")
	// Check if slack client is initialised
	if m.slackClient == nil {
		reqLogger.V(1).Info("Slack client is not initialised, can't send messages: %s", msg)
		return
	}
	// Send message to slack
	attachment := slack.Attachment{
		Color:      color,
		Title:      title,
		Text:       msg,
		AuthorName: BotName,
		Footer:     BotName,
		FooterIcon: BotLogo,
	}
	_, _, err := m.slackClient.PostMessage(m.slackChannel,
		slack.MsgOptionUsername(BotName),
		slack.MsgOptionAttachments(attachment))
	if err != nil {
		reqLogger.V(1).Info(fmt.Sprintf("can't send errors message to Slack: %v", err))
	}
}
