package engine

import (
	"container/heap"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"moktopus/simulation-service/internal/model"
)

const (
	eventRequestReceived = "REQUEST_RECEIVED"
	eventRequestRouted   = "REQUEST_ROUTED"
	eventProcessStart    = "PROCESS_START"
	eventProcessComplete = "PROCESS_COMPLETE"
	eventQueued          = "QUEUED"
	eventFailed          = "FAILED"
	eventRequestFailed   = "REQUEST_FAILED"
	eventRetry           = "RETRY"
	eventRequestComplete = "REQUEST_COMPLETE"
	eventTaxonomyVersion = "v1"
	eventCircuitOpen     = "CIRCUIT_OPEN"
	eventCircuitHalfOpen = "CIRCUIT_HALF_OPEN"
	eventCircuitClosed   = "CIRCUIT_CLOSED"
	eventAutoscaleUp     = "AUTOSCALE_UP"
	eventAutoscaleDown   = "AUTOSCALE_DOWN"
)

type Engine interface {
	Run(req model.RunRequest) (model.RunResponse, error)
	RunWithProgress(req model.RunRequest, onProgress func(model.ProgressSnapshot)) (model.RunResponse, error)
}

type simulationEngine struct{}

type randomizer struct {
	enabled bool
	rng     *rand.Rand

	deterministic bool
	seedUsed      int64

	arrivalJitter   float64
	latencyJitter   float64
	failureVariance float64
}

func New() Engine {
	return simulationEngine{}
}

func (e simulationEngine) Run(req model.RunRequest) (model.RunResponse, error) {
	return e.RunWithProgress(req, nil)
}

type requestState struct {
	id          int
	path        []string
	pathIndex   int
	startMs     float64
	queueWaitMs float64
	retries     int
	done        bool
	failed      bool
	fanoutWait  int // count of outstanding fan-out children
}

type queueState struct {
	availableAt []float64 // one slot per concurrent worker (M/M/c)
	serviceMs   float64
	maxDepth    int
	concurrency int
}

// circuitBreakerState models the circuit breaker pattern per node.
type circuitBreakerState struct {
	state               string // "closed", "open", "half-open"
	consecutiveFailures int
	lastFailureTimeMs   float64
	halfOpenAttempts    int
}

// autoscaleState tracks per-node replica scaling during simulation.
type autoscaleState struct {
	currentReplicas int
	lastScaleTimeMs float64
	cpuAccumulator  float64
	cpuSamples      int
}

// connectionPoolState tracks per-node connection pool usage.
type connectionPoolState struct {
	active         int     // currently in-use connections
	maxConnections int     // pool limit
	acquireTimeout float64 // ms before acquire fails
	leaked         int     // connections lost to leaks
}

// brownoutState tracks per-node load degradation.
type brownoutState struct {
	loadFactor       float64 // current load factor (0-1)
	activeSince      float64 // timeMs when brownout started
	requestsInWindow int     // requests processed in last window
	windowStartMs    float64 // start of current measurement window
}

type internalEvent struct {
	timeMs    float64
	seq       int
	eventType string
	requestID int
	nodeID    string
}

type eventQueue []internalEvent

func (q eventQueue) Len() int { return len(q) }
func (q eventQueue) Less(i, j int) bool {
	if q[i].timeMs == q[j].timeMs {
		return q[i].seq < q[j].seq
	}
	return q[i].timeMs < q[j].timeMs
}
func (q eventQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *eventQueue) Push(x any)   { *q = append(*q, x.(internalEvent)) }
func (q *eventQueue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	*q = old[:n-1]
	return item
}

