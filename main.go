package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"sync"
	"time"
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

			data[tickerId] = []byte("{}")

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

			data[tickerId] = bodyBytes

			c.Response().Header().Set("cached", "false")
			return c.JSONBlob(http.StatusOK, data[tickerId])
		}
		c.Response().Header().Set("cached", "true")
		return c.JSONBlob(http.StatusOK, data[tickerId])
	})

	e.Logger.Fatal(e.Start(":1323"))
}
