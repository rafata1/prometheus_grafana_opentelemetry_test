package main

import (
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"sync/atomic"
)

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "service_request_total",
		Help: "The total number of request",
	})
)

type Service struct {
	Name     string
	EndPoint string
}

var serviceA = Service{
	Name:     "service-a",
	EndPoint: "http://service-a:8080/data",
}

var serviceB = Service{
	Name:     "service-b",
	EndPoint: "http://service-b:8081/data",
}

var RoundRobinCounter int32

func RouteReqToServices(w http.ResponseWriter, req *http.Request) {
	opsProcessed.Inc()
	atomic.AddInt32(&RoundRobinCounter, 1)
	if RoundRobinCounter%2 == 0 {
		err := redirectToService(serviceA)
		if err != nil {
			err := redirectToService(serviceB)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	} else {
		err := redirectToService(serviceB)
		if err != nil {
			err := redirectToService(serviceA)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}

func redirectToService(s Service) error {
	resp, err := http.Get(s.EndPoint)
	if err != nil || resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("error connecting to %s", s.EndPoint))
	}
	return nil
}

func main() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/data", RouteReqToServices)
	fmt.Sprintf("load balancer is running on port %s", os.Getenv("PORT"))
	http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil)
}