func (simulationEngine) RunWithProgress(req model.RunRequest, onProgress func(model.ProgressSnapshot)) (model.RunResponse, error) {
	if len(req.Topology.Nodes) == 0 {
		return model.RunResponse{}, errors.New("topology requires at least one node")
	}
	if req.Traffic.DurationSeconds <= 0 {
		req.Traffic.DurationSeconds = 60
	}
	if req.Traffic.RequestsPerSecond <= 0 {
		req.Traffic.RequestsPerSecond = 25
	}
	if req.Traffic.BaseNetworkLatencyMs <= 0 {
		req.Traffic.BaseNetworkLatencyMs = 8
	}
	if req.Traffic.RetryPenaltyMs <= 0 {
		req.Traffic.RetryPenaltyMs = 20
	}
	if req.Traffic.MaxRetries < 0 {
		req.Traffic.MaxRetries = 0
	}
	randomness := buildRandomizer(req)

	nodesByID := make(map[string]model.TopologyNode, len(req.Topology.Nodes))
	nextByID := make(map[string][]string)
	for _, node := range req.Topology.Nodes {
		nodesByID[node.ID] = node
	}
	for _, edge := range req.Topology.Edges {
		nextByID[edge.Source] = append(nextByID[edge.Source], edge.Target)
	}

	path, err := derivePath(req.Topology.Nodes, nextByID)
	if err != nil {
		return model.RunResponse{}, err
	}

	requestsTotal := req.Traffic.DurationSeconds * req.Traffic.RequestsPerSecond
	if requestsTotal <= 0 {
		requestsTotal = 1
	}

	states := make(map[int]*requestState, requestsTotal)
	queueStates := make(map[string]*queueState)
	circuitBreakers := make(map[string]*circuitBreakerState)
	autoscaleStates := make(map[string]*autoscaleState)
	connPoolStates := make(map[string]*connectionPoolState)
	brownoutStates := make(map[string]*brownoutState)
	nodeStats := make(map[string]model.NodeRuntimeStats, len(req.Topology.Nodes))
	for _, node := range req.Topology.Nodes {
		nodeStats[node.ID] = model.NodeRuntimeStats{}
	}
	latencies := make([]float64, 0, requestsTotal)
	events := make([]model.SimulationEvent, 0, requestsTotal*6)

	var processingMs float64
	var maxQueueDepth int
	var totalQueueWait float64
	var circuitBreakerTrips int
	var autoscaleEvents int
	var connPoolExhausted int
	var backpressureShed int
	var brownoutActivations int
	var peakLoadFactor float64
	completed := 0
	failed := 0
	reportEvery := maxInt(1, requestsTotal/12)
	lastReportedResolved := 0

	pq := &eventQueue{}
	heap.Init(pq)
	seq := 0
	ratePerMs := float64(req.Traffic.RequestsPerSecond) / 1000.0
	if ratePerMs <= 0 {
		ratePerMs = 0.001
	}
	arrivalGapMs := 1 / ratePerMs

	for i := 0; i < requestsTotal; i++ {
		var arrival float64
		if randomness.enabled && i > 0 {
			// True Poisson process: exponential inter-arrival times
			u := randomness.nextFloat64()
			if u <= 0 {
				u = 0.0001
			}
			interArrival := -arrivalGapMs * math.Log(u) // exponential distribution
			if i == 1 {
				arrival = interArrival
			} else {
				arrival = states[i-1].startMs + interArrival
			}
		} else {
			arrival = float64(i) * arrivalGapMs
		}
		states[i] = &requestState{id: i, path: path, startMs: arrival}
		heap.Push(pq, internalEvent{timeMs: arrival, seq: seq, eventType: eventRequestReceived, requestID: i})
		seq++
	}

	for pq.Len() > 0 {
		ev := heap.Pop(pq).(internalEvent)
		state := states[ev.requestID]
		if state == nil || state.done {
			continue
		}

		events = appendEvent(events, ev.timeMs, ev.eventType, ev.requestID, ev.nodeID, "")
		updateNodeStats(nodeStats, ev.nodeID, ev.eventType)

		switch ev.eventType {
		case eventRequestReceived:
			firstNode := state.path[0]
			networkLatency := req.Traffic.BaseNetworkLatencyMs
			// Add service mesh sidecar overhead on ingress
			if req.Traffic.ServiceMesh != nil && req.Traffic.ServiceMesh.Enabled {
				networkLatency += req.Traffic.ServiceMesh.SidecarLatencyMs
				networkLatency += req.Traffic.ServiceMesh.MTLSOverheadMs
			}
			heap.Push(pq, internalEvent{
				timeMs:    ev.timeMs + networkLatency,
				seq:       seq,
				eventType: eventRequestRouted,
				requestID: ev.requestID,
				nodeID:    firstNode,
			})
			seq++

		case eventRequestRouted:
			node := nodesByID[ev.nodeID]
			if strings.EqualFold(node.Type, "queue") {
				qs := ensureQueueState(queueStates, node)
				slotIdx, slotAvail := nextAvailableSlot(qs)
				startAt := ev.timeMs
				if ev.timeMs < slotAvail {
					wait := slotAvail - ev.timeMs
					state.queueWaitMs += wait
					totalQueueWait += wait
					// Estimate depth: how many requests are queued across all workers
					depthEstimate := 0
					for _, slot := range qs.availableAt {
						if slot > ev.timeMs {
							depthEstimate += int(math.Ceil((slot - ev.timeMs) / qs.serviceMs))
						}
					}
					if depthEstimate > maxQueueDepth {
						maxQueueDepth = depthEstimate
					}
					if qs.maxDepth > 0 && depthEstimate >= qs.maxDepth {
						handleFailure(state, req, ev, pq, &seq, &failed, &events, "queue depth exceeded")
						continue
					}
					startAt = slotAvail
					events = appendEvent(events, ev.timeMs, eventQueued, ev.requestID, ev.nodeID, fmt.Sprintf("wait=%.2fms depth=%d", wait, depthEstimate))
				}

				qs.availableAt[slotIdx] = startAt + qs.serviceMs
				heap.Push(pq, internalEvent{
					timeMs:    startAt,
					seq:       seq,
					eventType: eventProcessStart,
					requestID: ev.requestID,
					nodeID:    ev.nodeID,
				})
				seq++
				continue
			}

			heap.Push(pq, internalEvent{timeMs: ev.timeMs, seq: seq, eventType: eventProcessStart, requestID: ev.requestID, nodeID: ev.nodeID})
			seq++

		case eventProcessStart:
			node := nodesByID[ev.nodeID]

			// Circuit breaker check: if open, reject immediately.
			if req.Traffic.CircuitBreaker != nil && req.Traffic.CircuitBreaker.Enabled {
				cb := ensureCircuitBreaker(circuitBreakers, ev.nodeID)
				switch cb.state {
				case "open":
					// Check if recovery time has elapsed → half-open
					if ev.timeMs-cb.lastFailureTimeMs >= req.Traffic.CircuitBreaker.RecoveryTimeMs {
						cb.state = "half-open"
						cb.halfOpenAttempts = 0
						events = appendEvent(events, ev.timeMs, eventCircuitHalfOpen, ev.requestID, ev.nodeID, "recovery elapsed")
					} else {
						// Circuit is open: fast-fail this request
						handleFailure(state, req, ev, pq, &seq, &failed, &events, "circuit breaker open")
						continue
					}
				case "half-open":
					// Allow limited requests through
					if cb.halfOpenAttempts >= req.Traffic.CircuitBreaker.HalfOpenRequests {
						handleFailure(state, req, ev, pq, &seq, &failed, &events, "circuit breaker half-open limit")
						continue
					}
					cb.halfOpenAttempts++
				}
			}

			// Connection pool exhaustion check
			if req.Traffic.ConnectionPool != nil && req.Traffic.ConnectionPool.Enabled {
				pool := ensureConnPool(connPoolStates, ev.nodeID, req.Traffic.ConnectionPool)
				// Simulate connection leak
				if req.Traffic.ConnectionPool.LeakRate > 0 && randomness.enabled {
					if randomness.nextFloat64() < req.Traffic.ConnectionPool.LeakRate {
						pool.leaked++
					}
				}
				effectiveMax := pool.maxConnections - pool.leaked
				if effectiveMax < 1 {
					effectiveMax = 1
				}
				if pool.active >= effectiveMax {
					// Pool exhausted: fail with timeout
					connPoolExhausted++
					handleFailure(state, req, ev, pq, &seq, &failed, &events, fmt.Sprintf("connection pool exhausted (%d/%d, %d leaked)", pool.active, pool.maxConnections, pool.leaked))
					continue
				}
				pool.active++
			}

			// Backpressure: load shedding when overloaded
			if req.Traffic.Backpressure != nil && req.Traffic.Backpressure.Enabled {
				loadFactor := computeLoadFactor(queueStates, ev.nodeID, ev.timeMs)
				if loadFactor > peakLoadFactor {
					peakLoadFactor = loadFactor
				}
				if loadFactor >= req.Traffic.Backpressure.ShedThreshold {
					shedProb := req.Traffic.Backpressure.ShedRate * (loadFactor - req.Traffic.Backpressure.ShedThreshold) / (1 - req.Traffic.Backpressure.ShedThreshold)
					if randomness.enabled && randomness.nextFloat64() < shedProb {
						backpressureShed++
						handleFailure(state, req, ev, pq, &seq, &failed, &events, fmt.Sprintf("backpressure shed (load=%.2f)", loadFactor))
						continue
					}
				}
			}

			procMs := processingTime(node, randomness)

			// Capacity-aware load-dependent latency degradation:
			// As node approaches saturation (based on instance profile), latency increases.
			profile := ResolveProfile(node)
			if profile.MaxRPS > 0 {
				// Use active connection pool count as concurrency estimate
				currentConcurrency := 0
				if pool, ok := connPoolStates[ev.nodeID]; ok {
					currentConcurrency = pool.active
				}
				replicas := 1
				if as, ok := autoscaleStates[ev.nodeID]; ok {
					replicas = as.currentReplicas
				} else if v, ok := getNumber(node.Config, "replicas"); ok && v > 0 {
					replicas = int(v)
				}
				if currentConcurrency > 0 {
					procMs = LoadDependentLatency(procMs, currentConcurrency, profile.MaxRPS, replicas)
				}
			}

			// Apply chaos slowdown multiplier if active.
			procMs *= chaosLatencyMultiplier(req.Chaos, ev.nodeID, ev.timeMs)

			// Brownout: gradual degradation under sustained load
			if req.Traffic.Brownout != nil && req.Traffic.Brownout.Enabled {
				bs := ensureBrownoutState(brownoutStates, ev.nodeID, ev.timeMs)
				bs.requestsInWindow++
				// Update load factor based on throughput vs capacity
				loadFactor := computeLoadFactor(queueStates, ev.nodeID, ev.timeMs)
				bs.loadFactor = loadFactor

				if loadFactor >= req.Traffic.Brownout.LoadThreshold {
					if bs.activeSince == 0 {
						bs.activeSince = ev.timeMs
						brownoutActivations++
					}
					// Progressive slowdown: linearly scale from threshold to max
					progress := (loadFactor - req.Traffic.Brownout.LoadThreshold) / (1 - req.Traffic.Brownout.LoadThreshold)
					if progress > 1 {
						progress = 1
					}
					slowdownMult := 1.0 + (req.Traffic.Brownout.MaxSlowdownPct/100)*progress
					procMs *= slowdownMult

					// Additional error rate during brownout
					if req.Traffic.Brownout.ErrorRateAtPeak > 0 && randomness.enabled {
						additionalErrorRate := req.Traffic.Brownout.ErrorRateAtPeak * progress
						if randomness.nextFloat64() < additionalErrorRate {
							handleFailure(state, req, ev, pq, &seq, &failed, &events, fmt.Sprintf("brownout failure (load=%.2f)", loadFactor))
							continue
						}
					}
				} else {
					bs.activeSince = 0
				}
			}

			processingMs += procMs

			// Autoscaling: track CPU and adjust replicas dynamically.
			if req.Traffic.Autoscaling != nil && req.Traffic.Autoscaling.Enabled {
				as := ensureAutoscaleState(autoscaleStates, ev.nodeID, queueStates)
				as.cpuAccumulator += procMs
				as.cpuSamples++

				// Compute real CPU utilization based on instance profile capacity
				nodeProfile := ResolveProfile(node)
				var instantCPU float64
				if nodeProfile.VCPUs > 0 {
					// Real CPU: concurrent processing vs available CPU cores
					concurrentEstimate := 0
					if pool, ok := connPoolStates[ev.nodeID]; ok {
						concurrentEstimate = pool.active
					} else if qs, hasQueue := queueStates[ev.nodeID]; hasQueue {
						for _, slot := range qs.availableAt {
							if slot > ev.timeMs {
								concurrentEstimate++
							}
						}
					}
					if concurrentEstimate < 1 {
						concurrentEstimate = 1
					}
					instantCPU = ComputeCPUUtilization(concurrentEstimate, procMs, nodeProfile.VCPUs, as.currentReplicas) * 100
				} else {
					instantCPU = (procMs / (30 * float64(as.currentReplicas))) * 100
				}

				// Check cooldown
				if ev.timeMs-as.lastScaleTimeMs >= req.Traffic.Autoscaling.CooldownMs {
					if instantCPU > req.Traffic.Autoscaling.ScaleUpThreshold*100 && as.currentReplicas < req.Traffic.Autoscaling.MaxReplicas {
						// Scale up: add replica after delay
						as.currentReplicas++
						as.lastScaleTimeMs = ev.timeMs + req.Traffic.Autoscaling.ScaleUpDelayMs
						autoscaleEvents++
						events = appendEvent(events, ev.timeMs, eventAutoscaleUp, ev.requestID, ev.nodeID, fmt.Sprintf("replicas=%d", as.currentReplicas))
						// Expand queue concurrency
						if qs, hasQueue := queueStates[ev.nodeID]; hasQueue {
							qs.availableAt = append(qs.availableAt, ev.timeMs+req.Traffic.Autoscaling.ScaleUpDelayMs)
							qs.concurrency = as.currentReplicas
						}
					} else if instantCPU < req.Traffic.Autoscaling.ScaleDownThreshold*100 && as.currentReplicas > 1 {
						// Scale down
						as.currentReplicas--
						as.lastScaleTimeMs = ev.timeMs
						autoscaleEvents++
						events = appendEvent(events, ev.timeMs, eventAutoscaleDown, ev.requestID, ev.nodeID, fmt.Sprintf("replicas=%d", as.currentReplicas))
						if qs, hasQueue := queueStates[ev.nodeID]; hasQueue && len(qs.availableAt) > 1 {
							qs.availableAt = qs.availableAt[:len(qs.availableAt)-1]
							qs.concurrency = as.currentReplicas
						}
					}
				}
			}

			heap.Push(pq, internalEvent{
				timeMs:    ev.timeMs + procMs,
				seq:       seq,
				eventType: eventProcessComplete,
				requestID: ev.requestID,
				nodeID:    ev.nodeID,
			})
			seq++

		case eventProcessComplete:
			node := nodesByID[ev.nodeID]

			// Release connection back to pool
			if req.Traffic.ConnectionPool != nil && req.Traffic.ConnectionPool.Enabled {
				if pool, ok := connPoolStates[ev.nodeID]; ok && pool.active > 0 {
					pool.active--
				}
			}

			if shouldFailWithChaos(state.id, state.retries, state.pathIndex, node, randomness, req.Chaos, ev.timeMs) {
				updateNodeStats(nodeStats, ev.nodeID, eventFailed)

				// Circuit breaker: track consecutive failures.
				if req.Traffic.CircuitBreaker != nil && req.Traffic.CircuitBreaker.Enabled {
					cb := ensureCircuitBreaker(circuitBreakers, ev.nodeID)
					cb.consecutiveFailures++
					cb.lastFailureTimeMs = ev.timeMs
					if cb.state == "half-open" {
						// Failure in half-open → trip back to open
						cb.state = "open"
						circuitBreakerTrips++
						events = appendEvent(events, ev.timeMs, eventCircuitOpen, ev.requestID, ev.nodeID, "half-open test failed")
					} else if cb.consecutiveFailures >= req.Traffic.CircuitBreaker.FailureThreshold {
						cb.state = "open"
						circuitBreakerTrips++
						events = appendEvent(events, ev.timeMs, eventCircuitOpen, ev.requestID, ev.nodeID, fmt.Sprintf("threshold=%d", req.Traffic.CircuitBreaker.FailureThreshold))
					}
				}

				handleFailure(state, req, ev, pq, &seq, &failed, &events, "node failure")
				maybeReportProgress(onProgress, completed, failed, lastReportedResolved, reportEvery, requestsTotal, float64(req.Traffic.DurationSeconds), latencies, totalQueueWait, maxQueueDepth, processingMs, nodeStats, ev)
				lastReportedResolved = completed + failed
				continue
			}

			// Circuit breaker: success resets consecutive failures.
			if req.Traffic.CircuitBreaker != nil && req.Traffic.CircuitBreaker.Enabled {
				cb := ensureCircuitBreaker(circuitBreakers, ev.nodeID)
				if cb.state == "half-open" {
					// Successful request in half-open → close circuit
					cb.state = "closed"
					cb.consecutiveFailures = 0
					events = appendEvent(events, ev.timeMs, eventCircuitClosed, ev.requestID, ev.nodeID, "recovered")
				} else {
					cb.consecutiveFailures = 0
				}
			}

			if state.pathIndex >= len(state.path)-1 {
				state.done = true
				completed++
				latency := ev.timeMs - state.startMs
				latencies = append(latencies, latency)
				events = appendEvent(events, ev.timeMs, eventRequestComplete, ev.requestID, ev.nodeID, "")
				maybeReportProgress(onProgress, completed, failed, lastReportedResolved, reportEvery, requestsTotal, float64(req.Traffic.DurationSeconds), latencies, totalQueueWait, maxQueueDepth, processingMs, nodeStats, ev)
				lastReportedResolved = completed + failed
				continue
			}

			state.pathIndex++
			nextNode := state.path[state.pathIndex]

			// Compute inter-node network latency using capacity module
			targetNodeObj := nodesByID[nextNode]
			networkLatency := InterNodeLatency(node, targetNodeObj, req.Traffic.BaseNetworkLatencyMs)

			// Cross-region latency: additional penalty if multi-region is explicitly configured
			if req.Traffic.MultiRegion != nil && req.Traffic.MultiRegion.Enabled {
				sourceRegion := getConfigString(node.Config, "region")
				targetRegion := getConfigString(targetNodeObj.Config, "region")
				if sourceRegion != "" && targetRegion != "" && sourceRegion != targetRegion {
					if req.Traffic.MultiRegion.CrossRegionLatencyMs > 0 {
						networkLatency += req.Traffic.MultiRegion.CrossRegionLatencyMs
					} else {
						networkLatency += 75 // default cross-region penalty
					}
				}
			}

			// Service mesh sidecar overhead per hop
			if req.Traffic.ServiceMesh != nil && req.Traffic.ServiceMesh.Enabled {
				if hasSidecar(targetNodeObj) {
					networkLatency += req.Traffic.ServiceMesh.SidecarLatencyMs * 2 // egress + ingress
					networkLatency += req.Traffic.ServiceMesh.MTLSOverheadMs
				}
			}

			heap.Push(pq, internalEvent{
				timeMs:    ev.timeMs + networkLatency,
				seq:       seq,
				eventType: eventRequestRouted,
				requestID: ev.requestID,
				nodeID:    nextNode,
			})
			seq++

		case eventRetry:
			node := state.path[state.pathIndex]
			heap.Push(pq, internalEvent{timeMs: ev.timeMs, seq: seq, eventType: eventRequestRouted, requestID: ev.requestID, nodeID: node})
			seq++
		}
	}

	duration := float64(req.Traffic.DurationSeconds)
	avgLatency := average(latencies)
	p50 := percentile(latencies, 0.50)
	p95 := percentile(latencies, 0.95)
	p99 := percentile(latencies, 0.99)
	p999 := percentile(latencies, 0.999)
	errorRate := 0.0
	if requestsTotal > 0 {
		errorRate = float64(failed) / float64(requestsTotal)
	}

	cpuEstimate := 0.0
	if duration > 0 {
		cpuEstimate = math.Min((processingMs/(duration*1000))*100, 100)
	}

	// Reliability engineering metrics
	availability := 0.0
	if requestsTotal > 0 {
		availability = float64(completed) / float64(requestsTotal)
	}
	availabilityNines := 0.0
	if availability > 0 && availability < 1 {
		availabilityNines = -math.Log10(1 - availability)
	} else if availability >= 1 {
		availabilityNines = 9.0 // effectively 100%
	}

	goodput := 0.0
	if duration > 0 {
		goodput = float64(completed) / duration
	}

	// Apdex score: T = 100ms (industry standard for web)
	apdexT := 100.0
	satisfied := 0
	tolerating := 0
	for _, l := range latencies {
		if l <= apdexT {
			satisfied++
		} else if l <= 4*apdexT {
			tolerating++
		}
	}
	apdexScore := 0.0
	if requestsTotal > 0 {
		apdexScore = (float64(satisfied) + float64(tolerating)*0.5) / float64(requestsTotal)
	}

	// MTTR/MTTF/MTBF from failure/recovery event timeline
	mttrMs, mttfMs, mtbfMs := computeReliabilityTimes(events, duration*1000)

	// SLO burn rate: target 99.9% availability over the window
	sloTarget := 0.999
	sloBurnRate := 0.0
	if errorRate > 0 && (1-sloTarget) > 0 {
		sloBurnRate = errorRate / (1 - sloTarget)
	}

	// Latency distribution
	latencyStdDev := stddev(latencies, avgLatency)
	latencyJitter := 0.0
	if avgLatency > 0 {
		latencyJitter = latencyStdDev / avgLatency // coefficient of variation
	}
	histogram := computeHistogram(latencies)

	// Concurrency peak: approximate from events as max overlapping requests
	concurrencyPeak := 0
	if requestsTotal > 0 && duration > 0 {
		// Little's Law: L = λ * W
		littlesL := (float64(requestsTotal) / (duration * 1000)) * avgLatency
		concurrencyPeak = int(math.Ceil(littlesL))
		if concurrencyPeak < 1 {
			concurrencyPeak = 1
		}
	}

	// Memory estimate: ~2KB per concurrent request baseline + queue depth
	memoryEstimate := float64(concurrencyPeak)*2048 + float64(maxQueueDepth)*512

	// Cost per request (simplified model: $0.0000001 per ms of CPU)
	costPerRequest := 0.0
	if completed > 0 {
		costPerRequest = (processingMs * 0.0000001) / float64(completed)
	}
	throughputPerDollar := 0.0
	if costPerRequest > 0 {
		throughputPerDollar = 1.0 / costPerRequest
	}

	metrics := model.Metrics{
		RequestsTotal:           requestsTotal,
		Completed:               completed,
		Failed:                  failed,
		RPS:                     round(float64(completed) / duration),
		AvgLatencyMs:            round(avgLatency),
		P50LatencyMs:            round(p50),
		P95LatencyMs:            round(p95),
		P99LatencyMs:            round(p99),
		P999LatencyMs:           round(p999),
		ErrorRate:               round(errorRate),
		QueueDepthMax:           maxQueueDepth,
		QueueWaitMs:             round(totalQueueWait),
		Throughput:              round(float64(completed) / duration),
		CPUEstimate:             round(cpuEstimate),
		Goodput:                 round(goodput),
		Availability:            round(availability),
		AvailabilityNines:       round(availabilityNines),
		MTTR_Ms:                 round(mttrMs),
		MTTF_Ms:                 round(mttfMs),
		MTBF_Ms:                 round(mtbfMs),
		ApdexScore:              round(apdexScore),
		ApdexT:                  apdexT,
		SLOBurnRate:             round(sloBurnRate),
		LatencyStdDev:           round(latencyStdDev),
		LatencyJitter:           round(latencyJitter),
		LatencyHistogram:        histogram,
		MemoryEstimate:          round(memoryEstimate),
		ConcurrencyPeak:         concurrencyPeak,
		CostPerRequest:          costPerRequest,
		ThroughputPerDollar:     round(throughputPerDollar),
		CircuitBreakerTrips:     circuitBreakerTrips,
		AutoscaleEvents:         autoscaleEvents,
		ConnectionPoolExhausted: connPoolExhausted,
		BackpressureShed:        backpressureShed,
		BrownoutActivations:     brownoutActivations,
		PeakLoadFactor:          round(peakLoadFactor),
	}

	// ─── Industry-Standard Resilience Metrics ─────────────────────────────

	// 1. Blast Radius Analysis: quantify failure impact per-node using dependency-weighted scoring
	metrics.BlastRadius = computeBlastRadius(nodeStats, nextByID, req.Topology.Nodes, requestsTotal)

	// 2. Cascading Failure Detection: identify domino-effect failure chains
	metrics.CascadeChain = detectCascadeChains(events, nextByID, nodeStats)

	// 3. Error Budget (Google SRE methodology): multi-window burn rate tracking
	metrics.ErrorBudget = computeErrorBudget(errorRate, sloTarget, duration)

	// 4. Steady-State Deviation: compare second-half metrics against first-half baseline
	metrics.SteadyStateDeviation = computeSteadyStateDeviation(latencies)

	// 5. Critical Path Analysis: identify latency bottleneck nodes
	metrics.CriticalPath = computeCriticalPath(path, nodesByID, queueStates, nodeStats, avgLatency)

	// 6. Composite Resilience Score (0-100)
	metrics.ResilienceScore = computeResilienceScore(metrics)

	eventsTruncated := false
	if len(events) > 1000 {
		eventsTruncated = true
		events = events[:1000]
	}

	virtualDurationMs := duration * 1000
	if len(events) > 0 {
		virtualDurationMs = events[len(events)-1].TimeMs
	}

	result := model.RunResponse{
		Metrics: metrics,
		Events:  events,
		Metadata: model.RunMetadata{
			Deterministic:         randomness.deterministic,
			SeedUsed:              randomness.seedUsed,
			VirtualDurationMs:     round(virtualDurationMs),
			EventTaxonomyVersion:  eventTaxonomyVersion,
			EventsTruncated:       eventsTruncated,
			ResolvedRequestCounts: completed + failed,
		},
	}

	// Generate capacity report with actionable insights
	result.CapacityReport = GenerateCapacityReport(req.Topology, req.Traffic, metrics, nodeStats)

	if onProgress != nil {
		onProgress(model.ProgressSnapshot{
			Progress:      1,
			Metrics:       metrics,
			CurrentEvent:  eventRequestComplete,
			CurrentTimeMs: round(duration * 1000),
			NodeStats:     cloneNodeStats(nodeStats),
		})
	}

	return result, nil
}

