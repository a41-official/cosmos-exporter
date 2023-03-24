package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type votePenaltyCounter struct {
	MissCount    string `json:"miss_count"`
	AbstainCount string `json:"abstain_count"`
	SuccessCount string `json:"success_count"`
}

func SeiMetricHandler(w http.ResponseWriter, r *http.Request, ApiAddress string) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	address := r.URL.Query().Get("address")

	votePenaltyMissCount := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:        "vote_penalty_miss_count",
			Help:        "Vote penalty miss count",
			ConstLabels: ConstLabels,
		},
	)

	votePenaltyAbstainCount := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:        "vote_penalty_abstain_count",
			Help:        "Vote penalty abstain count",
			ConstLabels: ConstLabels,
		},
	)

	votePenaltySuccessCount := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:        "vote_penalty_success_count",
			Help:        "Vote penalty success count",
			ConstLabels: ConstLabels,
		},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(votePenaltyMissCount)
	registry.MustRegister(votePenaltyAbstainCount)
	registry.MustRegister(votePenaltySuccessCount)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying oracle feeder metrics")
		queryStart := time.Now()
		fmt.Println("address: ", address)
		fmt.Println("ApiAddress: ", ApiAddress)

		response, err := http.Get(ApiAddress + "/sei-protocol/sei-chain/oracle/validators/" + address + "/vote_penalty_counter")
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get oracle feeder metrics")
			return
		}
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse oracle feeder metrics")
			return
		}

		fmt.Println(response.StatusCode)
		fmt.Println(string(body))

		var data map[string]votePenaltyCounter
		err = json.Unmarshal(body, &data)
		if err != nil {
			fmt.Println("Error decoding JSON: ", err)
			return
		}

		fmt.Println(data)

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying oracle feeder metrics")

		fmt.Println(data["vote_penalty_counter"].MissCount)
		fmt.Println(data["vote_penalty_counter"].AbstainCount)
		fmt.Println(data["vote_penalty_counter"].SuccessCount)

		missCount, _ := strconv.ParseFloat(data["vote_penalty_counter"].MissCount, 64)
		abstainCount, _ := strconv.ParseFloat(data["vote_penalty_counter"].AbstainCount, 64)
		successCount, _ := strconv.ParseFloat(data["vote_penalty_counter"].SuccessCount, 64)

		fmt.Println(missCount)
		fmt.Println(abstainCount)
		fmt.Println(successCount)

		votePenaltyMissCount.Add(missCount)
		votePenaltyAbstainCount.Add(abstainCount)
		votePenaltySuccessCount.Add(successCount)

	}()
	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/sei").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}
