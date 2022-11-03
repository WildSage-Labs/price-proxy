package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type (
	ticker struct {
		data []byte
		age  time.Time
	}
)

func main() {
	e := echo.New()
	e.HideBanner = true
	data := map[string]ticker{}
	var mut sync.RWMutex

	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	output.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	output.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s:", i)
	}
	output.FormatFieldValue = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%s", i))
	}

	logger := zerolog.New(output).With().Timestamp().Logger()

	webClient := http.Client{
		Timeout: time.Second * 5,
	}

	e.GET("/prices/:ticker", func(c echo.Context) error {
		tickerId := c.Param("ticker")

		if _, ok := data[tickerId]; !ok {
			url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s?tickers=false&market_data=true&community_data=false&developer_data=false&sparkline=false", tickerId)
			// we dont have this key in the map, lets update it
			// add it to queue
			mut.Lock()
			defer mut.Unlock()

			data[tickerId] = ticker{
				data: []byte("{}"),
				age:  time.Now(),
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				e.Logger.Warn(fmt.Sprintf("Failed to create request! Error was: %s", err.Error()))
				return c.String(400, "Request error")
			}
			resp, err := webClient.Do(req)
			if err != nil {
				e.Logger.Warn(fmt.Sprintf("Failed to do web request! Error was: %s", err.Error()))
				return c.String(400, "Request error")
			}
			defer resp.Body.Close()
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				e.Logger.Warn(fmt.Sprintf("Failed to read response body! Error was: %s", err.Error()))
				return c.String(400, "Request error")
			}

			data[tickerId] = ticker{
				data: bodyBytes,
				age:  time.Now(),
			}

			c.Response().Header().Set("cached", "false")
			return c.JSONBlob(http.StatusOK, data[tickerId].data)
		}
		c.Response().Header().Set("cached", "true")
		c.Response().Header().Set("update-time", data[tickerId].age.Format(time.RFC3339))
		c.Response().Header().Set("price-age", fmt.Sprintf("%.2fs", time.Now().Sub(data[tickerId].age).Seconds()))
		return c.JSONBlob(http.StatusOK, data[tickerId].data)
	})

	go func() {
		logger.Info().Msg("Starting update thread")
		// Start the update thread
		url := ""
		for {
			time.Sleep(time.Second * 10)
			if len(data) == 0 {
				logger.Info().Msg("No tickers to update")
				continue
			}
			for k, _ := range data {
				url = fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s?tickers=false&market_data=true&community_data=false&developer_data=false&sparkline=false", k)
				logger.Info().Str("ticker", k).Msg("Updating ticker")
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					logger.Warn().Msg(fmt.Sprintf("Failed to create request! Error was: %s", err.Error()))
					continue
				}

				resp, err := webClient.Do(req)
				if err != nil {
					logger.Warn().Msg(fmt.Sprintf("Failed to do web request! Error was: %s", err.Error()))
					continue
				}

				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Warn().Msg(fmt.Sprintf("Failed to read response body! Error was: %s", err.Error()))
					continue
				}
				resp.Body.Close()
				mut.Lock()
				data[k] = ticker{
					data: bodyBytes,
					age:  time.Now(),
				}
				mut.Unlock()
				logger.Info().Str("ticker", k).Msg("Ticker updated")
			}

		}
	}()

	e.Logger.Fatal(e.Start(":1323"))
}