func derivePath(nodes []model.TopologyNode, nextByID map[string][]string) ([]string, error) {
	start := ""
	for _, node := range nodes {
		if strings.EqualFold(node.Type, "api-gateway") {
			start = node.ID
			break
		}
	}
	if start == "" && len(nodes) > 0 {
		// Prefer load-balancer as start if no gateway
		for _, node := range nodes {
			if strings.EqualFold(node.Type, "load-balancer") {
				start = node.ID
				break
			}
		}
		if start == "" {
			start = nodes[0].ID
		}
	}

	// BFS to find the longest acyclic path (covers most topology nodes)
	path := []string{start}
	visited := map[string]bool{start: true}
	current := start
	for {
		next := nextByID[current]
		if len(next) == 0 {
			break
		}
		// Pick the first unvisited target (depth-first traversal following all edges)
		found := false
		for _, candidate := range next {
			if !visited[candidate] {
				visited[candidate] = true
				path = append(path, candidate)
				current = candidate
				found = true
				break
			}
		}
		if !found {
			break
		}
		if len(path) > len(nodes) {
			return nil, errors.New("unable to derive acyclic request path")
		}
	}

	if len(path) == 0 {
		return nil, errors.New("unable to derive request path")
	}
	return path, nil
}

func ensureQueueState(states map[string]*queueState, node model.TopologyNode) *queueState {
	if existing, ok := states[node.ID]; ok {
		return existing
	}
	serviceMs := 10.0
	maxDepth := 1000
	concurrency := 1
	if rate, ok := getNumber(node.Config, "processingRate"); ok && rate > 0 {
		serviceMs = 1000 / rate
	}
	if depth, ok := getNumber(node.Config, "maxDepth"); ok && depth > 0 {
		maxDepth = int(depth)
	}
	// M/M/c: support concurrent workers (replicas, workers, concurrency config)
	if c, ok := getNumber(node.Config, "concurrency"); ok && c > 1 {
		concurrency = int(c)
	} else if c, ok := getNumber(node.Config, "replicas"); ok && c > 1 {
		concurrency = int(c)
	} else if c, ok := getNumber(node.Config, "workers"); ok && c > 1 {
		concurrency = int(c)
	}
	if concurrency < 1 {
		concurrency = 1
	}
	slots := make([]float64, concurrency)
	state := &queueState{availableAt: slots, serviceMs: serviceMs, maxDepth: maxDepth, concurrency: concurrency}
	states[node.ID] = state
	return state
}

