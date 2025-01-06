package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// SendWebhook sends a Discord webhook message with the product information.
func SendWebhook(product Product) error {
	logger.Info().Msg("Product InStock!! \nSending Webhook")

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       product.Title,
				"description": fmt.Sprintf("%s\n", product.ShortDescription),
				"url":         fmt.Sprintf("https://store.ui.com/us/en/category/%s/products/%s", product.CollectionSlug, product.Slug),
				"fields": []map[string]interface{}{
					{
						"name":   "Price",
						"value":  fmt.Sprintf("$%d.%02d", product.Variants[0].DisplayPrice.Amount/100, product.Variants[0].DisplayPrice.Amount%100),
						"inline": true,
					},
					{
						"name":   "Status",
						"value":  product.Status,
						"inline": true,
					},
					{
						"name":   "Variant",
						"value":  product.Variants[0].ID,
						"inline": true,
					},
				},
				"thumbnail": map[string]string{
					"url": product.Thumbnail.URL,
				},
				"color": 0x0000ff,
				"footer": map[string]string{
					"text":     "Unifi Store Monitor",
					"icon_url": "https://tse3.mm.bing.net/th?id=OIP.RadjPrUUrLwqfVTEI5YqmwHaIV&pid=Api&P=0&w=300&h=300",
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to Marshal Payload")
		return err
	}

	resp, err := http.Post(DiscordWebhookURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to Send Webhook")
		return err
	}
	defer resp.Body.Close()

	logger.Info().Msg("Webhook Sent")
	return nil
}
