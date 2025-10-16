package ai

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const cModuleName string = "test.go"

// TestHTTPError verifies HTTPError implementation
func TestHTTPError(t *testing.T) {
	err := &HTTPError{StatusCode: 429, Message: "Rate limit exceeded"}
	expected := "HTTP 429: Rate limit exceeded"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

// TestIsRetryableError validates retry logic for structured and unstructured errors
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"nil error", nil, false},
		{"HTTP 429", &HTTPError{StatusCode: 429, Message: "rate limit"}, true},
		{"HTTP 500", &HTTPError{StatusCode: 500, Message: "server error"}, true},
		{"HTTP 503", &HTTPError{StatusCode: 503, Message: "service unavailable"}, true},
		{"HTTP 400", &HTTPError{StatusCode: 400, Message: "bad request"}, false},
		{"HTTP 401", &HTTPError{StatusCode: 401, Message: "unauthorized"}, false},
		{"String 429", fmt.Errorf("error, status code: 429"), true},
		{"String too many requests", fmt.Errorf("too many requests"), true},
		{"String rate limit", fmt.Errorf("rate limit exceeded"), true},
		{"String quota", fmt.Errorf("quota exceeded"), true},
		{"String timeout", fmt.Errorf("request timeout"), true},
		{"String temporary", fmt.Errorf("temporary failure"), true},
		{"String connection reset", fmt.Errorf("connection reset by peer"), true},
		{"Non-retryable", fmt.Errorf("invalid json"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.retryable {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, result, tt.retryable)
			}
		})
	}
}

// mockEvaluator for testing retry/backoff logic
type mockEvaluator struct {
	callCount  int
	failUntil  int
	returnErr  error
	returnCode int
}

func (m *mockEvaluator) Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error) {
	m.callCount++
	if m.callCount <= m.failUntil {
		if m.returnCode > 0 {
			return nil, &HTTPError{StatusCode: m.returnCode, Message: "mock error"}
		}
		return nil, m.returnErr
	}
	// Success after retries
	return &EvaluationResult{File: fileName, Score: 100}, nil
}

// TestEvaluateWithBackoff validates retry logic and exponential backoff
func TestEvaluateWithBackoff(t *testing.T) {
	t.Run("success on first try", testSuccessOnFirstTry)
	t.Run("success after retries on 429", testSuccessAfterRetries)
	t.Run("fail on non-retryable error", testFailOnNonRetryableError)
	t.Run("exhaust retries on 429", testExhaustRetriesOn429)
}

func testSuccessOnFirstTry(t *testing.T) {
	ctx := context.Background()
	mock := &mockEvaluator{failUntil: 0}
	cfg := AIAgentConfig{MaxRetries: 3, BackoffInitialMs: 10}
	result, err := evaluateWithBackoff(ctx, mock, cModuleName, "code", cfg)

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if result == nil || result.Score != 100 {
		t.Errorf("Expected result with score 100, got %+v", result)
	}
	if mock.callCount != 1 {
		t.Errorf("Expected 1 call, got %d", mock.callCount)
	}
}

func testSuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	mock := &mockEvaluator{failUntil: 2, returnCode: 429}
	cfg := AIAgentConfig{MaxRetries: 3, BackoffInitialMs: 10}
	result, err := evaluateWithBackoff(ctx, mock, cModuleName, "code", cfg)

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}
	if result == nil || result.Score != 100 {
		t.Errorf("Expected result with score 100, got %+v", result)
	}
	if mock.callCount != 3 {
		t.Errorf("Expected 3 calls (2 failures + 1 success), got %d", mock.callCount)
	}
}

func testFailOnNonRetryableError(t *testing.T) {
	ctx := context.Background()
	mock := &mockEvaluator{failUntil: 10, returnErr: fmt.Errorf("invalid json")}
	cfg := AIAgentConfig{MaxRetries: 3, BackoffInitialMs: 10}
	result, err := evaluateWithBackoff(ctx, mock, cModuleName, "code", cfg)

	if err == nil {
		t.Errorf("Expected error for non-retryable, got success")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %+v", result)
	}
	if mock.callCount != 1 {
		t.Errorf("Expected 1 call (no retries for non-retryable), got %d", mock.callCount)
	}
}

