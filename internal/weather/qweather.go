package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tinypanel-hub/internal/domain"
)

type QWeatherClient struct {
	baseURL     string
	apiKey      string
	bearerToken string
	location    string
	lang        string
	unit        string
	hours       string
	days        string
	httpClient  *http.Client
}

type QWeatherOptions struct {
	APIHost     string
	APIKey      string
	BearerToken string
	Location    string
	Lang        string
	Unit        string
	Hours       string
	Days        string
	HTTPClient  *http.Client
}

func NewQWeatherClient(opts QWeatherOptions) (*QWeatherClient, error) {
	host := strings.TrimSpace(opts.APIHost)
	location := strings.TrimSpace(opts.Location)
	apiKey := strings.TrimSpace(opts.APIKey)
	bearerToken := strings.TrimSpace(opts.BearerToken)
	if host == "" {
		return nil, errors.New("qweather api host is required")
	}
	if location == "" {
		return nil, errors.New("qweather location is required")
	}
	if apiKey == "" && bearerToken == "" {
		return nil, errors.New("qweather api_key or bearer_token is required")
	}

	baseURL, err := normalizeBaseURL(host)
	if err != nil {
		return nil, err
	}

	lang := strings.TrimSpace(opts.Lang)
	if lang == "" {
		lang = "zh"
	}
	unit := strings.TrimSpace(opts.Unit)
	if unit == "" {
		unit = "m"
	}
	hours := strings.TrimSpace(opts.Hours)
	if hours == "" {
		hours = "24h"
	}
	if !isAllowedValue(hours, "24h", "72h", "168h") {
		return nil, fmt.Errorf("unsupported qweather hourly range %q", hours)
	}
	days := strings.TrimSpace(opts.Days)
	if days == "" {
		days = "3d"
	}
	if !isAllowedValue(days, "3d", "7d", "10d", "15d", "30d") {
		return nil, fmt.Errorf("unsupported qweather daily range %q", days)
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &QWeatherClient{
		baseURL:     baseURL,
		apiKey:      apiKey,
		bearerToken: bearerToken,
		location:    location,
		lang:        lang,
		unit:        unit,
		hours:       hours,
		days:        days,
		httpClient:  httpClient,
	}, nil
}

func (c *QWeatherClient) Current(ctx context.Context) (domain.Weather, error) {
	current, err := c.current(ctx)
	if err != nil {
		return domain.Weather{}, err
	}
	hourly, err := c.hourly(ctx)
	if err != nil {
		return domain.Weather{}, err
	}
	daily, err := c.daily(ctx)
	if err != nil {
		return domain.Weather{}, err
	}

	current.Hourly = hourly
	current.Daily = daily
	return current, nil
}

func (c *QWeatherClient) current(ctx context.Context) (domain.Weather, error) {
	var payload qweatherNowResponse
	if err := c.get(ctx, "/v7/weather/now", &payload); err != nil {
		return domain.Weather{}, err
	}

	temp, err := parseQWeatherFloat(payload.Now.Temp, "temp")
	if err != nil {
		return domain.Weather{}, err
	}
	humidity, err := parseQWeatherFloat(payload.Now.Humidity, "humidity")
	if err != nil {
		return domain.Weather{}, err
	}

	updatedAt := parseQWeatherTime(payload.Now.ObsTime)
	if updatedAt.IsZero() {
		updatedAt = parseQWeatherTime(payload.UpdateTime)
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	return domain.Weather{
		Location:    c.location,
		Condition:   payload.Now.Text,
		Icon:        payload.Now.Icon,
		Temperature: temp,
		Humidity:    humidity,
		UpdatedAt:   updatedAt.UTC(),
	}, nil
}

func (c *QWeatherClient) hourly(ctx context.Context) ([]domain.WeatherHourlyForecast, error) {
	var payload qweatherHourlyResponse
	if err := c.get(ctx, "/v7/weather/"+c.hours, &payload); err != nil {
		return nil, err
	}

	items := make([]domain.WeatherHourlyForecast, 0, len(payload.Hourly))
	for _, item := range payload.Hourly {
		fxTime := parseQWeatherTime(item.FxTime)
		if fxTime.IsZero() {
			return nil, fmt.Errorf("qweather hourly fxTime %q is invalid", item.FxTime)
		}

		temp, err := parseQWeatherFloat(item.Temp, "hourly temp")
		if err != nil {
			return nil, err
		}
		humidity, err := parseQWeatherFloat(item.Humidity, "hourly humidity")
		if err != nil {
			return nil, err
		}
		precip, err := parseQWeatherFloat(item.Precip, "hourly precip")
		if err != nil {
			return nil, err
		}
		windSpeed, err := parseQWeatherFloat(item.WindSpeed, "hourly windSpeed")
		if err != nil {
			return nil, err
		}

		items = append(items, domain.WeatherHourlyForecast{
			Time:              fxTime.UTC(),
			Condition:         item.Text,
			Icon:              item.Icon,
			Temperature:       temp,
			Humidity:          humidity,
			Precipitation:     precip,
			PrecipProbability: parseOptionalQWeatherFloat(item.Pop),
			WindDirection:     item.WindDir,
			WindScale:         item.WindScale,
			WindSpeed:         windSpeed,
		})
	}
	return items, nil
}

func (c *QWeatherClient) daily(ctx context.Context) ([]domain.WeatherDailyForecast, error) {
	var payload qweatherDailyResponse
	if err := c.get(ctx, "/v7/weather/"+c.days, &payload); err != nil {
		return nil, err
	}

	items := make([]domain.WeatherDailyForecast, 0, len(payload.Daily))
	for _, item := range payload.Daily {
		tempMin, err := parseQWeatherFloat(item.TempMin, "daily tempMin")
		if err != nil {
			return nil, err
		}
		tempMax, err := parseQWeatherFloat(item.TempMax, "daily tempMax")
		if err != nil {
			return nil, err
		}
		humidity, err := parseQWeatherFloat(item.Humidity, "daily humidity")
		if err != nil {
			return nil, err
		}
		precip, err := parseQWeatherFloat(item.Precip, "daily precip")
		if err != nil {
			return nil, err
		}
		windSpeedDay, err := parseQWeatherFloat(item.WindSpeedDay, "daily windSpeedDay")
		if err != nil {
			return nil, err
		}
		windSpeedNight, err := parseQWeatherFloat(item.WindSpeedNight, "daily windSpeedNight")
		if err != nil {
			return nil, err
		}

		items = append(items, domain.WeatherDailyForecast{
			Date:               item.FxDate,
			Sunrise:            item.Sunrise,
			Sunset:             item.Sunset,
			ConditionDay:       item.TextDay,
			ConditionNight:     item.TextNight,
			IconDay:            item.IconDay,
			IconNight:          item.IconNight,
			TemperatureMin:     tempMin,
			TemperatureMax:     tempMax,
			Humidity:           humidity,
			Precipitation:      precip,
			WindDirectionDay:   item.WindDirDay,
			WindScaleDay:       item.WindScaleDay,
			WindSpeedDay:       windSpeedDay,
			WindDirectionNight: item.WindDirNight,
			WindScaleNight:     item.WindScaleNight,
			WindSpeedNight:     windSpeedNight,
		})
	}
	return items, nil
}

func (c *QWeatherClient) get(ctx context.Context, path string, dst qweatherPayload) error {
	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return err
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	q := u.Query()
	q.Set("location", c.location)
	q.Set("lang", c.lang)
	q.Set("unit", c.unit)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	} else {
		req.Header.Set("X-QW-Api-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qweather status %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dst.ResponseCode() != "200" {
		if dst.ResponseCode() == "" {
			return errors.New("qweather response missing code")
		}
		return fmt.Errorf("qweather code %s", dst.ResponseCode())
	}
	return nil
}

type qweatherPayload interface {
	ResponseCode() string
}

type qweatherNowResponse struct {
	Code       string `json:"code"`
	UpdateTime string `json:"updateTime"`
	Now        struct {
		ObsTime  string `json:"obsTime"`
		Temp     string `json:"temp"`
		Icon     string `json:"icon"`
		Text     string `json:"text"`
		Humidity string `json:"humidity"`
	} `json:"now"`
}

func (r *qweatherNowResponse) ResponseCode() string {
	return r.Code
}

type qweatherHourlyResponse struct {
	Code   string `json:"code"`
	Hourly []struct {
		FxTime    string `json:"fxTime"`
		Temp      string `json:"temp"`
		Icon      string `json:"icon"`
		Text      string `json:"text"`
		WindDir   string `json:"windDir"`
		WindScale string `json:"windScale"`
		WindSpeed string `json:"windSpeed"`
		Humidity  string `json:"humidity"`
		Pop       string `json:"pop"`
		Precip    string `json:"precip"`
	} `json:"hourly"`
}

func (r *qweatherHourlyResponse) ResponseCode() string {
	return r.Code
}

type qweatherDailyResponse struct {
	Code  string `json:"code"`
	Daily []struct {
		FxDate         string `json:"fxDate"`
		Sunrise        string `json:"sunrise"`
		Sunset         string `json:"sunset"`
		TempMax        string `json:"tempMax"`
		TempMin        string `json:"tempMin"`
		IconDay        string `json:"iconDay"`
		TextDay        string `json:"textDay"`
		IconNight      string `json:"iconNight"`
		TextNight      string `json:"textNight"`
		WindDirDay     string `json:"windDirDay"`
		WindScaleDay   string `json:"windScaleDay"`
		WindSpeedDay   string `json:"windSpeedDay"`
		WindDirNight   string `json:"windDirNight"`
		WindScaleNight string `json:"windScaleNight"`
		WindSpeedNight string `json:"windSpeedNight"`
		Humidity       string `json:"humidity"`
		Precip         string `json:"precip"`
	} `json:"daily"`
}

func (r *qweatherDailyResponse) ResponseCode() string {
	return r.Code
}

func normalizeBaseURL(host string) (string, error) {
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	u, err := url.Parse(host)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported qweather api host scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("qweather api host is invalid")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func parseQWeatherFloat(raw, field string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("qweather %s %q is invalid", field, raw)
	}
	return value, nil
}

func parseOptionalQWeatherFloat(raw string) *float64 {
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return &value
}

func parseQWeatherTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04Z07:00"} {
		t, err := time.Parse(layout, raw)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func isAllowedValue(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
