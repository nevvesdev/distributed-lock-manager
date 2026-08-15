package metrics

import "github.com/prometheus/client_golang/prometheus"

// agrupa todas as métricas Prometheus do distributed-lock-manager.
type Metrics struct {
	AcquireTotal    *prometheus.CounterVec
	ReleaseTotal    *prometheus.CounterVec
	RenewTotal      *prometheus.CounterVec
	ContentionTotal prometheus.Counter
	HoldDuration    prometheus.Histogram
}

// registra e retorna as métricas no registry padrão do Prometheus.
func New() *Metrics {
	m := &Metrics{
		AcquireTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dlm_lock_acquire_total",
				Help: "Total de tentativas de acquire, separadas por resultado.",
			},
			[]string{"result"}, // success | failed | error
		),
		ReleaseTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dlm_lock_release_total",
				Help: "Total de tentativas de release, separadas por resultado.",
			},
			[]string{"result"}, // success | not_found | not_owned | token_mismatch | error
		),
		RenewTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dlm_lock_renew_total",
				Help: "Total de renovações de lock, separadas por resultado.",
			},
			[]string{"result"}, // success | not_found | not_owned | token_mismatch | error
		),
		ContentionTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "dlm_lock_contention_total",
				Help: "Total de vezes que um acquire falhou porque o lock já estava ocupado.",
			},
		),
		HoldDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "dlm_lock_hold_duration_seconds",
				Help:    "Duração em segundos que um lock foi mantido até o release.",
				Buckets: prometheus.DefBuckets,
			},
		),
	}

	prometheus.MustRegister(
		m.AcquireTotal,
		m.ReleaseTotal,
		m.RenewTotal,
		m.ContentionTotal,
		m.HoldDuration,
	)

	return m
}