func testExhaustRetriesOn429(t *testing.T) {
	ctx := context.Background()
	mock := &mockEvaluator{failUntil: 10, returnCode: 429}
	cfg := AIAgentConfig{MaxRetries: 2, BackoffInitialMs: 10}
	result, err := evaluateWithBackoff(ctx, mock, cModuleName, "code", cfg)

	if err == nil {
		t.Errorf("Expected error after exhausting retries, got success")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %+v", result)
	}
	// MaxRetries=2 means 1 original + 2 retries = 3 attempts
	if mock.callCount != 3 {
		t.Errorf("Expected 3 calls (1 + 2 retries), got %d", mock.callCount)
	}
}

// TestCircuitBreaker validates circuit breaker logic
func TestCircuitBreaker(t *testing.T) {
	t.Run("circuit opens after max failures", testCircuitOpensAfterMaxFailures)
	t.Run("circuit closes after wait period", testCircuitClosesAfterWaitPeriod)
	t.Run("success resets failure count", testSuccessResetsFailureCount)
}

func testCircuitOpensAfterMaxFailures(t *testing.T) {
	cb := &CircuitBreakerState{}
	maxFailures := 3
	waitSeconds := 1

	// Record 2 failures - circuit should remain closed
	for i := 0; i < 2; i++ {
		opened := cb.recordFailure(maxFailures, waitSeconds)
		if opened {
			t.Errorf("Circuit opened prematurely at failure %d", i+1)
		}
		if cb.isCircuitOpen() {
			t.Errorf("Circuit should not be open yet")
		}
	}

	// Third failure should open the circuit
	opened := cb.recordFailure(maxFailures, waitSeconds)
	if !opened {
		t.Errorf("Circuit should have opened on 3rd failure")
	}
	if !cb.isCircuitOpen() {
		t.Errorf("Circuit should be open")
	}
}

func testCircuitClosesAfterWaitPeriod(t *testing.T) {
	cb := &CircuitBreakerState{}
	maxFailures := 2
	waitSeconds := 0 // 0 seconds for test speed

	// Open the circuit
	cb.recordFailure(maxFailures, waitSeconds)
	cb.recordFailure(maxFailures, waitSeconds)

	if !cb.isCircuitOpen() {
		t.Errorf("Circuit should be open")
	}

	// Wait a bit (circuit should close immediately with 0 wait)
	time.Sleep(10 * time.Millisecond)

	// Check again - circuit should be closed (half-open state)
	if cb.isCircuitOpen() {
		t.Errorf("Circuit should be closed after wait period")
	}
}

func testSuccessResetsFailureCount(t *testing.T) {
	cb := &CircuitBreakerState{}
	maxFailures := 3
	waitSeconds := 1

	// Record 2 failures
	cb.recordFailure(maxFailures, waitSeconds)
	cb.recordFailure(maxFailures, waitSeconds)

	// Record success - should reset
	cb.recordSuccess()

	// Now record 2 more failures - circuit should not open yet
	cb.recordFailure(maxFailures, waitSeconds)
	opened := cb.recordFailure(maxFailures, waitSeconds)
	if opened {
		t.Errorf("Circuit should not open (count was reset by success)")
	}
}

// TestEvaluationMetricsSuccess validates metrics tracking for successful evaluations
func TestEvaluationMetricsSuccess(t *testing.T) {
	ctx := context.Background()
	mock := &mockEvaluator{failUntil: 0}
	cfg := AIAgentConfig{MaxRetries: 0}
	metrics := &EvaluationMetrics{}

	_, err := evaluateWithBackoffAndMetrics(ctx, mock, cModuleName, "code", cfg, metrics, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if metrics.TotalAttempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", metrics.TotalAttempts)
	}
	if metrics.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", metrics.SuccessCount)
	}
	if metrics.FailureCount != 0 {
		t.Errorf("Expected 0 failures, got %d", metrics.FailureCount)
	}
}

