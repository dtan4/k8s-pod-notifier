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
	ch, err := c.api.GetChannelInfo(channel)
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve Slack channel info")
	}

	return ch.ID, nil
}

// PostAttachment posts message with attachment
func (c *Client) PostAttachment(channelID string, fields map[string]string) error {
	attachmentFields := []slack.AttachmentField{}

	for k, v := range fields {
		attachmentFields = append(attachmentFields, slack.AttachmentField{
			Title: k,
			Value: v,
		})
	}

	params := slack.PostMessageParameters{
		Attachments: []slack.Attachment{
			slack.Attachment{
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
