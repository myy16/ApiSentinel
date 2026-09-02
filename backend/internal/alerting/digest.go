package alerting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// DeliveryAlertPayload represents a delivery failure or anomaly alert.
type DeliveryAlertPayload struct {
	EventID        string `json:"eventId"`
	ProjectID      string `json:"projectId"`
	ProjectName    string `json:"projectName"`
	EndpointID     string `json:"endpointId"`
	EndpointName   string `json:"endpointName"`
	TargetURL      string `json:"targetUrl"`
	AlertKind      string `json:"alertKind"` // "INSTANT_CRITICAL" or "DIGEST_SUMMARY"
	StatusCode     int    `json:"statusCode"`
	ErrorType      string `json:"errorType"`
	ErrorMessage   string `json:"errorMessage"`
	TotalFailures  int    `json:"totalFailures"`
	WindowDuration string `json:"windowDuration"`
	Timestamp      string `json:"timestamp"`
}

// DigestCallback is the function invoked when a digest is ready to be dispatched.
type DigestCallback func(ctx context.Context, payload DeliveryAlertPayload)

type endpointFailureBucket struct {
	projectID    string
	projectName  string
	endpointName string
	targetURL    string
	lastStatus   int
	lastError    string
	failureCount int
	firstSeenAt  time.Time
	lastSeenAt   time.Time
}

// DeliveryDigestAccumulator collects and aggregates transient delivery failures (5xx, timeouts)
// to prevent notification spam across channels (anti-spam / debouncing).
type DeliveryDigestAccumulator struct {
	mu           sync.Mutex
	buckets      map[string]*endpointFailureBucket
	window       time.Duration
	threshold    int
	callback     DigestCallback
	ctx          context.Context
	cancel       context.CancelFunc
	flushTicker  *time.Ticker
}

// NewDeliveryDigestAccumulator creates an accumulator with a configurable time window and threshold.
func NewDeliveryDigestAccumulator(window time.Duration, threshold int, callback DigestCallback) *DeliveryDigestAccumulator {
	if window <= 0 {
		window = 1 * time.Minute
	}
	if threshold <= 0 {
		threshold = 10
	}

	ctx, cancel := context.WithCancel(context.Background())
	acc := &DeliveryDigestAccumulator{
		buckets:     make(map[string]*endpointFailureBucket),
		window:      window,
		threshold:   threshold,
		callback:    callback,
		ctx:         ctx,
		cancel:      cancel,
		flushTicker: time.NewTicker(window / 2),
	}

	go acc.periodicFlushLoop()

	return acc
}

// RecordFailure records a failure for an endpoint. If threshold is reached, flushes immediately.
func (a *DeliveryDigestAccumulator) RecordFailure(projectID, projectName, endpointID, endpointName, targetURL string, statusCode int, errMsg string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	bucket, exists := a.buckets[endpointID]
	now := time.Now()

	if !exists {
		bucket = &endpointFailureBucket{
			projectID:    projectID,
			projectName:  projectName,
			endpointName: endpointName,
			targetURL:    targetURL,
			lastStatus:   statusCode,
			lastError:    errMsg,
			failureCount: 1,
			firstSeenAt:  now,
			lastSeenAt:   now,
		}
		a.buckets[endpointID] = bucket
	} else {
		bucket.failureCount++
		bucket.lastStatus = statusCode
		bucket.lastError = errMsg
		bucket.lastSeenAt = now
	}

	// If threshold reached, flush this bucket immediately
	if bucket.failureCount >= a.threshold {
		a.flushBucketLocked(endpointID, bucket)
	}
}

// FlushAll flushes all pending digest buckets.
func (a *DeliveryDigestAccumulator) FlushAll() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for epID, bucket := range a.buckets {
		a.flushBucketLocked(epID, bucket)
	}
}

// Close stops the background flush goroutine and flushes remaining.
func (a *DeliveryDigestAccumulator) Close() {
	a.cancel()
	if a.flushTicker != nil {
		a.flushTicker.Stop()
	}
	a.FlushAll()
}

func (a *DeliveryDigestAccumulator) periodicFlushLoop() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-a.flushTicker.C:
			a.checkAndFlushExpired()
		}
	}
}

func (a *DeliveryDigestAccumulator) checkAndFlushExpired() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	for epID, bucket := range a.buckets {
		if now.Sub(bucket.firstSeenAt) >= a.window {
			a.flushBucketLocked(epID, bucket)
		}
	}
}

func (a *DeliveryDigestAccumulator) flushBucketLocked(endpointID string, bucket *endpointFailureBucket) {
	if bucket.failureCount == 0 {
		delete(a.buckets, endpointID)
		return
	}

	windowDurationStr := fmt.Sprintf("%ds", int(time.Since(bucket.firstSeenAt).Seconds()))
	if windowDurationStr == "0s" {
		windowDurationStr = "1s"
	}

	epPrefix := endpointID
	if len(epPrefix) > 8 {
		epPrefix = epPrefix[:8]
	}

	payload := DeliveryAlertPayload{
		EventID:        fmt.Sprintf("digest-%s-%d", epPrefix, time.Now().UnixNano()),
		ProjectID:      bucket.projectID,
		ProjectName:    bucket.projectName,
		EndpointID:     endpointID,
		EndpointName:   bucket.endpointName,
		TargetURL:      bucket.targetURL,
		AlertKind:      "DIGEST_SUMMARY",
		StatusCode:     bucket.lastStatus,
		ErrorType:      "DELIVERY_ANOMALY_DIGEST",
		ErrorMessage:   bucket.lastError,
		TotalFailures:  bucket.failureCount,
		WindowDuration: windowDurationStr,
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	delete(a.buckets, endpointID)

	if a.callback != nil {
		go func(p DeliveryAlertPayload) {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Msg("Recovered from digest callback panic")
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			a.callback(ctx, p)
		}(payload)
	}
}
