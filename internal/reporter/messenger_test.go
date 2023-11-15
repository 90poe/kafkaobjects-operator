package reporter_test

import (
	"testing"
	"time"

	"github.com/90poe/kafkaobjects-operator/internal/reporter"
	"github.com/90poe/kafkaobjects-operator/internal/reporter/mock_slack"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMessenger_Send(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mSlack := mock_slack.NewMockSlack(ctrl)
	// Use Client & URL from our local test server
	m, err := reporter.New(
		"",
		reporter.SlackChannel("test-channel"),
		reporter.SlackClient(mSlack),
		reporter.TickInterval(5*time.Second),
	)
	require.NoError(t, err)
	mSlack.EXPECT().PostMessage("test-channel", gomock.Any()).Return("", "", nil)
	m.Send("test message 1", reporter.OKMessage)
	time.Sleep(7 * time.Second)
	mSlack.EXPECT().PostMessage("test-channel", gomock.Any()).Return("", "", nil)
	m.Send("test message 2", reporter.OKMessage)
	time.Sleep(7 * time.Second)
}
