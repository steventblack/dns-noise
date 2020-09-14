//
// Copyright 2020 Steven T Black
//

package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
)

var (
	dnsReqVec = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dns_noise_request",
		Help: "The total number of DNS requests issued."},
		[]string{"type", "server", "rcode"})

	dnsRespVec = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dns_noise_response",
		Help: "The total number of DNS records received."},
		[]string{"type", "rcode", "server"})

	dnsRespTimeVec = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dns_noise_responsetime",
		Help:    "The response times for DNS queries.",
		Buckets: prometheus.LinearBuckets(50, 50, 15)},
		[]string{"type", "server"})
)

func metricsDnsReq(label, server, rcode string) {
	dnsReqVec.WithLabelValues(label, server).Inc()
}

func metricsDnsResp(label, server, rcode string) {
	dnsRespVec.WithLabelValues(label, rcode, server).Inc()
}

func metricsDnsRespTime(dur float64, label, server string) {
	dnsRespTimeVec.WithLabelValues(label, server).Observe(dur)
}

func metricsConfig(conf *Metrics) {
	if conf == nil {
		log.Println("Metrics not configured; omitting")
		return
	}

	if conf.Enabled == false {
		log.Println("Metrics disabled; omitting")
		return
	}

	http.Handle(conf.Path, promhttp.Handler())
	port := ":" + strconv.Itoa(conf.Port)

	go func() {
		http.ListenAndServe(port, nil)
	}()
}
