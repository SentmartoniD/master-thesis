param(
  [Parameter(Mandatory)][string]$Label,
  [int]$WindowSeconds = 75,
  [long]$EndEpoch = 0,
  [string]$Prometheus = "http://localhost:9090",
  [string]$OutFile = "results.csv"
)

if ($EndEpoch -eq 0) { $EndEpoch = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds() }
$W = "${WindowSeconds}s"
$S = $WindowSeconds

# Rewrites cAdvisor's "pod" label into a "job" label holding the deployment
# name, by stripping the ReplicaSet hash and pod suffix.
$ByService  = '"job", "$1", "pod", "^(.*)-[^-]+-[^-]+$"'
# k6 labels its metrics by scenario, not job. Map it so the client-side rows
# land in the same column as everything else.
$ByScenario = '"job", "$1", "scenario", "(.+)"'
$Cont = "namespace=`"redirect-application`", container!=`"`""

# metric name -> PromQL. $W is the window, $S the window in seconds.
$queries = [ordered]@{
  # ---- HTTP, per service
  "throughput_rps"        = "sum by (job) (increase(http_requests_total[$W])) / $S"
  "errors_rps"            = "(sum by (job) (increase(http_requests_total{status=~`"5..`"}[$W])) or sum by (job) (increase(http_requests_total[$W])) * 0) / $S"
  "error_ratio"           = "(sum by (job) (increase(http_requests_total{status=~`"5..`"}[$W])) or sum by (job) (increase(http_requests_total[$W])) * 0) / sum by (job) (increase(http_requests_total[$W]))"
  "latency_p50_s"         = "histogram_quantile(0.50, sum by (job, le) (rate(http_request_duration_seconds_bucket[$W])))"
  "latency_p95_s"         = "histogram_quantile(0.95, sum by (job, le) (rate(http_request_duration_seconds_bucket[$W])))"
  "latency_p99_s"         = "histogram_quantile(0.99, sum by (job, le) (rate(http_request_duration_seconds_bucket[$W])))"
  "in_flight_avg"         = "sum by (job) (avg_over_time(http_requests_in_flight[$W]))"
  "in_flight_max"         = "max by (job) (max_over_time(http_requests_in_flight[$W]))"

  # ---- client-side, pushed by k6. This is the capacity criterion:
  # the Prometheus histogram starts inside the gin middleware and excludes
  # kernel accept-queue time, which is where latency accumulates at saturation.
  "k6_latency_p95_s"      = "label_replace(max by (scenario) (max_over_time(k6_http_req_duration_p95[$W])), $ByScenario)"
  "k6_latency_p99_s"      = "label_replace(max by (scenario) (max_over_time(k6_http_req_duration_p99[$W])), $ByScenario)"
  "k6_latency_max_s"      = "label_replace(max by (scenario) (max_over_time(k6_http_req_duration_max[$W])), $ByScenario)"
  "k6_rps"                = "label_replace(sum by (scenario) (increase(k6_http_reqs_total[$W])), $ByScenario) / $S"
  "k6_failed_rate"        = "label_replace(max by (scenario) (max_over_time(k6_http_req_failed_rate[$W])), $ByScenario)"
  "k6_dropped_iterations" = "increase(k6_dropped_iterations_total[$W]) or vector(0)"
  "k6_vus_max"            = "max_over_time(k6_vus_max[$W])"

  # ---- database queries
  "db_query_rps"          = "sum by (job) (increase(db_queries_total[$W])) / $S"
  "db_query_errors"       = "sum by (job) (increase(db_queries_total{outcome=`"error`"}[$W])) or sum by (job) (increase(db_queries_total[$W])) * 0"
  "db_query_p95_s"        = "histogram_quantile(0.95, sum by (job, le) (rate(db_query_duration_seconds_bucket[$W])))"

  # ---- connection pool  (the core evidence)
  "pool_max_total"        = "sum by (job) (max_over_time(db_pool_max_conns[$W]))"
  "pool_max_per_pod"      = "avg by (job) (max_over_time(db_pool_max_conns[$W]))"
  "pool_acquired_total"   = "sum by (job) (avg_over_time(db_pool_acquired_conns[$W]))"
  "pool_acquired_per_pod" = "avg by (job) (avg_over_time(db_pool_acquired_conns[$W]))"
  "pool_acquired_max_pod" = "max by (job) (max_over_time(db_pool_acquired_conns[$W]))"
  "pool_utilisation_avg"  = "sum by (job) (avg_over_time(db_pool_acquired_conns[$W])) / sum by (job) (max_over_time(db_pool_max_conns[$W]))"
  "pool_empty_acquires"   = "sum by (job) (increase(db_pool_empty_acquire_total[$W]))"
  "pool_wait_total_s"     = "sum by (job) (increase(db_pool_empty_acquire_wait_seconds_total[$W]))"
  "pool_wait_mean_s"      = "sum by (job) (increase(db_pool_empty_acquire_wait_seconds_total[$W])) / sum by (job) (increase(db_pool_empty_acquire_total[$W]))"

  # ---- resources, from cAdvisor: covers both postgres pods, rabbitmq and k6
  "cpu_cores_total"       = "sum by (job) (label_replace(sum by (pod) (rate(container_cpu_usage_seconds_total{$Cont}[$W])), $ByService))"
  "cpu_cores_per_pod_avg" = "avg by (job) (label_replace(sum by (pod) (rate(container_cpu_usage_seconds_total{$Cont}[$W])), $ByService))"
  "cpu_cores_per_pod_max" = "max by (job) (label_replace(sum by (pod) (rate(container_cpu_usage_seconds_total{$Cont}[$W])), $ByService))"
  "memory_mb_total"       = "sum by (job) (label_replace(sum by (pod) (avg_over_time(container_memory_working_set_bytes{$Cont}[$W])), $ByService)) / 1024 / 1024"
  "memory_mb_per_pod_max" = "max by (job) (label_replace(sum by (pod) (max_over_time(container_memory_working_set_bytes{$Cont}[$W])), $ByService)) / 1024 / 1024"
  "goroutines_max"        = "max by (job) (max_over_time(go_goroutines[$W]))"

  # ---- per-pod, to verify the load balancer spread evenly
  "cpu_cores_by_pod"      = "sum by (pod) (rate(container_cpu_usage_seconds_total{$Cont}[$W]))"
  "rps_by_pod"            = "sum by (pod) (increase(http_requests_total[$W])) / $S"

  # ---- async path
  "clicks_published"      = "sum by (job) (increase(clicks_published_total[$W]))"
  "clicks_dropped"        = "sum by (job) (increase(clicks_dropped_total[$W]))"
  "clicks_consumed"       = "sum by (job) (increase(clicks_consumed_total{outcome=`"ok`"}[$W]))"
  "clicks_buffer_util"    = "avg by (job) (avg_over_time(clicks_buffer_used[$W])) / avg by (job) (max_over_time(clicks_buffer_capacity[$W]))"
  "queue_depth_avg"       = "avg_over_time(rabbitmq_queue_messages_ready[$W])"
  "queue_depth_max"       = "max_over_time(rabbitmq_queue_messages_ready[$W])"
  "flush_p95_s"           = "histogram_quantile(0.95, sum by (le) (rate(clicks_flush_duration_seconds_bucket[$W])))"

  # ---- replica count, recorded automatically
  "replicas"              = "count by (job) (up == 1)"
}

$rows = foreach ($name in $queries.Keys) {
  $uri = "$Prometheus/api/v1/query?query=$([uri]::EscapeDataString($queries[$name]))&time=$EndEpoch"
  try {
    $resp = Invoke-RestMethod -Uri $uri -TimeoutSec 20
  } catch {
    Write-Warning "$name : request failed"
    continue
  }
  if ($resp.status -ne 'success' -or $resp.data.result.Count -eq 0) {
    Write-Warning "$name : no data"
    continue
  }
  foreach ($series in $resp.data.result) {
    [pscustomobject]@{
      run        = $Label
      end_epoch  = $EndEpoch
      window_s   = $WindowSeconds
      metric     = $name
      job        = $(if ($series.metric.job) { $series.metric.job }
                     elseif ($series.metric.pod) { $series.metric.pod }
                     else { "-" })
      value      = [double]$series.value[1]
    }
  }
}

$exists = Test-Path $OutFile
$rows | Export-Csv -Path $OutFile -NoTypeInformation -Append:$exists
Write-Host "$Label : wrote $($rows.Count) rows to $OutFile"
