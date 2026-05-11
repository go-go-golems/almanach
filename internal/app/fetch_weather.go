package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// wttrResponse represents the subset of wttr.in JSON we care about.
type wttrResponse struct {
	CurrentCondition []struct {
		TempC         string `json:"temp_C"`
		Humidity      string `json:"humidity"`
		WindSpeedKmph string `json:"windspeedKmph"`
		WeatherDesc   []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
	} `json:"current_condition"`
	Weather []struct {
		MaxTempC string `json:"maxtempC"`
		MinTempC string `json:"mintempC"`
	} `json:"weather"`
}

// fetchWeather fetches current weather from wttr.in (no API key needed).
// Returns nil if the API is unavailable.
func fetchWeather(cfg Config) *WeatherData {
	location := "auto"
	// TODO: read location from cfg or request override

	url := fmt.Sprintf("https://wttr.in/%s?format=j1", location)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var wttr wttrResponse
	if err := json.Unmarshal(body, &wttr); err != nil {
		return nil
	}

	if len(wttr.CurrentCondition) == 0 {
		return nil
	}

	cc := wttr.CurrentCondition[0]
	condition := ""
	if len(cc.WeatherDesc) > 0 {
		condition = cc.WeatherDesc[0].Value
	}

	high, low := "", ""
	if len(wttr.Weather) > 0 {
		high = wttr.Weather[0].MaxTempC + "°C"
		low = wttr.Weather[0].MinTempC + "°C"
	}

	return &WeatherData{
		Temp:      cc.TempC + "°C",
		High:      high,
		Low:       low,
		Condition: condition,
		Humidity:  cc.Humidity + "%",
		Wind:      cc.WindSpeedKmph + " km/h",
	}
}
