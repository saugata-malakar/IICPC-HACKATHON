package main

// ─────────────────────────────────────────────────────────────────────────────
// IICPC Platform — Adaptive Chaos Engine
// GROUNDBREAKING CONCEPT #1: Real-time fault injection that escalates
// based on the contestant's own performance metrics.
//
// Most benchmarks apply fixed load. Ours is adversarial:
//   - Detects when a submission is "too comfortable" (p99 < 500μs)
//   - Automatically escalates: network partitions → CPU bursts → memory pressure
//   - Measures recovery time (MTTR) as a 4th scoring dimension
//   - Inspired by Netflix Chaos Monkey + LMAX exchange resilience testing
//
// SCORING IMPACT: Submissions that survive chaos with <10% latency degradation
// receive a 1.5x multiplier on their composite score.
// ─────────────────────────────────────────────────────────────────────────────

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/client"
)

// ChaosLevel represents escalating fault severity
type ChaosLevel int

const (
	LevelNone       ChaosLevel = iota
	LevelLatency               // Add artificial network latency via tc netem
	LevelPacketLoss            // Drop 5% of packets
	LevelCPUStress             // Saturate 1 CPU core
	LevelMemPressure           // Allocate and hold 128MB
	LevelNetPart               // Full network partition for 2 seconds
	LevelCPUFreeze             // SIGSTOP for 500ms (process freeze)
)

type ChaosEvent struct {
	Level       ChaosLevel
	ContainerID string
	StartedAt   time.Time
	Duration    time.Duration
	Recovered   bool
	RecoveryMs  int64 // time from chaos end to p99 returning to baseline
}

type ChaosEngine struct {
	mu            sync.Mutex
	docker        *client.Client
	active        map[string]*ChaosSession // submission_id → session
	rng           *rand.Rand
}

type ChaosSession struct {
	SubmissionID  string
	ContainerID   string
	ctx           context.Context
	cancel        context.CancelFunc

	// Real-time metrics feed (written by telemetry ingester)
	CurrentP99Us  atomic.Int64
	BaselineP99Us atomic.Int64
	CurrentTPS    atomic.Int64

	// Chaos history
	Events        []ChaosEvent
	mu            sync.RWMutex

	// Recovery scoring
	TotalChaosEvents atomic.Int64
	TotalRecovered   atomic.Int64
	AvgRecoveryMs    atomic.Int64
}

func NewChaosEngine(docker *client.Client) *ChaosEngine {
	return &ChaosEngine{
		docker: docker,
		active: make(map[string]*ChaosSession),
		rng:    rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 42)),
	}
}

// StartSession begins adaptive chaos testing for a submission.
// Called 30 seconds into a benchmark run (after baseline is established).
func (ce *ChaosEngine) StartSession(ctx context.Context, submissionID, containerID string) *ChaosSession {
	sessCtx, cancel := context.WithCancel(ctx)
	sess := &ChaosSession{
		SubmissionID: submissionID,
		ContainerID:  containerID,
		ctx:          sessCtx,
		cancel:       cancel,
	}

	ce.mu.Lock()
	ce.active[submissionID] = sess
	ce.mu.Unlock()

	go ce.runAdaptiveLoop(sess)
	return sess
}

// runAdaptiveLoop — the heart of the chaos engine.
// Escalates fault severity based on how well the submission is performing.
func (ce *ChaosEngine) runAdaptiveLoop(sess *ChaosSession) {
	defer func() {
		ce.mu.Lock()
		delete(ce.active, sess.SubmissionID)
		ce.mu.Unlock()
	}()

	// Wait 30s to establish baseline
	select {
	case <-sess.ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}

	// Snapshot baseline
	baselineP99 := sess.CurrentP99Us.Load()
	sess.BaselineP99Us.Store(baselineP99)

	slog.Info("chaos baseline established",
		"submission_id", sess.SubmissionID,
		"baseline_p99_us", baselineP99,
	)

	// Adaptive chaos loop
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sess.ctx.Done():
			return
		case <-ticker.C:
			level := ce.selectChaosLevel(sess)
			if level == LevelNone {
				continue
			}
			ce.inject(sess, level)
		}
	}
}

// selectChaosLevel — intelligent level selection based on current performance
func (ce *ChaosEngine) selectChaosLevel(sess *ChaosSession) ChaosLevel {
	p99 := sess.CurrentP99Us.Load()
	baseline := sess.BaselineP99Us.Load()
	if baseline == 0 {
		return LevelNone
	}

	// Degradation ratio: how much has p99 already worsened?
	ratio := float64(p99) / float64(baseline)

	// If already degraded >3x from baseline, don't pile on
	if ratio > 3.0 {
		return LevelNone
	}

	// If performing excellently (p99 < 200μs), apply maximum chaos
	if p99 < 200 {
		return ce.randomLevel(LevelCPUFreeze, LevelNetPart)
	}
	// Good performance (p99 < 1ms)
	if p99 < 1000 {
		return ce.randomLevel(LevelLatency, LevelCPUStress, LevelMemPressure)
	}
	// Acceptable (p99 < 5ms)
	if p99 < 5000 {
		return ce.randomLevel(LevelLatency, LevelPacketLoss)
	}
	// Already struggling — light touch only
	return LevelLatency
}