// nextAvailableSlot returns the index and time of the earliest free worker in an M/M/c queue
func nextAvailableSlot(qs *queueState) (int, float64) {
	minIdx := 0
	minTime := qs.availableAt[0]
	for i := 1; i < len(qs.availableAt); i++ {
		if qs.availableAt[i] < minTime {
			minTime = qs.availableAt[i]
			minIdx = i
		}
	}
	return minIdx, minTime
}

func handleFailure(state *requestState, req model.RunRequest, ev internalEvent, pq *eventQueue, seq *int, failed *int, events *[]model.SimulationEvent, detail string) {
	*events = appendEvent(*events, ev.timeMs, eventFailed, ev.requestID, ev.nodeID, detail)
	if state.retries < req.Traffic.MaxRetries {
		state.retries++
		retryAt := ev.timeMs + req.Traffic.RetryPenaltyMs
		heap.Push(pq, internalEvent{timeMs: retryAt, seq: *seq, eventType: eventRetry, requestID: ev.requestID, nodeID: ev.nodeID})
		*seq = *seq + 1
		return
	}
	state.done = true
	state.failed = true
	*events = appendEvent(*events, ev.timeMs, eventRequestFailed, ev.requestID, ev.nodeID, "retry budget exhausted")
	*failed++
}

func appendEvent(events []model.SimulationEvent, timeMs float64, eventType string, requestID int, nodeID, detail string) []model.SimulationEvent {
	return append(events, model.SimulationEvent{
		TimeMs:    round(timeMs),
		Type:      eventType,
		RequestID: requestID,
		NodeID:    nodeID,
		Detail:    detail,
	})
}

