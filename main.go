package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// The weatherData struct that handles the returned weather data
// from the API.
type weatherData struct {
	Name string `json:"name"`
	Main struct {
		Kelvin float64 `json:"temp"`
	} `json:"main"`
}

// Main entry point for the program.
func main() {
	http.HandleFunc("/", hello)
	http.HandleFunc("/weather/", weather)
	http.ListenAndServe(":8000", nil)
}

// Say hello!
func hello(writer http.ResponseWriter, req *http.Request) {
	writer.Write([]byte("Hello!"))
}

// weather is the http handler function for utilizing the weather API. It processes
// the URL, calls the query function, and writes the output of that function to the
// response stream. If an error object is returned by the query function, an Http 500
// error is written to the response stream.
func weather(writer http.ResponseWriter, req *http.Request) {
	city := strings.SplitN(req.URL.Path, "/", 3)[2]
	data, err := query(city)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(writer).Encode(data)
}

// query takes the name of a city as a string and queries the OpenWeatherMap API
// for weather data. This function either returns a weatherData struct of the 
// returned data, or an error object.
func query(city string) (weatherData, error) {
	resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?q=" + city)
	if err != nil {
		return weatherData{}, err
	}

	defer resp.Body.Close()

	var d weatherData

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return weatherData{}, err
	}

	return d, nil
}