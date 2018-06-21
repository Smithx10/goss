package outputs

import (
	"io"
	"time"

	"github.com/aelsabbahy/goss/resource"
	"github.com/aelsabbahy/goss/util"
	"github.com/prometheus/client_golang/prometheus"
)

type Prom struct{}

func (p Prom) Output(w io.Writer, results <-chan []resource.TestResult,
	startTime time.Time, outConfig util.OutputConfig) (exitCode int) {

	for resultGroup := range results {
		for _, testResult := range resultGroup {
			var setValue float64 = 0
			if testResult.Successful == false {
				setValue = float64(1)
			}

			gossGauge.With(prometheus.Labels{
				"resource_type": testResult.ResourceType,
				"resource_id":   testResult.ResourceId,
				"property":      testResult.Property,
				"title":         testResult.Title,
			}).Set(setValue)
		}
	}

	return 0
}

var (
	gossGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "goss",
		Help: "Lets you know if goss assertions were true 0, or false 1"},
		[]string{"resource_type", "resource_id", "property", "title"},
	)
)

func init() {
	prometheus.MustRegister(gossGauge)
	RegisterOutputer("prometheus", &Prom{}, []string{})
}
