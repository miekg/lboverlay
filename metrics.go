package lboverlay

import (
	"github.com/coredns/coredns/plugin"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	hcCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "lboverlay",
		Name:      "healthcheck_total",
		Help:      "Total number of health checks successfully applied.",
	}, []string{"server"})
)
