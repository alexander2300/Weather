package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type WeatherResponse struct {
	Date        string  `json:"date"`
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
}

type geocodeResult struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Country   string  `json:"country"`
	} `json:"results"`
}

func getWeather(date, location string) (*WeatherResponse, error) {
	// 1. Geocode location
	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1", url.QueryEscape(location))
	resp, err := http.Get(geoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to geocode location: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read geocode response: %v", err)
	}
	var geo geocodeResult
	if err := json.Unmarshal(body, &geo); err != nil {
		return nil, fmt.Errorf("failed to parse geocode response: %v", err)
	}
	if len(geo.Results) == 0 {
		return nil, fmt.Errorf("location not found")
	}
	lat := geo.Results[0].Latitude
	lon := geo.Results[0].Longitude
	resolvedLoc := fmt.Sprintf("%s, %s", geo.Results[0].Name, geo.Results[0].Country)

	// 2. Query Open-Meteo for historical weather
	weatherURL := fmt.Sprintf("https://archive-api.open-meteo.com/v1/archive?latitude=%f&longitude=%f&start_date=%s&end_date=%s&daily=temperature_2m_max,temperature_2m_min,weathercode&timezone=auto", lat, lon, date, date)
	fmt.Println("Weather URL: ", weatherURL)
	wresp, err := http.Get(weatherURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather: %v", err)
	}
	defer wresp.Body.Close()
	wbody, err := ioutil.ReadAll(wresp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read weather response: %v", err)
	}
	// Parse weather response
	var wdata struct {
		Daily struct {
			TemperatureMax []float64 `json:"temperature_2m_max"`
			TemperatureMin []float64 `json:"temperature_2m_min"`
			Weathercode    []int     `json:"weathercode"`
		} `json:"daily"`
	}
	if err := json.Unmarshal(wbody, &wdata); err != nil {
		return nil, fmt.Errorf("failed to parse weather response: %v", err)
	}
	if len(wdata.Daily.TemperatureMax) == 0 {
		return nil, fmt.Errorf("no weather data for this date/location")
	}
	// Map weathercode to condition
	code := wdata.Daily.Weathercode[0]
	condition := weatherCodeToString(code)
	// Use average of min/max temp
	temp := (wdata.Daily.TemperatureMax[0] + wdata.Daily.TemperatureMin[0]) / 2
	temp = temp*9/5 + 32 // Convert to Fahrenheit
	return &WeatherResponse{
		Date:        date,
		Location:    resolvedLoc,
		Temperature: temp,
		Condition:   condition,
	}, nil
}

func weatherCodeToString(code int) string {
	// Open-Meteo weather codes: https://open-meteo.com/en/docs#api_form
	switch code {
	case 0:
		return "Clear sky"
	case 1, 2, 3:
		return "Mainly clear, partly cloudy, and overcast"
	case 45, 48:
		return "Fog and depositing rime fog"
	case 51, 53, 55:
		return "Drizzle: Light, moderate, and dense intensity"
	case 56, 57:
		return "Freezing Drizzle: Light and dense intensity"
	case 61, 63, 65:
		return "Rain: Slight, moderate and heavy intensity"
	case 66, 67:
		return "Freezing Rain: Light and heavy intensity"
	case 71, 73, 75:
		return "Snow fall: Slight, moderate, and heavy intensity"
	case 77:
		return "Snow grains"
	case 80, 81, 82:
		return "Rain showers: Slight, moderate, and violent"
	case 85, 86:
		return "Snow showers slight and heavy"
	case 95:
		return "Thunderstorm: Slight or moderate"
	case 96, 99:
		return "Thunderstorm with hail"
	default:
		return "Unknown"
	}
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	fmt.Println(r.Host, r.Body, r.Method, r.URL)
	date := r.URL.Query().Get("date")
	location := r.URL.Query().Get("location")
	if date == "" || location == "" {
		http.Error(w, "Missing date or location parameter", http.StatusBadRequest)
		return
	}
	weather, err := getWeather(date, location)
	if err != nil {
		http.Error(w, "Failed to get weather data", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(weather)
	fmt.Println()
}

func getLANIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		// Skip interfaces that are down or loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue // skip interface if can't get addresses
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Check for valid IPv4 LAN address
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an IPv4 address
			}

			// Match private IP ranges
			if isPrivateIP(ip) {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no LAN IP found")
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsPrivate()
}

func main() {
	http.HandleFunc("/weather", weatherHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	num, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("Invalid PORT: %v", err)
	}

	lanIP, err := getLANIP()
	if err != nil {
		log.Fatalf("Failed to get LAN IP: %v", err)
	}

	for {
		portStr := strconv.Itoa(num)
		fmt.Printf("Trying to start server on host %s, port %s\n", lanIP, portStr)
		err := http.ListenAndServe(":"+portStr, nil)
		if err != nil {
			fmt.Printf("Port %s in use or failed: %v\n", portStr, err)
			num++ // try next port
		} else {
			break // success (though in practice ListenAndServe blocks)
		}
	}
}