func processingTime(node model.TopologyNode, randomness randomizer) float64 {
	// Resolve base latency from instance profile (capacity-aware)
	profile := ResolveProfile(node)
	base := profile.BaseLatency

	// Allow explicit override from node config (takes priority over instance profile)
	if v, ok := getNumber(node.Config, "latencyMs"); ok {
		base = v
	} else if v, ok := getNumber(node.Config, "processingTimeMs"); ok {
		base = v
	}

	// Cold start penalty for serverless
	if runtime, ok := node.Config["runtime"]; ok {
		if rt, ok := runtime.(string); ok && rt == "serverless" {
			coldStart := 150.0 // default cold start
			if cs, ok := getNumber(node.Config, "coldStartMs"); ok {
				coldStart = cs
			}
			// Apply cold start probability (first request or after idle)
			if randomness.enabled && randomness.nextFloat64() < 0.05 {
				base += coldStart
			}
		}
	}

	if !randomness.enabled || randomness.latencyJitter <= 0 {
		return base
	}

	// Log-normal distribution: realistic long-tail latency model
	u1 := randomness.nextFloat64()
	u2 := randomness.nextFloat64()
	if u1 <= 0 {
		u1 = 0.0001
	}
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	sigma := randomness.latencyJitter
	multiplier := math.Exp(z * sigma)
	if multiplier < 0.2 {
		multiplier = 0.2
	}
	if multiplier > 5.0 {
		multiplier = 5.0
	}
	return base * multiplier
}

func shouldFail(requestID, retries, pathIndex int, node model.TopologyNode, randomness randomizer) bool {
	rate := 0.0
	if v, ok := getNumber(node.Config, "failureRate"); ok {
		rate = v
	}
	if v, ok := getNumber(node.Config, "failureChance"); ok && v > rate {
		rate = v
	}
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	if randomness.enabled && randomness.failureVariance > 0 {
		rate = rate + ((randomness.nextFloat64()*2 - 1) * randomness.failureVariance * rate)
		if rate < 0 {
			rate = 0
		}
		if rate > 1 {
			rate = 1
		}
	}

	if randomness.enabled {
		return randomness.nextFloat64() < rate
	}

	seed := uint64((requestID+1)*97 + (retries+1)*193 + (pathIndex+1)*389 + len(node.ID)*17)
	rand := float64((seed*1664525+1013904223)%10000) / 10000
	return rand < rate
}

// shouldFailWithChaos extends shouldFail with chaos injection overrides.
func shouldFailWithChaos(requestID, retries, pathIndex int, node model.TopologyNode, randomness randomizer, chaos []model.ChaosInjection, timeMs float64) bool {
	// Check for active chaos injections targeting this node.
	for _, c := range chaos {
		if !isChaosActive(c, node.ID, timeMs) {
			continue
		}
		switch c.Type {
		case "node-crash":
			return true // node is down
		case "packet-loss":
			if c.PacketLossPct > 0 {
				if randomness.enabled {
					return randomness.nextFloat64() < (c.PacketLossPct / 100)
				}
				return float64(requestID%100) < c.PacketLossPct
			}
		case "slowdown":
			// Slowdown doesn't cause failures, handled in processingTime
		}
		// Override failure rate if specified
		if c.FailureRate > 0 {
			if randomness.enabled {
				return randomness.nextFloat64() < c.FailureRate
			}
			seed := uint64((requestID+1)*97 + (retries+1)*193 + (pathIndex+1)*389)
			r := float64((seed*1664525+1013904223)%10000) / 10000
			return r < c.FailureRate
		}
	}
	// Fall back to normal failure logic.
	return shouldFail(requestID, retries, pathIndex, node, randomness)
}

