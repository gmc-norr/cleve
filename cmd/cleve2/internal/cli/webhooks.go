package cli

import (
	"context"
	"log/slog"

	"github.com/gmc-norr/cleve"
	"github.com/maehler/webhook"
)

func SendWebhookMessage(ctx context.Context, client *webhook.Client, payload cleve.WebhookMessage) error {
	if client == nil {
		return nil
	}

	res, err := client.SendContext(ctx, payload)
	if err == nil {
		slog.Info("successfully sent webhook message", "message_type", payload.MessageType, "attempts", res.Attempts, "status", res.Response.Status)
	} else {
		slog.Error("failed to send webhook message", "message_type", payload.MessageType, "attempts", res.Attempts, "error", err)
	}
	return err
}
