package util

import (
	"log"
	"time"

	"github.com/juicymango/yeah_woo_go/model"
)

var MetricsMap map[string]*model.Metrics

func GetMetrics(key string) *model.Metrics {
	if MetricsMap == nil {
		MetricsMap = make(map[string]*model.Metrics)
	}
	metrics := MetricsMap[key]
	if metrics != nil {
		return metrics
	}
	metrics = &model.Metrics{}
	MetricsMap[key] = metrics
	return metrics
}

func GetMetricsDeferFunc(metrics *model.Metrics) func() {
	startTime := time.Now()
	return func() {
		if metrics == nil {
			return
		}
		metrics.Count++
		metrics.TotalTime += time.Since(startTime)
	}
}

func GetMetricsOutput(metrics *model.Metrics) {
	metrics.TotalTimeMS = metrics.TotalTime.Milliseconds()
	if metrics.Count != 0 {
		metrics.AvgTimeMS = float64(metrics.TotalTimeMS) / float64(metrics.Count)
	}
}

func LogMetricsResult() {
	for key, metrics := range MetricsMap {
		GetMetricsOutput(metrics)
		log.Printf("LogMetricsResult key:%s, result:%s", key, JsonString(metrics))
	}
}