// chaosLatencyMultiplier returns a latency multiplier for active chaos slowdowns.
func chaosLatencyMultiplier(chaos []model.ChaosInjection, nodeID string, timeMs float64) float64 {
	for _, c := range chaos {
		if c.Type == "slowdown" && isChaosActive(c, nodeID, timeMs) && c.Multiplier > 1 {
			return c.Multiplier
		}
	}
	return 1.0
}

// isChaosActive checks if a chaos injection applies to a node at a given time.
func isChaosActive(c model.ChaosInjection, nodeID string, timeMs float64) bool {
	if c.Target != "*" && c.Target != nodeID {
		return false
	}
	if timeMs < c.StartMs {
		return false
	}
	if c.DurationMs > 0 && timeMs > c.StartMs+c.DurationMs {
		return false
	}
	return true
}

func buildRandomizer(req model.RunRequest) randomizer {
	rCfg := req.Traffic.Randomness
	r := randomizer{
		enabled:         rCfg.Enabled,
		arrivalJitter:   clamp(rCfg.ArrivalJitter, 0, 1),
		latencyJitter:   clamp(rCfg.LatencyJitter, 0, 1),
		failureVariance: clamp(rCfg.FailureVariance, 0, 1),
	}
	if !r.enabled {
		r.deterministic = true
		return r
	}

	seed := rCfg.Seed
	deterministic := true
	if seed == 0 {
		seed = time.Now().UnixNano()
		deterministic = false
	}
	r.deterministic = deterministic
	r.seedUsed = seed
	r.rng = rand.New(rand.NewSource(seed))
	return r
}

