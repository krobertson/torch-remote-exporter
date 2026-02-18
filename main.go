package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	TORCH_HOST    = os.Getenv("TORCH_HOST")
	TORCH_PORT    = os.Getenv("TORCH_PORT")
	TORCH_PASS    = os.Getenv("TORCH_PASS")
	INTERVAL      = os.Getenv("INTERVAL")
	timerInterval = time.Minute
)

type StatusEnum int

const (
	STOPPED = iota
	STARTING
	RUNNING
	CRASHED
)

type serverStatus struct {
	SimSpeed    float64    `json:"simSpeed"`
	MemberCount int        `json:"memberCount"`
	Uptime      Duration   `json:"uptime"`
	Status      StatusEnum `json:"status"`
}

type playerStatus struct {
	ClientID     int64  `json:"clientID"`
	Name         string `json:"name"`
	PromoteLevel int    `json:"promoteLevel"`
}

type worldStatus struct {
	Name   string `json:"name"`
	SizeKb int    `json:"sizeKb"`
}

var (
	metricSimSpeed    = prometheus.NewGauge(prometheus.GaugeOpts{Name: "spaceengineers_sim_speed"})
	metricPlayerCount = prometheus.NewGauge(prometheus.GaugeOpts{Name: "spaceengineers_player_count"})
	metricGameReady   = prometheus.NewGauge(prometheus.GaugeOpts{Name: "spaceengineers_game_ready"})
	metricUptime      = prometheus.NewGauge(prometheus.GaugeOpts{Name: "spaceengineers_uptime"})
	metricGridCount   = prometheus.NewGauge(prometheus.GaugeOpts{Name: "spaceengineers_grid_count"})
	metricBannedCount = prometheus.NewGauge(prometheus.GaugeOpts{Name: "spaceengineers_banned_player_count"})
	metricWorldSize   = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "spaceengineers_world_size"}, []string{"world"})
)

func doServerStatus() error {
	var status *serverStatus
	err := makeRequest("/api/v1/server/status", &status)
	if err != nil {
		return err
	}

	metricSimSpeed.Set(status.SimSpeed)
	metricPlayerCount.Set(float64(status.MemberCount))
	metricGameReady.Set(float64(status.Status))
	metricUptime.Set(status.Uptime.Seconds())
	return nil
}

func doGetGridCount() error {
	var gridIds []int64
	err := makeRequest("/api/v1/grids", &gridIds)
	if err != nil {
		return err
	}

	metricGridCount.Set(float64(len(gridIds)))
	return nil
}

func doGetBannedCount() error {
	var playerIds []int64
	err := makeRequest("/api/v1/players/banned", &playerIds)
	if err != nil {
		return err
	}

	metricBannedCount.Set(float64(len(playerIds)))
	return nil
}

func doGetWorlds() error {
	var worldIds []string
	err := makeRequest("/api/v1/worlds", &worldIds)
	if err != nil {
		return err
	}

	for _, id := range worldIds {
		var world *worldStatus
		err = makeRequest(fmt.Sprintf("/api/v1/worlds/%s", id), &world)
		if err != nil {
			return err
		}

		metricWorldSize.WithLabelValues(world.Name).Set(float64(world.SizeKb))
	}

	return nil
}

var metrics []func() error = []func() error{
	doServerStatus,
	doGetGridCount,
	doGetBannedCount,
	doGetWorlds,
}

func metricsLoop() {
	log.Printf("poll metrics every %s", timerInterval.String())
	// loop all metrics on startup
	log.Println("processing metrics")
	for _, f := range metrics {
		if err := f(); err != nil {
			log.Printf("error processing metrics: %v", err)
		}
	}

	ticker := time.NewTicker(timerInterval)
	defer ticker.Stop()

	// loop on the ticker collecting metrics
	for range ticker.C {
		log.Println("processing metrics")

		for _, f := range metrics {
			if err := f(); err != nil {
				log.Printf("error processing metrics: %v", err)
			}
		}
	}
}

func main() {
	if TORCH_HOST == "" || TORCH_PORT == "" || TORCH_PASS == "" {
		log.Fatal("Set TORCH_HOST, TORCH_PORT, and TORCH_PASS")
	}

	if INTERVAL != "" {
		var err error
		timerInterval, err = time.ParseDuration(INTERVAL)
		if err != nil {
			log.Fatalf("Failed to parse INTERVAL: %v", err)
			return
		}
	}

	prometheus.MustRegister(metricSimSpeed)
	prometheus.MustRegister(metricPlayerCount)
	prometheus.MustRegister(metricGameReady)
	prometheus.MustRegister(metricUptime)
	prometheus.MustRegister(metricGridCount)
	prometheus.MustRegister(metricBannedCount)
	prometheus.MustRegister(metricWorldSize)

	go metricsLoop()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9090", nil)
}

func makeRequest(path string, dest any) error {
	url := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", TORCH_HOST, TORCH_PORT),
		Path:   path,
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+TORCH_PASS)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	return dec.Decode(&dest)
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		parts := strings.SplitN(value, ":", 3)
		if len(parts) != 3 {
			return errors.New("duration had unexpected format")
		}

		s := fmt.Sprintf("%sh%sm%ss", parts[0], parts[1], parts[2])

		var err error
		d.Duration, err = time.ParseDuration(s)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}
