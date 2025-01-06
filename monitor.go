package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	c "github.com/ostafen/clover"
	"github.com/rs/zerolog/log"
)

const (
	HomeURL           = "https://store.ui.com/us/en"
	DiscordWebhookURL = "https://discord.com/api/webhooks/898013000000000000/3gWz4cG9xYh4gU7lHl3OcE5k8hBw0bZ4d4Y" // Replace with your Discord webhook URL
)

type UnifiStore struct {
	ProdURLs []string
	Headers  map[string]string
}

type Product struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Status           string    `json:"status"`
	ShortDescription string    `json:"shortDescription"`
	CollectionSlug   string    `json:"collectionSlug"`
	Slug             string    `json:"slug"`
	Thumbnail        Thumbnail `json:"thumbnail"`
	Variants         []Variant `json:"variants"`
}

type Thumbnail struct {
	URL string `json:"url"`
}

type Variant struct {
	ID           string `json:"id"`
	DisplayPrice struct {
		Amount   int    `json:"amount"`
		Currency string `json:"currency"`
	} `json:"displayPrice"`
}

type PageProps struct {
	Product Product `json:"product"`
}

type Response struct {
	PageProps PageProps `json:"pageProps"`
}

// NewUnifiStore initializes and returns a pointer to a new UnifiStore instance.
// It sets up default HTTP headers, including user-agent and cookie headers,
// necessary for making requests to the Unifi store. The Initialized field is
// set to false by default, indicating that the store has not yet loaded any
// product data.

func NewUnifiStore() *UnifiStore {
	return &UnifiStore{
		Headers: map[string]string{
			"accept":          "*/*",
			"accept-language": "en-US,en;q=0.6",
			"user-agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			"priority":        "u=0, i",
		},
	}
}

// FetchXAppBuild fetches the x-app-build header from the Unifi store homepage.
//
// The x-app-build header is used to construct the URL for fetching product information.
// If the header is not found, an error is returned.
//
// FetchXAppBuild will set the ProdURL field of the UnifiStore object to the correct
// URL for fetching the product information.
func (store *UnifiStore) FetchXAppBuild() error {
	logger.Info().Msg("Fetching X-App-Build...")
	req, err := http.NewRequest("GET", HomeURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request")
		return err
	}
	for key, value := range store.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to Fetch Home Page")
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to Read Response")
		return err
	}

	soup := string(body)
	regexPattern := `https://assets\.ecomm\.ui\.com/_next/static/([a-zA-Z0-9]+)/_buildManifest\.js`
	re := regexp.MustCompile(regexPattern)

	matches := re.FindStringSubmatch(soup)
	if len(matches) > 1 {
		buildID := matches[1]
		store.ProdURLs = []string{
			fmt.Sprintf("https://store.ui.com/_next/data/%s/us/en/category/all-power-tech/collections/power-tech/products/usp-pdu-pro.json", buildID),
			fmt.Sprintf("https://store.ui.com/_next/data/%s/us/en/category/network-storage/collections/unifi-new-integrations-network-storage/products/unas-pro.json", buildID),
		}
		logger.Info().Msg(fmt.Sprintf("Extracted X-App-Build: %s", buildID))
		return nil
	}
	//

	logger.Error().Msg("X-App-Build Not Found")
	return fmt.Errorf("X-App-Build Not Found")
}

// MonitorProduct fetches the product information from the Unifi store and
// returns the product or an error if the request fails or the JSON is
// malformed.
func (store *UnifiStore) MonitorProduct(url string) (Product, error) {
	logger.Info().Msg("Monitoring Products")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request")
		return Product{}, err
	}
	for key, value := range store.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Request Failed")
		return Product{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed Response Read")
		return Product{}, err
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return Product{}, fmt.Errorf("failed to parse JSON: %v", err)
	}
	logger.Info().Msg(fmt.Sprintf("Product: %+v", response.PageProps.Product.Title))
	return response.PageProps.Product, nil
}

// Start runs an infinite loop where it fetches the X-App-Build and monitors
// the product. If either operation fails, it logs the error and waits 30
// seconds before trying again.
func (store *UnifiStore) Start() {
	logger.Info().Msg("Starting Monitor")
	var err error
	db, err := c.Open("clover-db")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to open database")
	}
	db.CreateCollection("products")
	for {
		if err := store.FetchXAppBuild(); err != nil {
			logger.Error().Err(err).Msg("Failed to Fetch X-App-Build")
			time.Sleep(30 * time.Second)
			continue
		}

		for _, url := range store.ProdURLs {
			product, err := store.MonitorProduct(url)
			doc := c.NewDocument()
			doc.Set("products", product)

			docId, _ := db.InsertOne("products", doc)
			logger.Info().Msg(fmt.Sprintf("Inserted Document: %s", docId))
			db.ExportCollection("products", "products.json")
			defer db.Close()
			if err != nil {
				logger.Error().Err(err).Msg("Failed to Monitor Product")
				time.Sleep(30 * time.Second)
				continue
			}

			// redditClient, err := CreateRedditClient()
			// if err != nil {
			// 	logger.Error().Err(err).Msg("Failed to create Reddit client")
			// 	time.Sleep(30 * time.Second)
			// 	continue
			// }

			if product.Status == "Available" {
				err = SendWebhook(product)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to send Discord webhook")
				}
				// err = CreatePost(redditClient, product)
				// if err != nil {
				// 	logger.Error().Err(err).Msg("Failed to create Reddit post")
				// 	time.Sleep(30 * time.Second)
				// 	continue
				// }
			}
		}
		logger.Info().Msg("Product Not InStock - Checking Again in 30s")
		// logger.Info().Msg(fmt.Sprintf("Product: %+v", product))
		time.Sleep(30 * time.Second)
	}
}
