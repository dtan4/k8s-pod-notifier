package slack

import (
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

// Client represents the wrapper of Slack API client
type Client struct {
	api *slack.Client
}

// NewClient creates Client object
func NewClient(token string) *Client {
	return &Client{
		api: slack.New(token),
	}
}

// GetChannelID retrieves internal ID of the given channel
func (c *Client) GetChannelID(channel string) (string, error) {
	chs, err := c.api.GetChannels(true)
	if err != nil {
		return "", errors.Wrap(err, "failed to list Slack channels")
	}

	for _, ch := range chs {
		if ch.Name == channel {
			return ch.ID, nil
		}
	}

	return "", errors.Errorf("channel %s is not found", channel)
}

// PostMessageWithAttachment posts message with attachment
func (c *Client) PostMessageWithAttachment(channelID, color, title, text string, fields []*AttachmentField) error {
	attachmentFields := []slack.AttachmentField{}

	for _, field := range fields {
		attachmentFields = append(attachmentFields, slack.AttachmentField{
			Title: field.Title,
			Value: field.Value,
			Short: true,
		})
	}

	params := slack.PostMessageParameters{
		Attachments: []slack.Attachment{
			slack.Attachment{
				Title:  title,
				Text:   text,
				Color:  color,
				Fields: attachmentFields,
			},
		},
	}

	_, _, err := c.api.PostMessage(channelID, "", params)
	if err != nil {
		return errors.Wrap(err, "failed to post message to Slack")
	}

	return nil
}