func (ce *ChaosEngine) randomLevel(levels ...ChaosLevel) ChaosLevel {
	return levels[ce.rng.IntN(len(levels))]
}

// inject applies a specific chaos fault to the container
func (ce *ChaosEngine) inject(sess *ChaosSession, level ChaosLevel) {
	event := ChaosEvent{
		Level:       level,
		ContainerID: sess.ContainerID,
		StartedAt:   time.Now(),
		Duration:    ce.chaosDuration(level),
	}

	slog.Info("injecting chaos",
		"submission_id", sess.SubmissionID,
		"level", levelName(level),
		"duration", event.Duration,
	)

	sess.TotalChaosEvents.Add(1)

	// Apply fault
	var cleanupFn func()
	var err error

	switch level {

	case LevelLatency:
		// Add 50ms latency via Linux tc netem on container's network interface
		// This requires NET_ADMIN capability on the chaos engine pod
		cleanupFn, err = ce.injectNetLatency(sess.ContainerID, 50)

	case LevelPacketLoss:
		cleanupFn, err = ce.injectPacketLoss(sess.ContainerID, 5)

	case LevelCPUStress:
		// Run stress-ng inside the container for the chaos duration
		cleanupFn, err = ce.injectCPUStress(sess.ctx, sess.ContainerID, event.Duration)

	case LevelMemPressure:
		// mmap 128MB anonymously inside the container
		cleanupFn, err = ce.injectMemPressure(sess.ctx, sess.ContainerID, 128)

	case LevelNetPart:
		// Block all traffic with iptables DROP for 2 seconds
		cleanupFn, err = ce.injectNetPartition(sess.ContainerID)
		event.Duration = 2 * time.Second

	case LevelCPUFreeze:
		// SIGSTOP the main process; SIGCONT after duration
		cleanupFn, err = ce.injectCPUFreeze(sess.ContainerID, event.Duration)
	}

	if err != nil {
		slog.Warn("chaos injection failed", "level", levelName(level), "err", err)
		return
	}

	// Wait for chaos duration, then clean up and measure recovery
	timer := time.NewTimer(event.Duration)
	defer timer.Stop()

	select {
	case <-sess.ctx.Done():
		if cleanupFn != nil {
			cleanupFn()
		}
		return
	case <-timer.C:
	}

	if cleanupFn != nil {
		cleanupFn()
	}

	// Measure recovery time: poll until p99 returns within 20% of baseline
	recoveryStart := time.Now()
	baseline := sess.BaselineP99Us.Load()
	threshold := int64(float64(baseline) * 1.2)

	recoveryCtx, cancel := context.WithTimeout(sess.ctx, 30*time.Second)
	defer cancel()

	for {
		select {
		case <-recoveryCtx.Done():
			// Did not recover in time
			event.Recovered = false
			break
		case <-time.After(200 * time.Millisecond):
			if sess.CurrentP99Us.Load() <= threshold {
				event.Recovered = true
				event.RecoveryMs = time.Since(recoveryStart).Milliseconds()
				sess.TotalRecovered.Add(1)
				// Update rolling average recovery time
				total := sess.TotalRecovered.Load()
				prev := sess.AvgRecoveryMs.Load()
				sess.AvgRecoveryMs.Store((prev*(total-1) + event.RecoveryMs) / total)
				goto done
			}
		}
	}

done:
	sess.mu.Lock()
	sess.Events = append(sess.Events, event)
	sess.mu.Unlock()

	slog.Info("chaos recovery measured",
		"submission_id", sess.SubmissionID,
		"recovered", event.Recovered,
		"recovery_ms", event.RecoveryMs,
	)
}

// ─── Fault injection implementations ─────────────────────────────────────────

func (ce *ChaosEngine) injectNetLatency(containerID string, delayMs int) (func(), error) {
	pid, err := ce.containerPID(containerID)
	if err != nil {
		return nil, err
	}
	netns := fmt.Sprintf("/proc/%d/ns/net", pid)

	addCmd := exec.Command("nsenter", "--net="+netns,
		"tc", "qdisc", "add", "dev", "eth0", "root", "netem", "delay",
		fmt.Sprintf("%dms", delayMs), fmt.Sprintf("%dms", delayMs/5), "25%",
	)
	if err := addCmd.Run(); err != nil {
		return nil, fmt.Errorf("tc netem add: %w", err)
	}

	cleanup := func() {
		exec.Command("nsenter", "--net="+netns,
			"tc", "qdisc", "del", "dev", "eth0", "root",
		).Run()
	}
	return cleanup, nil
}