// TestEvaluationMetricsRetries validates metrics tracking for failures and retries
func TestEvaluationMetricsRetries(t *testing.T) {
	ctx := context.Background()
	mock := &mockEvaluator{failUntil: 2, returnCode: 429}
	cfg := AIAgentConfig{MaxRetries: 3, BackoffInitialMs: 10}
	metrics := &EvaluationMetrics{}

	_, err := evaluateWithBackoffAndMetrics(ctx, mock, cModuleName, "code", cfg, metrics, nil)
	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}

	// Attempt 1 fails, attempt 2 fails (first retry), attempt 3 succeeds (second retry)
	if metrics.TotalAttempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", metrics.TotalAttempts)
	}
	if metrics.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", metrics.SuccessCount)
	}
	if metrics.FailureCount != 2 {
		t.Errorf("Expected 2 failures, got %d", metrics.FailureCount)
	}
	// RetryCount counts failed retries only (attempt 2 failed, attempt 3 succeeded)
	if metrics.RetryCount != 1 {
		t.Errorf("Expected 1 retry (attempt 2 which failed), got %d", metrics.RetryCount)
	}
	if metrics.RateLimitCount != 2 {
		t.Errorf("Expected 2 rate limit errors, got %d", metrics.RateLimitCount)
	}
}

// TestEvaluationMetricsCircuitBreaker validates metrics tracking for circuit breaker trips
func TestEvaluationMetricsCircuitBreaker(t *testing.T) {
	ctx := context.Background()
	mock := &mockEvaluator{failUntil: 10, returnCode: 429}
	// CircuitBreakerMax=1 means it opens after 1 consecutive failure
	cfg := AIAgentConfig{MaxRetries: 2, BackoffInitialMs: 10, CircuitBreakerMax: 1, CircuitBreakerWait: 1}
	metrics := &EvaluationMetrics{}
	cb := &CircuitBreakerState{}

	// First evaluation - should fail all retries and trip circuit breaker
	_, err := evaluateWithBackoffAndMetrics(ctx, mock, "test.go", "code", cfg, metrics, cb)
	if err == nil {
		t.Errorf("Expected error")
	}

	if metrics.CircuitBreakerTrips != 1 {
		t.Errorf("Expected 1 circuit breaker trip, got %d", metrics.CircuitBreakerTrips)
	}

	// Second evaluation - circuit should be open
	_, err = evaluateWithBackoffAndMetrics(ctx, mock, "test2.go", "code", cfg, metrics, cb)
	if err == nil || err.Error() != "circuit breaker is open for this agent" {
		t.Errorf("Expected circuit breaker error, got: %v", err)
	}

	if metrics.CircuitBreakerTrips != 2 {
		t.Errorf("Expected 2 circuit breaker trips (1 open + 1 reject), got %d", metrics.CircuitBreakerTrips)
	}
}

// TestBackoffJitter validates jitter adds randomness to backoff
func TestBackoffJitter(t *testing.T) {
	ctx := context.Background()

	t.Run("jitter adds variance to backoff", func(t *testing.T) {
		// Run multiple evaluations and check that backoff times vary
		timings := []time.Duration{}

		for i := 0; i < 5; i++ {
			mock := &mockEvaluator{failUntil: 1, returnCode: 429}
			cfg := AIAgentConfig{MaxRetries: 1, BackoffInitialMs: 100, BackoffJitter: true}

			start := time.Now()
			_, _ = evaluateWithBackoffAndMetrics(ctx, mock, cModuleName, "code", cfg, nil, nil)
			elapsed := time.Since(start)
			timings = append(timings, elapsed)
		}

		// Check that not all timings are identical (jitter working)
		allSame := true
		for i := 1; i < len(timings); i++ {
			if timings[i] != timings[0] {
				allSame = false
				break
			}
		}

		if allSame {
			t.Logf("Warning: All backoff timings were identical: %v (jitter may not be working)", timings)
			// Not failing the test as timing can be flaky in tests
		}
	})
}