func (r randomizer) nextFloat64() float64 {
	if !r.enabled || r.rng == nil {
		return 0
	}
	return r.rng.Float64()
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func getNumber(config map[string]any, key string) (float64, bool) {
	value, ok := config[key]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	index := int(math.Ceil(float64(len(copyValues))*p)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(copyValues) {
		index = len(copyValues) - 1
	}
	return copyValues[index]
}

func round(value float64) float64 {
	return math.Round(value*100) / 100
}

func maybeReportProgress(
	onProgress func(model.ProgressSnapshot),
	completed int,
	failed int,
	lastReportedResolved int,
	reportEvery int,
	requestsTotal int,
	duration float64,
	latencies []float64,
	totalQueueWait float64,
	maxQueueDepth int,
	processingMs float64,
	nodeStats map[string]model.NodeRuntimeStats,
	ev internalEvent,
) {
	if onProgress == nil {
		return
	}
	resolved := completed + failed
	if resolved == 0 {
		return
	}
	if resolved-lastReportedResolved < reportEvery && resolved != requestsTotal {
		return
	}
	errorRate := 0.0
	if requestsTotal > 0 {
		errorRate = float64(failed) / float64(requestsTotal)
	}
	cpuEstimate := 0.0
	if duration > 0 {
		cpuEstimate = math.Min((processingMs/(duration*1000))*100, 100)
	}
	onProgress(model.ProgressSnapshot{
		Progress: round(float64(resolved) / float64(requestsTotal)),
		Metrics: model.Metrics{
			RequestsTotal: requestsTotal,
			Completed:     completed,
			Failed:        failed,
			RPS:           round(float64(completed) / duration),
			AvgLatencyMs:  round(average(latencies)),
			P50LatencyMs:  round(percentile(latencies, 0.50)),
			P95LatencyMs:  round(percentile(latencies, 0.95)),
			P99LatencyMs:  round(percentile(latencies, 0.99)),
			ErrorRate:     round(errorRate),
			QueueDepthMax: maxQueueDepth,
			QueueWaitMs:   round(totalQueueWait),
			Throughput:    round(float64(completed) / duration),
			CPUEstimate:   round(cpuEstimate),
		},
		CurrentEvent:  ev.eventType,
		CurrentTimeMs: round(ev.timeMs),
		NodeStats:     cloneNodeStats(nodeStats),
	})
}

func updateNodeStats(stats map[string]model.NodeRuntimeStats, nodeID, eventType string) {
	if nodeID == "" {
		return
	}

	current := stats[nodeID]
	current.TouchCount++
	switch eventType {
	case eventFailed:
		current.FailureCount++
	case eventQueued:
		current.QueuedCount++
	case eventProcessComplete:
		current.CompletedCount++
	}
	stats[nodeID] = current
}

func cloneNodeStats(stats map[string]model.NodeRuntimeStats) map[string]model.NodeRuntimeStats {
	if len(stats) == 0 {
		return nil
	}
	copyStats := make(map[string]model.NodeRuntimeStats, len(stats))
	for key, value := range stats {
		copyStats[key] = value
	}
	return copyStats
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func stddev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}

func computeHistogram(latencies []float64) [12]int {
	// Buckets: 0-5, 5-10, 10-25, 25-50, 50-100, 100-250, 250-500, 500-1000, 1000-2500, 2500-5000, 5000-10000, 10000+
	boundaries := [11]float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
	var hist [12]int
	for _, l := range latencies {
		placed := false
		for i, b := range boundaries {
			if l <= b {
				hist[i]++
				placed = true
				break
			}
		}
		if !placed {
			hist[11]++
		}
	}
	return hist
}

func computeReliabilityTimes(events []model.SimulationEvent, totalDurationMs float64) (mttrMs, mttfMs, mtbfMs float64) {
	// Track failure and recovery events to compute MTTR, MTTF, MTBF
	// A "failure" is a REQUEST_FAILED event; a "recovery" is the next REQUEST_COMPLETE after a failure
	var failureTimes []float64
	var recoveryDurations []float64
	var timeBetweenFailures []float64

	lastFailureTime := -1.0
	lastRecoveryTime := 0.0

	for _, ev := range events {
		switch ev.Type {
		case eventRequestFailed:
			failureTimes = append(failureTimes, ev.TimeMs)
			if lastFailureTime >= 0 {
				timeBetweenFailures = append(timeBetweenFailures, ev.TimeMs-lastFailureTime)
			} else {
				// Time from start to first failure
				timeBetweenFailures = append(timeBetweenFailures, ev.TimeMs)
			}
			lastFailureTime = ev.TimeMs
		case eventRequestComplete:
			if lastFailureTime >= 0 && ev.TimeMs > lastFailureTime && lastRecoveryTime <= lastFailureTime {
				recoveryDurations = append(recoveryDurations, ev.TimeMs-lastFailureTime)
				lastRecoveryTime = ev.TimeMs
			}
		}
	}

	if len(recoveryDurations) > 0 {
		sum := 0.0
		for _, d := range recoveryDurations {
			sum += d
		}
		mttrMs = sum / float64(len(recoveryDurations))
	}

	if len(timeBetweenFailures) > 0 {
		sum := 0.0
		for _, d := range timeBetweenFailures {
			sum += d
		}
		mtbfMs = sum / float64(len(timeBetweenFailures))
		mttfMs = mtbfMs - mttrMs
		if mttfMs < 0 {
			mttfMs = mtbfMs * 0.9 // fallback: MTTF is ~90% of MTBF
		}
	} else if totalDurationMs > 0 {
		// No failures — MTTF is the whole duration
		mttfMs = totalDurationMs
		mtbfMs = totalDurationMs
	}

	return
}

func getConfigString(config map[string]any, key string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func ensureCircuitBreaker(states map[string]*circuitBreakerState, nodeID string) *circuitBreakerState {
	if cb, ok := states[nodeID]; ok {
		return cb
	}
	cb := &circuitBreakerState{state: "closed"}
	states[nodeID] = cb
	return cb
}

func ensureAutoscaleState(states map[string]*autoscaleState, nodeID string, queueStates map[string]*queueState) *autoscaleState {
	if as, ok := states[nodeID]; ok {
		return as
	}
	replicas := 1
	if qs, ok := queueStates[nodeID]; ok {
		replicas = qs.concurrency
	}
	as := &autoscaleState{currentReplicas: replicas}
	states[nodeID] = as
	return as
}

func hasSidecar(node model.TopologyNode) bool {
	if v, ok := node.Config["sidecar"]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	// Default: services and api-gateways have sidecars in mesh mode
	typeLower := strings.ToLower(node.Type)
	return typeLower == "service" || typeLower == "api-gateway"
}

// --- Connection Pool helpers ---

func ensureConnPool(states map[string]*connectionPoolState, nodeID string, cfg *model.ConnectionPoolConfig) *connectionPoolState {
	if pool, ok := states[nodeID]; ok {
		return pool
	}
	maxConn := cfg.MaxConnections
	if maxConn <= 0 {
		maxConn = 100
	}
	pool := &connectionPoolState{
		maxConnections: maxConn,
		acquireTimeout: cfg.AcquireTimeout,
	}
	states[nodeID] = pool
	return pool
}

// --- Load factor computation ---

func computeLoadFactor(queueStates map[string]*queueState, nodeID string, timeMs float64) float64 {
	qs, ok := queueStates[nodeID]
	if !ok {
		return 0
	}
	// Load factor: ratio of busy workers to total capacity
	busy := 0
	for _, slot := range qs.availableAt {
		if slot > timeMs {
			busy++
		}
	}
	if qs.concurrency <= 0 {
		return 0
	}
	return float64(busy) / float64(qs.concurrency)
}

// --- Brownout helpers ---

func ensureBrownoutState(states map[string]*brownoutState, nodeID string, timeMs float64) *brownoutState {
	if bs, ok := states[nodeID]; ok {
		// Reset window every 1000ms
		if timeMs-bs.windowStartMs > 1000 {
			bs.requestsInWindow = 0
			bs.windowStartMs = timeMs
		}
		return bs
	}
	bs := &brownoutState{windowStartMs: timeMs}
	states[nodeID] = bs
	return bs
}

// ─── Industry-Standard Resilience Metric Computations ─────────────────────

// computeBlastRadius calculates how much of the system is impacted when each node fails.
// Uses dependency-weighted scoring: nodes with more downstream dependents have higher impact.
func computeBlastRadius(nodeStats map[string]model.NodeRuntimeStats, nextByID map[string][]string, nodes []model.TopologyNode, requestsTotal int) *model.BlastRadiusAnalysis {
	if len(nodes) == 0 {
		return nil
	}

	totalNodes := len(nodes)
	perNodeImpact := make(map[string]float64, totalNodes)

	// For each node, compute its downstream reach (BFS) as a fraction of total nodes
	for _, node := range nodes {
		reachable := bfsReachCount(node.ID, nextByID, totalNodes)
		// Impact = fraction of nodes downstream + weighted by traffic through this node
		trafficWeight := 0.0
		if requestsTotal > 0 {
			stats := nodeStats[node.ID]
			trafficWeight = float64(stats.TouchCount) / float64(requestsTotal)
		}
		// Combined score: topology reach (60%) + traffic weight (40%)
		impact := 0.6*float64(reachable)/float64(totalNodes) + 0.4*trafficWeight
		if impact > 1 {
			impact = 1
		}
		perNodeImpact[node.ID] = round(impact)
	}

	// Find most vulnerable node
	var mostVulnerable string
	var highestScore float64
	impactedCount := 0
	for nodeID, score := range perNodeImpact {
		if score > 0.1 {
			impactedCount++
		}
		if score > highestScore {
			highestScore = score
			mostVulnerable = nodeID
		}
	}

	// User impact estimate: based on the most vulnerable node's traffic share
	userImpact := 0.0
	if requestsTotal > 0 && mostVulnerable != "" {
		stats := nodeStats[mostVulnerable]
		userImpact = (float64(stats.TouchCount) / float64(requestsTotal)) * 100
	}

	return &model.BlastRadiusAnalysis{
		ImpactedNodes:       impactedCount,
		TotalNodes:          totalNodes,
		ImpactPercentage:    round(float64(impactedCount) / float64(totalNodes) * 100),
		UserImpactEstimate:  round(userImpact),
		PerNodeImpact:       perNodeImpact,
		MostVulnerableNode:  mostVulnerable,
		MostVulnerableScore: round(highestScore),
	}
}

// bfsReachCount counts how many nodes are reachable from startID via directed edges.
func bfsReachCount(startID string, nextByID map[string][]string, maxNodes int) int {
	visited := map[string]bool{startID: true}
	queue := []string{startID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, neighbor := range nextByID[current] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
		if len(visited) >= maxNodes {
			break
		}
	}
	// Don't count the start node itself
	return len(visited) - 1
}

// detectCascadeChains identifies cascading failure patterns from the event timeline.
// A cascade occurs when a failure in node A causes failures in downstream nodes B, C...
func detectCascadeChains(events []model.SimulationEvent, nextByID map[string][]string, nodeStats map[string]model.NodeRuntimeStats) []model.CascadeFailure {
	// Track per-node failure windows (bursts of failures within 500ms of each other)
	type failureBurst struct {
		nodeID  string
		startMs float64
		lastMs  float64
		count   int
	}

	nodeFailureTimes := make(map[string][]float64)
	for _, ev := range events {
		if ev.Type == eventFailed && ev.NodeID != "" {
			nodeFailureTimes[ev.NodeID] = append(nodeFailureTimes[ev.NodeID], ev.TimeMs)
		}
	}

	var chains []model.CascadeFailure

	// For each edge (source → target), check if failure bursts in source
	// are followed by failure bursts in target within a propagation window
	propagationWindowMs := 500.0
	for source, targets := range nextByID {
		sourceFails := nodeFailureTimes[source]
		if len(sourceFails) < 3 { // need a meaningful burst
			continue
		}
		for _, target := range targets {
			targetFails := nodeFailureTimes[target]
			if len(targetFails) < 2 {
				continue
			}
			// Check temporal correlation: target failures starting after source failures
			for _, sf := range sourceFails {
				for _, tf := range targetFails {
					delay := tf - sf
					if delay > 0 && delay < propagationWindowMs {
						mechanism := "backpressure"
						if delay < 50 {
							mechanism = "timeout"
						}
						chains = append(chains, model.CascadeFailure{
							SourceNode:    source,
							AffectedNode:  target,
							PropagationMs: round(delay),
							Mechanism:     mechanism,
							Depth:         1,
						})
						// Only record first correlation per edge
						goto nextTarget
					}
				}
			}
		nextTarget:
		}
	}

	// Deduplicate and compute transitive depths
	if len(chains) > 0 {
		chains = deduplicateAndDepth(chains)
	}

	// Cap at 20 chains for response size
	if len(chains) > 20 {
		chains = chains[:20]
	}
	return chains
}

// deduplicateAndDepth removes duplicate cascade entries and computes transitive depths.
func deduplicateAndDepth(chains []model.CascadeFailure) []model.CascadeFailure {
	seen := make(map[string]bool)
	var unique []model.CascadeFailure
	depthMap := make(map[string]int) // nodeID → max depth as target

	for _, c := range chains {
		key := c.SourceNode + "→" + c.AffectedNode
		if seen[key] {
			continue
		}
		seen[key] = true
		// Depth: if source was already a target, this is depth+1
		depth := 1
		if d, ok := depthMap[c.SourceNode]; ok {
			depth = d + 1
		}
		c.Depth = depth
		depthMap[c.AffectedNode] = depth
		unique = append(unique, c)
	}
	return unique
}

// computeErrorBudget implements Google SRE error budget methodology with multi-window burn rates.
func computeErrorBudget(errorRate, sloTarget, durationSec float64) *model.ErrorBudgetStatus {
	budgetTotal := 1 - sloTarget // e.g., 0.001 for 99.9%
	if budgetTotal <= 0 {
		budgetTotal = 0.001
	}

	budgetConsumed := 0.0
	if budgetTotal > 0 {
		budgetConsumed = errorRate / budgetTotal
	}
	if budgetConsumed > 1 {
		budgetConsumed = 1
	}
	budgetRemaining := 1 - budgetConsumed

	// Burn rates extrapolated to standard windows
	// burnRate = (errors observed / duration) / (budget / window)
	// Simplified: how fast we're burning relative to steady-state consumption
	instantBurnRate := 0.0
	if budgetTotal > 0 {
		instantBurnRate = errorRate / budgetTotal
	}

	// Extrapolate to standard SRE windows (relative to sim duration)
	burnRate1h := instantBurnRate  // 1h window burn
	burnRate6h := instantBurnRate  // same rate assumed steady over 6h
	burnRate24h := instantBurnRate // same rate assumed steady over 24h

	// Time to exhaustion (in minutes) at current burn rate
	timeToExhaustion := 0.0
	if errorRate > 0 && budgetRemaining > 0 {
		// At current error rate, how many minutes until budget = 0
		// budget_remaining * budget_total = allowed errors remaining
		// errorRate = errors per request (fraction)
		// With constant traffic, exhaust in: remaining_budget / burn_rate (in same time units)
		timeToExhaustion = (budgetRemaining * budgetTotal * durationSec) / (errorRate * 60)
		if timeToExhaustion < 0 {
			timeToExhaustion = 0
		}
	}
	if errorRate == 0 {
		timeToExhaustion = math.MaxFloat64 // never exhausts
	}

	// Alert severity based on burn rate (Google SRE multi-window alerting)
	severity := "none"
	if instantBurnRate >= 14.4 {
		severity = "page" // 14.4x = exhausts 30-day budget in 2h
	} else if instantBurnRate >= 6 {
		severity = "critical" // 6x = exhausts in ~5h
	} else if instantBurnRate >= 3 {
		severity = "warning" // 3x = exhausts in ~10h
	} else if instantBurnRate >= 1 {
		severity = "info" // at budget, but consuming
	}

	return &model.ErrorBudgetStatus{
		SLOTarget:         sloTarget,
		BudgetTotal:       round(budgetTotal * 100), // as percentage
		BudgetConsumed:    round(budgetConsumed * 100),
		BudgetRemaining:   round(budgetRemaining * 100),
		BurnRate1h:        round(burnRate1h),
		BurnRate6h:        round(burnRate6h),
		BurnRate24h:       round(burnRate24h),
		TimeToExhaustion:  round(timeToExhaustion),
		AlertSeverity:     severity,
		WindowDurationSec: int(durationSec),
	}
}

// computeSteadyStateDeviation measures how much the system deviated from its initial steady state.
// Compares first-half latencies (baseline) against second-half (under sustained load/chaos).
func computeSteadyStateDeviation(latencies []float64) float64 {
	if len(latencies) < 10 {
		return 0
	}

	mid := len(latencies) / 2
	firstHalf := latencies[:mid]
	secondHalf := latencies[mid:]

	baselineAvg := average(firstHalf)
	sustainedAvg := average(secondHalf)

	if baselineAvg <= 0 {
		return 0
	}

	deviation := math.Abs(sustainedAvg-baselineAvg) / baselineAvg * 100
	return round(deviation)
}

// computeCriticalPath identifies which nodes contribute most to end-to-end latency.
func computeCriticalPath(path []string, nodesByID map[string]model.TopologyNode, queueStates map[string]*queueState, nodeStats map[string]model.NodeRuntimeStats, avgLatency float64) []model.CriticalPathNode {
	if len(path) == 0 || avgLatency <= 0 {
		return nil
	}

	result := make([]model.CriticalPathNode, 0, len(path))
	totalContrib := 0.0

	for _, nodeID := range path {
		qs, ok := queueStates[nodeID]
		serviceMs := 10.0
		if ok {
			serviceMs = qs.serviceMs
		}

		// Estimate per-node contribution including queue wait
		queueWait := 0.0
		if ok && qs.concurrency > 0 {
			// Average wait based on utilization
			stats := nodeStats[nodeID]
			utilization := float64(stats.TouchCount) / float64(maxInt(stats.CompletedCount+stats.FailureCount, 1))
			queueWait = serviceMs * utilization * 0.3 // rough M/M/c estimate
		}

		contrib := serviceMs + queueWait
		totalContrib += contrib

		// Saturation: failure rate as proxy
		stats := nodeStats[nodeID]
		saturation := 0.0
		if stats.TouchCount > 0 {
			saturation = float64(stats.FailureCount) / float64(stats.TouchCount)
		}

		result = append(result, model.CriticalPathNode{
			NodeID:         nodeID,
			LatencyContrib: round(contrib),
			Saturation:     round(saturation),
		})
	}

	// Compute percentages and identify bottlenecks
	for i := range result {
		if totalContrib > 0 {
			result[i].ContribPercentage = round(result[i].LatencyContrib / totalContrib * 100)
		}
		result[i].Bottleneck = result[i].ContribPercentage > 40
	}

	return result
}

// computeResilienceScore produces a 0-100 composite score based on multiple resilience dimensions.
// Inspired by Gremlin's reliability score methodology.
func computeResilienceScore(m model.Metrics) float64 {
	score := 100.0

	// Availability dimension (40% weight): penalize based on error rate
	availScore := m.Availability * 100
	if availScore < 99.9 {
		score -= (100 - availScore) * 0.4 * 10 // amplify small availability drops
	}

	// Latency dimension (20% weight): penalize high tail latency ratio
	if m.P50LatencyMs > 0 {
		tailRatio := m.P99LatencyMs / m.P50LatencyMs
		if tailRatio > 10 {
			score -= 15
		} else if tailRatio > 5 {
			score -= 10
		} else if tailRatio > 3 {
			score -= 5
		}
	}

	// SLO burn dimension (15% weight)
	if m.SLOBurnRate > 10 {
		score -= 15
	} else if m.SLOBurnRate > 5 {
		score -= 10
	} else if m.SLOBurnRate > 1 {
		score -= 5
	}

	// Recovery dimension (15% weight): penalize slow MTTR
	if m.MTTR_Ms > 5000 {
		score -= 15
	} else if m.MTTR_Ms > 2000 {
		score -= 10
	} else if m.MTTR_Ms > 500 {
		score -= 5
	}

	// Saturation dimension (10% weight)
	if m.CPUEstimate > 90 {
		score -= 10
	} else if m.CPUEstimate > 70 {
		score -= 5
	}

	if score < 0 {
		score = 0
	}
	return round(score)
}
