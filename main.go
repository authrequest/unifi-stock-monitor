package main

import (
	"fmt"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
)

// const (
// 	Power_Dist_Pro_UUID = "c5ddda0a-b78f-4ec3-bcd4-f5078313c480"
// 	Power_Dist_HD_UUID  = "ce18f263-6330-4b80-bdeb-5ce835d730c0"
// )

var logger = zerolog.New(
	zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		FormatLevel: func(i interface{}) string {
			return fmt.Sprintf("[%-6s]", i) // Custom level format in square brackets
		},
	},
).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()

func main() {
	logger.Info().Msg("Starting...")

	store := NewUnifiStore()
	go store.Start()
	select {}
}
