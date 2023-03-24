package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type votePenaltyCounter struct {
	missCount    int `json:"miss_count"`
	abstainCount int `json:"abstain_count"`
	successCount int `json:"success_count"`
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
			Name:        "vote_penalty_miss_count",
			Help:        "Vote penalty miss count",
			ConstLabels: ConstLabels,
		},
	)

	votePenaltySuccessCount := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:        "vote_penalty_miss_count",
			Help:        "Vote penalty miss count",
			ConstLabels: ConstLabels,
		},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(votePenaltyMissCount)
	registry.MustRegister(votePenaltyAbstainCount)
	registry.MustRegister(votePenaltySuccessCount)

	var wg sync.WaitGroup

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

		respBytes, _ := ioutil.ReadAll(response.Body)
		var respBody votePenaltyCounter
		json.Unmarshal(respBytes, &respBody)

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying oracle feeder metrics")

		votePenaltyMissCount(respBody.missCount)
		votePenaltyAbstainCount(respBody.abstainCount)
		votePenaltySuccessCount(respBody.successCount)

	}()
	wg.Add(1)
	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/sei").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}
