package main

import (
	"encoding/json"
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

	votePenaltyCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "cosmos_sei_oracle_vote_penalty_count",
			Help:        "Vote penalty miss count",
			ConstLabels: ConstLabels,
		},
		[]string{"type"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(votePenaltyCount)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying oracle feeder metrics")
		queryStart := time.Now()

		response, err := http.Get(ApiAddress + "/sei-protocol/sei-chain/oracle/validators/" + address + "/vote_penalty_counter")
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get oracle feeder metrics")
			return
		}

		body, err := ioutil.ReadAll(response.Body)
		defer response.Body.Close()

		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse oracle feeder metrics")
			return
		}

		var data map[string]votePenaltyCounter
		err = json.Unmarshal(body, &data)
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Error decoding JSON")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying oracle feeder metrics")

		missCount, _ := strconv.ParseFloat(data["vote_penalty_counter"].MissCount, 64)
		abstainCount, _ := strconv.ParseFloat(data["vote_penalty_counter"].AbstainCount, 64)
		successCount, _ := strconv.ParseFloat(data["vote_penalty_counter"].SuccessCount, 64)

		votePenaltyCount.WithLabelValues("miss").Add(missCount)
		votePenaltyCount.WithLabelValues("abstain").Add(abstainCount)
		votePenaltyCount.WithLabelValues("success").Add(successCount)

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