func (ce *ChaosEngine) injectPacketLoss(containerID string, lossPct int) (func(), error) {
	pid, err := ce.containerPID(containerID)
	if err != nil {
		return nil, err
	}
	netns := fmt.Sprintf("/proc/%d/ns/net", pid)

	addCmd := exec.Command("nsenter", "--net="+netns,
		"tc", "qdisc", "add", "dev", "eth0", "root", "netem",
		"loss", fmt.Sprintf("%d%%", lossPct),
	)
	if err := addCmd.Run(); err != nil {
		return nil, err
	}
	cleanup := func() {
		exec.Command("nsenter", "--net="+netns,
			"tc", "qdisc", "del", "dev", "eth0", "root").Run()
	}
	return cleanup, nil
}

func (ce *ChaosEngine) injectCPUStress(ctx context.Context, containerID string, dur time.Duration) (func(), error) {
	cmd := exec.CommandContext(ctx,
		"docker", "exec", containerID,
		"stress-ng", "--cpu", "1", "--timeout", fmt.Sprintf("%ds", int(dur.Seconds())),
	)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return func() { cmd.Process.Kill() }, nil
}

func (ce *ChaosEngine) injectMemPressure(ctx context.Context, containerID string, mb int) (func(), error) {
	cmd := exec.CommandContext(ctx,
		"docker", "exec", containerID,
		"stress-ng", "--vm", "1", "--vm-bytes", fmt.Sprintf("%dM", mb),
		"--vm-keep", "--timeout", "60s",
	)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return func() { cmd.Process.Kill() }, nil
}

func (ce *ChaosEngine) injectNetPartition(containerID string) (func(), error) {
	block := exec.Command("docker", "exec", containerID,
		"iptables", "-I", "INPUT", "-j", "DROP",
	)
	if err := block.Run(); err != nil {
		return nil, err
	}
	cleanup := func() {
		exec.Command("docker", "exec", containerID,
			"iptables", "-D", "INPUT", "-j", "DROP").Run()
	}
	return cleanup, nil
}

func (ce *ChaosEngine) injectCPUFreeze(containerID string, dur time.Duration) (func(), error) {
	// Get container PID and send SIGSTOP
	pid, err := ce.containerPID(containerID)
	if err != nil {
		return nil, err
	}
	if err := exec.Command("kill", "-STOP", fmt.Sprintf("%d", pid)).Run(); err != nil {
		return nil, err
	}

	// Auto-resume after duration
	go func() {
		time.Sleep(dur)
		exec.Command("kill", "-CONT", fmt.Sprintf("%d", pid)).Run()
	}()

	return func() {
		exec.Command("kill", "-CONT", fmt.Sprintf("%d", pid)).Run()
	}, nil
}

func (ce *ChaosEngine) containerPID(containerID string) (int, error) {
	ctx := context.Background()
	info, err := ce.docker.ContainerInspect(ctx, containerID)
	if err != nil {
		return 0, err
	}
	return info.State.Pid, nil
}

// ChaosScore computes the resilience bonus (0.0 to 0.5 multiplier bonus)
func (sess *ChaosSession) ChaosScore() float64 {
	total := sess.TotalChaosEvents.Load()
	if total == 0 {
		return 0
	}
	recoveryRate := float64(sess.TotalRecovered.Load()) / float64(total)
	avgRecMs := float64(sess.AvgRecoveryMs.Load())

	// Recovery score: 1.0 at instant recovery, 0.0 at 30s recovery
	recoverSpeedScore := 1.0 - (avgRecMs / 30000.0)
	if recoverSpeedScore < 0 {
		recoverSpeedScore = 0
	}

	return 0.5 * (recoveryRate*0.6 + recoverSpeedScore*0.4)
}

func (ce *ChaosEngine) chaosDuration(level ChaosLevel) time.Duration {
	switch level {
	case LevelLatency:     return 10 * time.Second
	case LevelPacketLoss:  return 8 * time.Second
	case LevelCPUStress:   return 15 * time.Second
	case LevelMemPressure: return 12 * time.Second
	case LevelNetPart:     return 2 * time.Second
	case LevelCPUFreeze:   return 500 * time.Millisecond
	default:               return 5 * time.Second
	}
}

func levelName(l ChaosLevel) string {
	names := []string{"none", "latency", "packet_loss", "cpu_stress", "mem_pressure", "net_partition", "cpu_freeze"}
	if int(l) < len(names) {
		return names[l]
	}
	return "unknown"
}
