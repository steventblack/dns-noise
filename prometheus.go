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
		[]string{"type", "server"})

	dnsRespVec = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dns_noise_response",
		Help: "The total number of DNS records received"},
		[]string{"type", "rcode"})
)

func metricsDnsReq(label, server string) {
	dnsReqVec.WithLabelValues(label, server).Inc()
}

func metricsDnsResp(label, rcode string) {
	dnsRespVec.WithLabelValues(label, rcode).Inc()
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
