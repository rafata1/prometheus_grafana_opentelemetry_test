package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"log"
	"math/rand"
	"net/http"
	"os"
	"serviceA/config"
	"serviceA/otellib"
	"time"
)

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "service_request_total",
		Help: "The total number of request",
	})
)

var serviceC = "http://service-b-master:8083/Data"

func GetDataHandler(w http.ResponseWriter, req *http.Request) {
	ctx, span := otel.GetTracerProvider().Tracer("GetDataHandler").Start(req.Context(), "CallServiceB")
	defer span.End()
	latencies := []time.Duration{
		1 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}
	time.Sleep(latencies[rand.Int()%len(latencies)])

	_, err := otelhttp.Get(ctx, serviceC)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	opsProcessed.Inc()
	log.Println("get data")
	w.WriteHeader(http.StatusOK)
}

func main() {
	serviceName := os.Getenv("NAME")
	if len(serviceName) == 0 {
		serviceName = "DefaultService"
	}

	tracerProvider, shutdown := otellib.InitOtel(serviceName, "local", config.JaegerConfig{
		Host: "jaeger",
		Port: 6831,
	})
	defer shutdown()

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	mux := &http.ServeMux{}

	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/data", otelhttp.NewHandler(http.HandlerFunc(GetDataHandler), "GetDataHandler"))
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
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
