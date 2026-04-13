package mail

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SESMailer sends email via the AWS SES v2 API using instance-role credentials.
type SESMailer struct {
	client *sesv2.Client
	from   string
	logger *slog.Logger
}

func NewSESMailer(ctx context.Context, region, from string, log *slog.Logger) (*SESMailer, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	return &SESMailer{
		client: sesv2.NewFromConfig(cfg),
		from:   from,
		logger: log,
	}, nil
}

func (m *SESMailer) Send(ctx context.Context, to, subject, body string) error {
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(m.from),
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject)},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body)},
				},
			},
		},
	}

	if _, err := m.client.SendEmail(ctx, input); err != nil {
		m.logger.Error("SES send failed", "to", to, "subject", subject, "err", err)
		return fmt.Errorf("ses send email: %w", err)
	}
	m.logger.Info("email sent via SES", "to", to, "subject", subject)
	return nil
}
