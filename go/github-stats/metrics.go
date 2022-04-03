package github_stats

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const MetricsNamespace = "github_stats"

var (
	PRMeanResolutionTimeSecs = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "pr_mean_resolution_time_seconds",
	})

	PRReviewsCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "pr_reviews_count",
	}, []string{
		"username",
	})
)
