package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQWeatherClientCurrent(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("X-QW-Api-Key")
		if r.URL.Query().Get("location") != "101020100" {
			t.Fatalf("location = %q, want 101020100", r.URL.Query().Get("location"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v7/weather/now":
			_, _ = w.Write([]byte(`{
				"code": "200",
				"updateTime": "2026-05-10T12:10+08:00",
				"now": {
					"obsTime": "2026-05-10T12:05+08:00",
					"temp": "24",
					"icon": "101",
					"text": "多云",
					"humidity": "72"
				}
			}`))
		case "/v7/weather/24h":
			_, _ = w.Write([]byte(`{
				"code": "200",
				"hourly": [
					{
						"fxTime": "2026-05-10T13:00+08:00",
						"temp": "25",
						"icon": "104",
						"text": "阴",
						"windDir": "东南风",
						"windScale": "1-3",
						"windSpeed": "8",
						"humidity": "70",
						"pop": "20",
						"precip": "0.0"
					}
				]
			}`))
		case "/v7/weather/3d":
			_, _ = w.Write([]byte(`{
				"code": "200",
				"daily": [
					{
						"fxDate": "2026-05-10",
						"sunrise": "05:01",
						"sunset": "18:40",
						"tempMax": "28",
						"tempMin": "20",
						"iconDay": "101",
						"textDay": "多云",
						"iconNight": "305",
						"textNight": "小雨",
						"windDirDay": "东南风",
						"windScaleDay": "1-3",
						"windSpeedDay": "8",
						"windDirNight": "东北风",
						"windScaleNight": "1-3",
						"windSpeedNight": "6",
						"humidity": "75",
						"precip": "1.2"
					}
				]
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewQWeatherClient(QWeatherOptions{
		APIHost:    server.URL,
		APIKey:     "test-key",
		Location:   "101020100",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	weather, err := client.Current(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gotAuth != "test-key" {
		t.Fatalf("X-QW-Api-Key = %q, want test-key", gotAuth)
	}
	if weather.Location != "101020100" || weather.Condition != "多云" || weather.Icon != "101" || weather.Temperature != 24 || weather.Humidity != 72 {
		t.Fatalf("unexpected weather: %+v", weather)
	}
	if len(weather.Hourly) != 1 || weather.Hourly[0].Condition != "阴" || weather.Hourly[0].Icon != "104" || weather.Hourly[0].Temperature != 25 {
		t.Fatalf("unexpected hourly forecast: %+v", weather.Hourly)
	}
	if len(weather.Daily) != 1 || weather.Daily[0].ConditionNight != "小雨" || weather.Daily[0].IconNight != "305" || weather.Daily[0].TemperatureMax != 28 {
		t.Fatalf("unexpected daily forecast: %+v", weather.Daily)
	}
	if weather.UpdatedAt.IsZero() {
		t.Fatal("updated_at is zero")
	}
}

func TestQWeatherClientUsesBearerTokenWhenProvided(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/v7/weather/now":
			_, _ = w.Write([]byte(`{"code":"200","now":{"temp":"1","text":"晴","humidity":"2"}}`))
		case "/v7/weather/24h":
			_, _ = w.Write([]byte(`{"code":"200","hourly":[]}`))
		case "/v7/weather/3d":
			_, _ = w.Write([]byte(`{"code":"200","daily":[]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewQWeatherClient(QWeatherOptions{
		APIHost:     server.URL,
		APIKey:      "test-key",
		BearerToken: "jwt-token",
		Location:    "101020100",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.Current(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer jwt-token" {
		t.Fatalf("Authorization = %q, want Bearer jwt-token", gotAuth)
	}
}
