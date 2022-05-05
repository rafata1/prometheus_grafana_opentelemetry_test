package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"loadBalancer/config"
	"loadBalancer/otellib"
	"math/rand"
	"net/http"
	"os"
	"sync/atomic"
	"time"
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
	newCtx, span := otel.GetTracerProvider().Tracer("RouteReqToServices").Start(req.Context(), "RouteReqToServices")
	defer span.End()
	latencies := []time.Duration{
		1 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}

	time.Sleep(latencies[rand.Int()%len(latencies)])

	opsProcessed.Inc()
	atomic.AddInt32(&RoundRobinCounter, 1)
	if RoundRobinCounter%2 == 0 {
		err := redirectToService(serviceA, newCtx)
		if err != nil {
			err := redirectToService(serviceB, newCtx)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	} else {
		err := redirectToService(serviceB, newCtx)
		if err != nil {
			err := redirectToService(serviceA, newCtx)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}

func redirectToService(s Service, ctx context.Context) error {
	resp, err := otelhttp.Get(ctx, s.EndPoint)
	if err != nil || resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("error connecting to %s", s.EndPoint))
	}
	return nil
}

func main() {
	tracerProvider, shutdown := otellib.InitOtel("LoadBalancer", "local", config.JaegerConfig{
		Host: "localhost",
		Port: 6831,
	})
	defer shutdown()

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	mux := &http.ServeMux{}

	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/data", otelhttp.NewHandler(http.HandlerFunc(RouteReqToServices), "RouteReqToServices"))
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8001"
	}

	fmt.Printf("service is runing on port %s", port)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
