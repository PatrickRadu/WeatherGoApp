package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type Weather struct {
	Address     string        `json:"address"`
	RequestDate string        `json:"requestdate"`
	Days        []WeatherDays `json:"days"`
}

type WeatherDays struct {
	Description    string  `json:"description"`
	Temperature    float64 `json:"temperature"`
	TemperatureMax float64 `json:"temperaturemax"`
	TemperatureMin float64 `json:"temperaturemin"`
	Datetime       string  `json:"datetime"`
	FeelsLike      float64 `json:"feelslike"`
}

var apiKey string
var redisClient *redis.Client
var ctx = context.Background()

func main() {

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	redisClient = client

	e := echo.New()
	err := godotenv.Load()
	if err != nil {
		e.Logger.Fatal("Error loading .env file")
	}
	secretKey := os.Getenv("key")
	apiKey = secretKey
	e.GET("/", getWeather)
	e.POST("/location", getWeatherInLocation)
	e.Logger.Fatal(e.Start(":8080"))
}

func getWeatherInLocation(c echo.Context) error {
	location := c.FormValue("location")
	cachedWeather, err := redisClient.Get(ctx, location).Result()
	if err == nil {
		return c.String(http.StatusOK, cachedWeather)
	}
	resp, err := http.Get("https://weather.visualcrossing.com/VisualCrossingWebServices/rest/services/timeline/" + location + "?unitGroup=us&include=days&key=" + apiKey + "&contentType=json")
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get weather data from api")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to read weather data body")
	}
	jsonData := string(body)
	var weatherData map[string]interface{}
	err = json.Unmarshal([]byte(jsonData), &weatherData)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to parse weather data")
	}
	varResponse := Weather{}
	varResponse.Address = weatherData["address"].(string)
	varResponse.RequestDate = time.Now().Format("2006-01-02")
	for _, day := range weatherData["days"].([]interface{}) {
		dayMap := day.(map[string]interface{})
		varResponse.Days = append(varResponse.Days, WeatherDays{
			Description:    dayMap["description"].(string),
			Temperature:    dayMap["temp"].(float64),
			TemperatureMax: dayMap["tempmax"].(float64),
			TemperatureMin: dayMap["tempmin"].(float64),
			Datetime:       dayMap["datetime"].(string),
			FeelsLike:      dayMap["feelslike"].(float64),
		})
	}
	jsonResponse, err := json.Marshal(varResponse)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to MARSHALL weather data")
	}
	redisClient.Set(ctx, location, jsonResponse, 1*time.Hour)
	return c.JSON(http.StatusOK, varResponse)
}

func getWeather(c echo.Context) error {
	resp, err := http.Get("https://weather.visualcrossing.com/VisualCrossingWebServices/rest/services/timeline/Cluj?unitGroup=us&include=days&key=" + apiKey + "&contentType=json")
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get weather data")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to read weather data body")
	}
	// i want to make this to json
	jsonData := string(body)
	var weatherData map[string]interface{}
	err = json.Unmarshal([]byte(jsonData), &weatherData)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to parse weather data")
	}

	varResponse := Weather{}
	varResponse.Address = weatherData["address"].(string)
	varResponse.RequestDate = time.Now().Format("2006-01-02")
	varResponse.Days = []WeatherDays{}
	for _, day := range weatherData["days"].([]interface{}) {
		dayMap := day.(map[string]interface{})
		varResponse.Days = append(varResponse.Days, WeatherDays{
			Description:    dayMap["description"].(string),
			Temperature:    dayMap["temp"].(float64),
			TemperatureMax: dayMap["tempmax"].(float64),
			TemperatureMin: dayMap["tempmin"].(float64),
			Datetime:       dayMap["datetime"].(string),
			FeelsLike:      dayMap["feelslike"].(float64),
		})
	}
	return c.JSON(http.StatusOK, varResponse)
}
