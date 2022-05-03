package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
)

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "service_request_total",
		Help: "The total number of request",
	})
)

func GetDataHandler(w http.ResponseWriter, req *http.Request) {
	opsProcessed.Inc()
	log.Println("get data")
	w.WriteHeader(http.StatusOK)
}

func main() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/data", GetDataHandler)
	fmt.Printf("service is runing on port %s", os.Getenv("PORT"))
	http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil)
}
