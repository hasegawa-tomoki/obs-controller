package obs

import (
	"log"

	"github.com/slack-go/slack"
)

type SlackContainer struct {
	Client  *slack.Client
	Channel string
}

func (s *SlackContainer) Send2slack(message string) {
	_, _, err := s.Client.PostMessage(
		s.Channel,
		slack.MsgOptionText(message, true),
	)
	if err != nil {
		log.Println(err)
	}
}
