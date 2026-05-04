// Package metrics defines all application metrics and provides helpers for
// initialising and shutting down the OTel MeterProvider for batch (cron)
// runs. Metrics are exported directly to Google Cloud Monitoring via the
// GoogleCloud OTel exporter — no collector daemon is required.
//
// Each Run() call is a fresh process, so counters use delta temporality:
// every run reports its own increments rather than a lifetime total.
// Google Cloud Monitoring stitches delta samples into cumulative time-series.
package metrics

import (
	"context"
	"fmt"

	gcpexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const instrumentationName = "github.com/EpicMandM/esxi-lab-provider"

// Metrics holds all application instruments.
// Construct via New(); instruments are safe for concurrent use.
type Metrics struct {
	// Tier-1: business outcomes
	RunDuration          metric.Float64Histogram
	VMRestoreTotal       metric.Int64Counter
	VMRestoreDuration    metric.Float64Histogram
	CalendarEventsActive metric.Int64UpDownCounter
	EmailSendTotal       metric.Int64Counter
	PasswordRotateTotal  metric.Int64Counter

	// Tier-2: operational visibility
	WireGuardKeyRotateTotal metric.Int64Counter
	WireGuardPeerRegTotal   metric.Int64Counter
	CalendarFetchDuration   metric.Float64Histogram

	// Tier-3: inventory / nice-to-have
	VMInventoryTotal metric.Int64UpDownCounter
	RunTotal         metric.Int64Counter
}

// New creates all OTel instruments using the given meter.
func New(meter metric.Meter) (*Metrics, error) {
	var err error
	m := &Metrics{}

	// Tier-1
	if m.RunDuration, err = meter.Float64Histogram(
		"lab.run.duration",
		metric.WithDescription("Total duration of a single orchestrator run in seconds"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, fmt.Errorf("lab.run.duration: %w", err)
	}

	if m.VMRestoreTotal, err = meter.Int64Counter(
		"lab.vm.restore.total",
		metric.WithDescription("Number of VM snapshot restores attempted"),
	); err != nil {
		return nil, fmt.Errorf("lab.vm.restore.total: %w", err)
	}

	if m.VMRestoreDuration, err = meter.Float64Histogram(
		"lab.vm.restore.duration",
		metric.WithDescription("Time taken to restore a single VM snapshot in seconds"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, fmt.Errorf("lab.vm.restore.duration: %w", err)
	}

	if m.CalendarEventsActive, err = meter.Int64UpDownCounter(
		"lab.calendar.events.active",
		metric.WithDescription("Number of calendar events active at the time of the run"),
	); err != nil {
		return nil, fmt.Errorf("lab.calendar.events.active: %w", err)
	}

	if m.EmailSendTotal, err = meter.Int64Counter(
		"lab.email.send.total",
		metric.WithDescription("Number of credential emails sent"),
	); err != nil {
		return nil, fmt.Errorf("lab.email.send.total: %w", err)
	}

	if m.PasswordRotateTotal, err = meter.Int64Counter(
		"lab.password.rotation.total",
		metric.WithDescription("Number of ESXi user password rotations attempted"),
	); err != nil {
		return nil, fmt.Errorf("lab.password.rotation.total: %w", err)
	}

	// Tier-2
	if m.WireGuardKeyRotateTotal, err = meter.Int64Counter(
		"lab.wireguard.key.rotation.total",
		metric.WithDescription("Number of WireGuard key-pair rotations attempted"),
	); err != nil {
		return nil, fmt.Errorf("lab.wireguard.key.rotation.total: %w", err)
	}

	if m.WireGuardPeerRegTotal, err = meter.Int64Counter(
		"lab.wireguard.peer.registration.total",
		metric.WithDescription("Number of OPNsense WireGuard peer registrations attempted"),
	); err != nil {
		return nil, fmt.Errorf("lab.wireguard.peer.registration.total: %w", err)
	}

	if m.CalendarFetchDuration, err = meter.Float64Histogram(
		"lab.calendar.fetch.duration",
		metric.WithDescription("Time taken to fetch calendar events in seconds"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, fmt.Errorf("lab.calendar.fetch.duration: %w", err)
	}

	// Tier-3
	if m.VMInventoryTotal, err = meter.Int64UpDownCounter(
		"lab.vm.inventory.total",
		metric.WithDescription("Number of VMs found in the ESXi inventory"),
	); err != nil {
		return nil, fmt.Errorf("lab.vm.inventory.total: %w", err)
	}

	if m.RunTotal, err = meter.Int64Counter(
		"lab.run.total",
		metric.WithDescription("Total number of orchestrator runs"),
	); err != nil {
		return nil, fmt.Errorf("lab.run.total: %w", err)
	}

	return m, nil
}

// InitProvider initialises a Google Cloud Monitoring MeterProvider and
// registers it as the global OTel provider. Call the returned shutdown
// function (with a deadline context) before the process exits to flush
// all metrics.
//
// The GCP project and credentials are resolved automatically from
// Application Default Credentials (GOOGLE_APPLICATION_CREDENTIALS env var
// or the service account key deployed alongside the binary).
func InitProvider(ctx context.Context) (meter metric.Meter, shutdown func(context.Context) error, err error) {
	exporter, err := gcpexporter.New()
	if err != nil {
		return nil, nil, fmt.Errorf("create googlecloud exporter: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter),
		),
	)

	otel.SetMeterProvider(provider)

	meter = provider.Meter(instrumentationName)

	return meter, provider.Shutdown, nil
}

// Noop returns a no-op meter. Use in tests or when metrics are disabled.
func Noop() metric.Meter {
	return otel.GetMeterProvider().Meter(instrumentationName)
}
