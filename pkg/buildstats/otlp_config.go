package buildstats

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	otelExporterOTLPEndpoint       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	otelExporterOTLPTracesEndpoint = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	otelExporterOTLPProtocol       = "OTEL_EXPORTER_OTLP_PROTOCOL"
	otelExporterOTLPTracesProtocol = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"
	defaultOTLPProtocol            = "http/protobuf"
)

// OTLPConfig describes optional OpenTelemetry trace export configuration.
// Export remains disabled unless an OTLP endpoint is explicitly configured.
type OTLPConfig struct {
	Endpoint string
	Protocol string
}

// Enabled reports whether trace export was explicitly configured.
func (c OTLPConfig) Enabled() bool {
	return c.Endpoint != ""
}

// ResolveOTLPConfig applies the standard OTEL environment-variable precedence
// for trace exporters. The signal-specific traces endpoint is used as-is. A
// generic OTLP endpoint receives the standard /v1/traces suffix.
func ResolveOTLPConfig(getenv func(string) string) (OTLPConfig, error) {
	if getenv == nil {
		return OTLPConfig{}, nil
	}

	protocol := strings.TrimSpace(getenv(otelExporterOTLPTracesProtocol))
	if protocol == "" {
		protocol = strings.TrimSpace(getenv(otelExporterOTLPProtocol))
	}
	if protocol == "" {
		protocol = defaultOTLPProtocol
	}
	if err := validateOTLPProtocol(protocol); err != nil {
		return OTLPConfig{}, err
	}

	tracesEndpoint := strings.TrimSpace(getenv(otelExporterOTLPTracesEndpoint))
	if tracesEndpoint != "" {
		endpoint, err := validateOTLPEndpoint(tracesEndpoint)
		if err != nil {
			return OTLPConfig{}, fmt.Errorf("%s: %w", otelExporterOTLPTracesEndpoint, err)
		}
		return OTLPConfig{Endpoint: endpoint, Protocol: protocol}, nil
	}

	genericEndpoint := strings.TrimSpace(getenv(otelExporterOTLPEndpoint))
	if genericEndpoint == "" {
		return OTLPConfig{Protocol: protocol}, nil
	}
	endpoint, err := appendOTLPTracePath(genericEndpoint)
	if err != nil {
		return OTLPConfig{}, fmt.Errorf("%s: %w", otelExporterOTLPEndpoint, err)
	}
	return OTLPConfig{Endpoint: endpoint, Protocol: protocol}, nil
}

func validateOTLPProtocol(protocol string) error {
	switch protocol {
	case "grpc", "http/protobuf", "http/json":
		return nil
	default:
		return fmt.Errorf("unsupported OTLP protocol %q", protocol)
	}
}

func validateOTLPEndpoint(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("endpoint must use http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("endpoint host is required")
	}
	return parsed.String(), nil
}

func appendOTLPTracePath(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("endpoint must use http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("endpoint host is required")
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/v1/traces"
	return parsed.String(), nil
}
