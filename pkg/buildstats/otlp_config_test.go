package buildstats

import (
	"strings"
	"testing"
)

func TestResolveOTLPConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      map[string]string
		endpoint string
		protocol string
		enabled  bool
	}{
		{
			name:     "disabled by default",
			protocol: defaultOTLPProtocol,
		},
		{
			name: "generic HTTP endpoint appends traces path",
			env: map[string]string{
				otelExporterOTLPEndpoint: "http://tempo:4318/collector/",
			},
			endpoint: "http://tempo:4318/collector/v1/traces",
			protocol: defaultOTLPProtocol,
			enabled:  true,
		},
		{
			name: "HTTP JSON endpoint preserves query",
			env: map[string]string{
				otelExporterOTLPEndpoint: "https://collector/base?tenant=example",
				otelExporterOTLPProtocol: "http/json",
			},
			endpoint: "https://collector/base/v1/traces?tenant=example",
			protocol: "http/json",
			enabled:  true,
		},
		{
			name: "generic gRPC endpoint stays unmodified",
			env: map[string]string{
				otelExporterOTLPEndpoint: "http://tempo:4317/collector",
				otelExporterOTLPProtocol: "grpc",
			},
			endpoint: "http://tempo:4317/collector",
			protocol: "grpc",
			enabled:  true,
		},
		{
			name: "traces endpoint wins and is used as is",
			env: map[string]string{
				otelExporterOTLPEndpoint:       "http://generic:4318",
				otelExporterOTLPTracesEndpoint: "https://traces.example.test/custom",
				otelExporterOTLPProtocol:       "grpc",
				otelExporterOTLPTracesProtocol: "http/json",
			},
			endpoint: "https://traces.example.test/custom",
			protocol: "http/json",
			enabled:  true,
		},
		{
			name: "generic protocol used when traces protocol is absent",
			env: map[string]string{
				otelExporterOTLPTracesEndpoint: "http://tempo:4318/v1/traces",
				otelExporterOTLPProtocol:       "grpc",
			},
			endpoint: "http://tempo:4318/v1/traces",
			protocol: "grpc",
			enabled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getenv := func(key string) string { return tt.env[key] }
			got, err := ResolveOTLPConfig(getenv)
			if err != nil {
				t.Fatalf("ResolveOTLPConfig: %v", err)
			}
			if got.Endpoint != tt.endpoint || got.Protocol != tt.protocol || got.Enabled() != tt.enabled {
				t.Fatalf("config = %#v, enabled=%v; want endpoint=%q protocol=%q enabled=%v", got, got.Enabled(), tt.endpoint, tt.protocol, tt.enabled)
			}
		})
	}
}

func TestResolveOTLPConfigRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, env := range []map[string]string{
		{otelExporterOTLPProtocol: "udp"},
		{otelExporterOTLPTracesEndpoint: "tempo:4318"},
		{otelExporterOTLPEndpoint: "file:///tmp/collector"},
		{otelExporterOTLPTracesEndpoint: "http://tempo:4318/v1/traces", otelExporterOTLPTracesProtocol: "udp"},
	} {
		if _, err := ResolveOTLPConfig(func(key string) string { return env[key] }); err == nil {
			t.Fatalf("ResolveOTLPConfig(%v) unexpectedly succeeded", env)
		}
	}
}

func TestResolveOTLPConfigDoesNotLeakMalformedEndpoint(t *testing.T) {
	t.Parallel()
	for _, key := range []string{otelExporterOTLPEndpoint, otelExporterOTLPTracesEndpoint} {
		_, err := ResolveOTLPConfig(func(name string) string {
			if name == key {
				return "http://user:private-password@collector/%zz?token=private-token"
			}
			return ""
		})
		if err == nil {
			t.Fatal("malformed endpoint accepted")
		}
		if strings.Contains(err.Error(), "private-") {
			t.Fatalf("endpoint credentials leaked: %s", err)
		}
	}
}

func TestResolveOTLPConfigNilReader(t *testing.T) {
	t.Parallel()
	got, err := ResolveOTLPConfig(nil)
	if err != nil || got.Enabled() || got != (OTLPConfig{}) {
		t.Fatalf("nil reader = %#v, %v", got, err)
	}
}
