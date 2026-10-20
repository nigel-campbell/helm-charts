package datadog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/DataDog/helm-charts/test/common"
)

func Test_ddotCollectorImageDigest(t *testing.T) {
	// Synthetic digests for rendering only; these are not pullable images.
	collectorDigest := "sha256:" + strings.Repeat("a", 64)
	agentDigest := "sha256:" + strings.Repeat("b", 64)

	tests := []struct {
		name         string
		overrides    map[string]string
		agentImage   string
		otelImage    string
		errorMessage string
	}{
		{
			name:       "standalone digest leaves agent tag unchanged",
			agentImage: "registry.datadoghq.com/agent:7.78.0",
			otelImage:  "registry.datadoghq.com/ddot-collector@" + collectorDigest,
		},
		{
			name: "independent agent and collector digests",
			overrides: map[string]string{
				"agents.image.digest": agentDigest,
			},
			agentImage: "registry.datadoghq.com/agent@" + agentDigest,
			otelImage:  "registry.datadoghq.com/ddot-collector@" + collectorDigest,
		},
		{
			name: "empty collector digest preserves derived tag",
			overrides: map[string]string{
				"datadog.otelCollector.image.digest": "",
				"agents.image.digest":                agentDigest,
			},
			agentImage: "registry.datadoghq.com/agent@" + agentDigest,
			otelImage:  "registry.datadoghq.com/ddot-collector:7.78.0",
		},
		{
			name: "custom registry",
			overrides: map[string]string{
				"registry": "registry.example.com/datadog",
			},
			agentImage: "registry.example.com/datadog/agent:7.78.0",
			otelImage:  "registry.example.com/datadog/ddot-collector@" + collectorDigest,
		},
		{
			name: "site specific registry",
			overrides: map[string]string{
				"datadog.site": "us3.datadoghq.com",
			},
			agentImage: "datadoghq.azurecr.io/agent:7.78.0",
			otelImage:  "datadoghq.azurecr.io/ddot-collector@" + collectorDigest,
		},
		{
			name: "FIPS digest has no tag suffix",
			overrides: map[string]string{
				"useFIPSAgent": "true",
			},
			agentImage: "registry.datadoghq.com/agent:7.78.0-fips",
			otelImage:  "registry.datadoghq.com/ddot-collector@" + collectorDigest,
		},
		{
			name: "non standalone collector uses agent digest",
			overrides: map[string]string{
				"datadog.otelCollector.useStandaloneImage": "false",
				"agents.image.tagSuffix":                   "full",
				"agents.image.digest":                      agentDigest,
			},
			agentImage: "registry.datadoghq.com/agent@" + agentDigest,
			otelImage:  "registry.datadoghq.com/agent@" + agentDigest,
		},
		{
			name: "legacy full image fallback ignores standalone digest",
			overrides: map[string]string{
				"agents.image.tag": "7.66.0-full",
			},
			agentImage: "registry.datadoghq.com/agent:7.66.0-full",
			otelImage:  "registry.datadoghq.com/agent:7.66.0-full",
		},
		{
			name: "digest does not bypass standalone version check",
			overrides: map[string]string{
				"agents.image.tag": "7.66.0",
			},
			errorMessage: "datadog.otelCollector.useStandaloneImage is only supported for agent versions 7.67.0+",
		},
		{
			name: "digest does not bypass FIPS version check",
			overrides: map[string]string{
				"agents.image.tag": "7.77.0",
				"useFIPSAgent":     "true",
			},
			errorMessage: "The standalone FIPS ddot-collector image is not available before 7.78.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides := map[string]string{
				"datadog.apiKeyExistingSecret":             "datadog-secret",
				"datadog.appKeyExistingSecret":             "datadog-secret",
				"datadog.otelCollector.enabled":            "true",
				"datadog.otelCollector.useStandaloneImage": "true",
				"datadog.otelCollector.image.digest":       collectorDigest,
				"agents.image.tag":                         "7.78.0",
			}
			for key, value := range tt.overrides {
				overrides[key] = value
			}
			manifest, err := common.RenderChart(t, common.HelmCommand{
				ReleaseName: "datadog",
				ChartPath:   "../../charts/datadog",
				ShowOnly:    []string{"templates/daemonset.yaml"},
				Values:      []string{"../../charts/datadog/values.yaml"},
				Overrides:   overrides,
			})
			if tt.errorMessage != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMessage)
				return
			}
			require.NoError(t, err)
			verifyAgentImage(t, manifest, tt.agentImage)
			verifyOtelImage(t, manifest, tt.otelImage)
		})
	}
}
