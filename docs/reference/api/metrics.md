# Metrics Package Public API

## Exported Types

```go
type Collector struct
type JMTMetrics interface
type NoOpMetrics struct{}
type PrometheusMetrics struct
type MetricsServer struct
```

## Exported Interfaces

```go
type JMTMetrics interface {
    RecordCommit(duration float64, batchSize int)
    RecordOperation(opType string)
    RecordLookup(duration float64, found bool)
    RecordProofGeneration(duration float64, proofType string, proofSize int)
    RecordProofValidationFailure()
    RecordHashOperation()
    RecordDBRead()
    RecordDBWrite()
    RecordError(errorType string)
    UpdateTreeHeight(height int)
    UpdateTreeNodeCount(count int)
    UpdateVersionCount(count int)
    UpdateDatabaseSize(bytes int64)
}
```

## Exported Functions

```go
func NewCollector(enabled bool) *Collector
func NewPrometheusMetrics() *PrometheusMetrics
func NewPrometheusMetricsWithRegistry(registerer prometheus.Registerer) *PrometheusMetrics
func NewMetricsServer(addr string, collector *Collector) *MetricsServer
func ServeMetricsHTTP(addr string, metricsEnabled bool) error
```

## Exported Methods

### Collector methods
```go
func (c *Collector) JMT() JMTMetrics
func (c *Collector) IsEnabled() bool
func (c *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request)
func (c *Collector) StartHTTPServer(addr string) error
```

### NoOpMetrics methods
```go
func (n NoOpMetrics) RecordCommit(duration float64, batchSize int)
func (n NoOpMetrics) RecordOperation(opType string)
func (n NoOpMetrics) RecordLookup(duration float64, found bool)
func (n NoOpMetrics) RecordProofGeneration(duration float64, proofType string, proofSize int)
func (n NoOpMetrics) RecordProofValidationFailure()
func (n NoOpMetrics) RecordHashOperation()
func (n NoOpMetrics) RecordDBRead()
func (n NoOpMetrics) RecordDBWrite()
func (n NoOpMetrics) RecordError(errorType string)
func (n NoOpMetrics) UpdateTreeHeight(height int)
func (n NoOpMetrics) UpdateTreeNodeCount(count int)
func (n NoOpMetrics) UpdateVersionCount(count int)
func (n NoOpMetrics) UpdateDatabaseSize(bytes int64)
```

### PrometheusMetrics methods
```go
func (m *PrometheusMetrics) RecordCommit(duration float64, batchSize int)
func (m *PrometheusMetrics) RecordOperation(opType string)
func (m *PrometheusMetrics) RecordLookup(duration float64, found bool)
func (m *PrometheusMetrics) RecordProofGeneration(duration float64, proofType string, proofSize int)
func (m *PrometheusMetrics) RecordProofValidationFailure()
func (m *PrometheusMetrics) RecordHashOperation()
func (m *PrometheusMetrics) RecordDBRead()
func (m *PrometheusMetrics) RecordDBWrite()
func (m *PrometheusMetrics) RecordError(errorType string)
func (m *PrometheusMetrics) UpdateTreeHeight(height int)
func (m *PrometheusMetrics) UpdateTreeNodeCount(count int)
func (m *PrometheusMetrics) UpdateVersionCount(count int)
func (m *PrometheusMetrics) UpdateDatabaseSize(bytes int64)
```

### MetricsServer methods
```go
func (s *MetricsServer) Start() error
func (s *MetricsServer) Stop(ctx context.Context) error
func (s *MetricsServer) GetAddr() string
func (s *MetricsServer) IsEnabled() bool
```