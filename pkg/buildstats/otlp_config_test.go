package buildstats

import "testing"

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
		t := tt
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
		{otelExporterOTLPTracesEndpoint: "tempo:4318"},
		{otelExporterOTLPEndpoint: "file:///tmp/collector"},
		{otelExporterOTLPTracesEndpoint: "http://tempo:4318/v1/traces", otelExporterOTLPTracesProtocol: "udp"},
	} {
		env := env
		if _, err := ResolveOTLPConfig(func(key string) string { return env[key] }); err == nil {
			t.Fatalf("ResolveOTLPConfig(%v) unexpectedly succeeded", env)
		}
	}
}
