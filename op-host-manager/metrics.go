package ophostmanager

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	MetricsNamespace = "op_host_manager"
)

var (
	labelUpdates = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "label_updates",
		Help:      "total number of label updates",
	}, []string{
		"foo",
	})
)

func IncrementLabelUpdates(value string) {
	labelUpdates.WithLabelValues(value).Inc()
}
