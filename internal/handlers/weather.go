package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

// WeatherHandler proxies requests to the open-meteo forecast API.
func WeatherHandler(c echo.Context) error {
	lat := c.QueryParam("lat")
	long := c.QueryParam("long")

	if lat == "" || long == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "lat and long query params are required")
	}

	forecastURL := url.URL{
		Scheme: "https",
		Host:   "api.open-meteo.com",
		Path:   "/v1/forecast",
	}

	query := forecastURL.Query()
	query.Set("latitude", lat)
	query.Set("longitude", long)
	query.Set("daily", "temperature_2m_min,temperature_2m_max")
	query.Set("current", "temperature_2m")
	query.Set("timezone", "America/Chicago")
	query.Set("forecast_days", "1")
	forecastURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(c.Request().Context(), http.MethodGet, forecastURL.String(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to build weather request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("weather request failed: %v", err))
	}
	defer resp.Body.Close()

	var payload any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "invalid response from weather service")
	}

	return c.JSON(resp.StatusCode, payload)
}
