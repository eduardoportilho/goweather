package main

import (
    "encoding/json"
    "log"
    "net/http"
    "strings"
    "time"
)

func main() {
    multiWeatherProvider := MultiWeatherProvider{
        OpenWeatherMapProvider{apiKey: "your-key-here"},
        WeatherUndergroundProvider{apiKey: "your-key-here"},
    }

    http.HandleFunc("/weather/", func(writer http.ResponseWriter, request *http.Request) {
        begin := time.Now()
        city := strings.SplitN(request.URL.Path, "/", 3)[2]

        temp, err := multiWeatherProvider.temperature(city)
        if err != nil {
            http.Error(writer, err.Error(), http.StatusInternalServerError)
            return
        }

        writer.Header().Set("Content-Type", "application/json; charset=utf-8")
        json.NewEncoder(writer).Encode(map[string]interface{}{
            "city": city,
            "temp": temp,
            "took": time.Since(begin).String(),
        })
    })

    http.ListenAndServe(":8080", nil)
}

type WeatherProvider interface {
    temperature(city string) (float64, error) // in Kelvin, naturally
}

type OpenWeatherMapProvider struct {
    apiKey string
}

func (openWeatherMapProvider OpenWeatherMapProvider) temperature(city string) (float64, error) {
    resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?APPID=" + openWeatherMapProvider.apiKey + "&q=" + city)
    if err != nil {
        return 0, err
    }

    defer resp.Body.Close()

    var openWeatherMapResponse struct {
        Main struct {
            Kelvin float64 `json:"temp"`
        } `json:"main"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&openWeatherMapResponse); err != nil {
        return 0, err
    }

    log.Printf("openWeatherMap: %s: %.2f", city, openWeatherMapResponse.Main.Kelvin)
    return openWeatherMapResponse.Main.Kelvin, nil
}

type WeatherUndergroundProvider struct {
    apiKey string
}

func (weatherUndergroundProvider WeatherUndergroundProvider) temperature(city string) (float64, error) {
    resp, err := http.Get("http://api.wunderground.com/api/" + weatherUndergroundProvider.apiKey + "/conditions/q/" + city + ".json")
    if err != nil {
        return 0, err
    }

    defer resp.Body.Close()

    var weatherUndergroundResponse struct {
        Observation struct {
            Celsius float64 `json:"temp_c"`
        } `json:"current_observation"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&weatherUndergroundResponse); err != nil {
        return 0, err
    }

    kelvin := weatherUndergroundResponse.Observation.Celsius + 273.15
    log.Printf("weatherUnderground: %s: %.2f", city, kelvin)
    return kelvin, nil
}

type MultiWeatherProvider []WeatherProvider

func (multiWeatherProvider MultiWeatherProvider) temperature(city string) (float64, error) {
    sum := 0.0

    for _, provider := range multiWeatherProvider {
        temperatureKelvin, err := provider.temperature(city)
        if err != nil {
            return 0, err
        }

        sum += temperatureKelvin
    }

    return sum / float64(len(multiWeatherProvider)), nil
}
