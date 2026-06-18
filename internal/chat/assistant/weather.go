package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const weatherAPIBase = "https://api.weatherapi.com/v1"

type weatherClient struct {
	apiKey string
	http   *http.Client
}

func newWeatherClient() *weatherClient {
	return &weatherClient{
		apiKey: os.Getenv("WEATHERAPI_KEY"),
		http:   http.DefaultClient,
	}
}

type weatherReport struct {
	Location struct {
		Name      string `json:"name"`
		Region    string `json:"region"`
		Country   string `json:"country"`
		Localtime string `json:"localtime"`
	} `json:"location"`
	Current struct {
		LastUpdated string  `json:"last_updated"`
		TempC       float64 `json:"temp_c"`
		FeelslikeC  float64 `json:"feelslike_c"`
		Condition   struct {
			Text string `json:"text"`
		} `json:"condition"`
		WindKph  float64 `json:"wind_kph"`
		WindDir  string  `json:"wind_dir"`
		Humidity int     `json:"humidity"`
		PrecipMm float64 `json:"precip_mm"`
	} `json:"current"`
	Forecast *struct {
		Forecastday []forecastDay `json:"forecastday"`
	} `json:"forecast,omitempty"`
}

type forecastDay struct {
	Date string `json:"date"`
	Day  struct {
		MaxtempC      float64 `json:"maxtemp_c"`
		MintempC      float64 `json:"mintemp_c"`
		MaxwindKph    float64 `json:"maxwind_kph"`
		TotalprecipMm float64 `json:"totalprecip_mm"`
		Condition     struct {
			Text string `json:"text"`
		} `json:"condition"`
	} `json:"day"`
}

// GetWeather fetches current conditions and, when days > 0, a multi-day forecast
// from WeatherAPI (https://www.weatherapi.com/).
func GetWeather(ctx context.Context, location string, days int) (string, error) {
	location = strings.TrimSpace(location)
	if location == "" {
		return "", fmt.Errorf("location is required")
	}

	client := newWeatherClient()
	if client.apiKey == "" {
		return "", fmt.Errorf("WEATHERAPI_KEY environment variable is not set")
	}

	if days > 0 {
		if days > 14 {
			days = 14
		}
		return client.fetchForecast(ctx, location, days)
	}

	return client.fetchCurrent(ctx, location)
}

func (c *weatherClient) fetchCurrent(ctx context.Context, location string) (string, error) {
	endpoint := weatherAPIBase + "/current.json?" + c.query(location)
	report, err := c.request(ctx, endpoint)
	if err != nil {
		return "", err
	}

	return formatCurrentWeather(report), nil
}

func (c *weatherClient) fetchForecast(ctx context.Context, location string, days int) (string, error) {
	params := c.query(location)
	params += "&days=" + strconv.Itoa(days)
	endpoint := weatherAPIBase + "/forecast.json?" + params

	report, err := c.request(ctx, endpoint)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(formatCurrentWeather(report))
	b.WriteString("\n\nForecast:\n")
	for _, day := range report.Forecast.Forecastday {
		fmt.Fprintf(&b, "- %s: %s, high %.0f°C, low %.0f°C, max wind %.0f kph, precipitation %.1f mm\n",
			day.Date,
			day.Day.Condition.Text,
			day.Day.MaxtempC,
			day.Day.MintempC,
			day.Day.MaxwindKph,
			day.Day.TotalprecipMm,
		)
	}

	return strings.TrimSpace(b.String()), nil
}

func (c *weatherClient) query(location string) string {
	return "key=" + url.QueryEscape(c.apiKey) + "&q=" + url.QueryEscape(location)
}

func (c *weatherClient) request(ctx context.Context, endpoint string) (*weatherReport, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weather API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read weather API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var report weatherReport
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, fmt.Errorf("failed to parse weather API response: %w", err)
	}

	return &report, nil
}

func formatCurrentWeather(report *weatherReport) string {
	loc := report.Location
	current := report.Current

	place := loc.Name
	if loc.Region != "" {
		place += ", " + loc.Region
	}
	if loc.Country != "" {
		place += ", " + loc.Country
	}

	return fmt.Sprintf(
		"Current weather in %s (local time %s, updated %s):\n"+
			"Condition: %s\n"+
			"Temperature: %.1f°C (feels like %.1f°C)\n"+
			"Wind: %.1f kph %s\n"+
			"Humidity: %d%%\n"+
			"Precipitation: %.1f mm",
		place,
		loc.Localtime,
		current.LastUpdated,
		current.Condition.Text,
		current.TempC,
		current.FeelslikeC,
		current.WindKph,
		current.WindDir,
		current.Humidity,
		current.PrecipMm,
	)
}
