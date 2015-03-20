package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// The weatherData struct that handles the returned weather data
// from the API.
type weatherData struct {
	Name string `json:"name"`
	Main struct {
		Kelvin float64 `json:"temp"`
	} `json:"main"`
}

// Weather provider interface
type weatherProvider interface {
	temperature(city string) (float64, error)
}

type openWeatherMap struct{}
type weatherUnderground struct{}
type multiWeatherProvider []weatherProvider

var wuKey string
var mw multiWeatherProvider

// Main entry point for the program.
func main() {
	getAPIKeys()

	mw = multiWeatherProvider{
		openWeatherMap{},
		weatherUnderground{},
	}

	http.HandleFunc("/", hello)
	http.HandleFunc("/weather/", weather)

	fmt.Println("Listening on :8000")
	http.ListenAndServe(":8000", nil)
}

// Say hello!
func hello(writer http.ResponseWriter, req *http.Request) {
	writer.Write([]byte("Hello!"))
}

func getAPIKeys() {
	// Weather Underground
	key, err := ioutil.ReadFile("weatherunderground.key")
	if err != nil {
		fmt.Printf("Unable to read weatherunderground keyfile.\n")
		fmt.Println(err)
	} else {
		fmt.Printf("Weather Underground API key loaded: %s\n", key)
	}
	wuKey = string(key)
}

// weather is the http handler function for utilizing the weather API. It processes
// the URL, calls the query function, and writes the output of that function to the
// response stream. If an error object is returned by the query function, an Http 500
// error is written to the response stream.
func weather(writer http.ResponseWriter, req *http.Request) {
	begin := time.Now()
	city := strings.SplitN(req.URL.Path, "/", 3)[2]

	temp, err := mw.temperature(city)
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

}

// query takes the name of a city as a string and queries the OpenWeatherMap API
// for weather data. This function either returns a weatherData struct of the
// returned data, or an error object.
func (w openWeatherMap) temperature(city string) (float64, error) {
	resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?q=" + city)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var d struct {
		Main struct {
			Kelvin float64 `json:"temp"`
		} `json:"main"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	fmt.Printf("OpenWeatherMap responded with %.2fK for %s\n", d.Main.Kelvin, city)

	return d.Main.Kelvin, nil
}

func (w weatherUnderground) temperature(city string) (float64, error) {
	if wuKey == "" {
		return 0, errors.New("Weather Underground API key must be set")
	}

	resp, err := http.Get("http://api.wunderground.com/api/" + wuKey + "/conditions/q/" + city + ".json")
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var d struct {
		Observation struct {
			Celcius float64 `json:"temp_c"`
		} `json:"current_observation"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	kelvin := d.Observation.Celcius + 273.15
	fmt.Printf("Weather Underground responded with %.2fK for %s\n", kelvin, city)

	return kelvin, nil
}

func (w multiWeatherProvider) temperature(city string) (float64, error) {
	// Make one channel for temperatures and one channel for errors.
	// Each provider will push a value into only one channel.
	temps := make(chan float64, len(w))
	errs := make(chan error, len(w))

	// For each provider, spawn a goroutine with an anonymous function.
	// That function will invoke the temperature method and forward the response.
	for _, provider := range w {
		go func(p weatherProvider) {
			k, err := p.temperature(city)
			if err != nil {
				errs <- err
				return
			}
			temps <- k
		}(provider)
	}

	sum := 0.0

	// Collect a temperature or error from each provider
	for i := 0; i < len(w); i++ {
		select {
		case temp := <-temps:
			f := (temp * 1.8) - 459.67
			fmt.Printf("%.2fK converts to %.2fF\n", temp, f)
			sum += f
		case err := <-errs:
			return 0, err
		}
	}

	return sum / float64(len(w)), nil
}
