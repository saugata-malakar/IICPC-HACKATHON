import { NextResponse } from "next/server";

export async function GET() {
  return NextResponse.json({
    submissions: [
      { id: "sub-001", team_name: "QuantumTraders", rank: 1, score: 0.947, p50_latency: 0.12, p90_latency: 0.34, p99_latency: 0.89, tps: 148200, correctness: 0.998, chaos_bonus: 0.15, status: "completed", timestamp: Date.now(), sparkline: [80,85,90,88,92,95,94,97,96,98], predicted_saturation_tps: 185000 },
      { id: "sub-002", team_name: "NanoSecond Capital", rank: 2, score: 0.912, p50_latency: 0.18, p90_latency: 0.45, p99_latency: 1.23, tps: 132400, correctness: 0.995, chaos_bonus: 0.12, status: "running", timestamp: Date.now(), sparkline: [70,75,78,82,80,85,88,86,90,92], predicted_saturation_tps: 165000 },
      { id: "sub-003", team_name: "ByteForce", rank: 3, score: 0.889, p50_latency: 0.22, p90_latency: 0.58, p99_latency: 1.67, tps: 119800, correctness: 0.991, chaos_bonus: 0.08, status: "completed", timestamp: Date.now(), sparkline: [60,65,70,68,72,75,78,80,82,85], predicted_saturation_tps: 150000 },
      { id: "sub-004", team_name: "LowLatencyLabs", rank: 4, score: 0.856, p50_latency: 0.28, p90_latency: 0.72, p99_latency: 2.1, tps: 105600, correctness: 0.987, chaos_bonus: 0.05, status: "running", timestamp: Date.now(), sparkline: [55,58,62,60,65,68,70,72,75,78], predicted_saturation_tps: 135000 },
      { id: "sub-005", team_name: "TurboExchange", rank: 5, score: 0.823, p50_latency: 0.35, p90_latency: 0.91, p99_latency: 2.8, tps: 95200, correctness: 0.982, chaos_bonus: 0.03, status: "completed", timestamp: Date.now(), sparkline: [50,52,55,58,60,62,65,68,70,72] },
      { id: "sub-006", team_name: "AlphaEngine", rank: 6, score: 0.798, p50_latency: 0.42, p90_latency: 1.15, p99_latency: 3.4, tps: 87400, correctness: 0.978, chaos_bonus: 0.0, status: "running", timestamp: Date.now(), sparkline: [45,48,50,53,55,58,60,62,64,66] },
    ],
    type: "leaderboard_update"
  });
}

export async function POST(request: Request) {
  const body = await request.json();
  return NextResponse.json({
    success: true,
    submission_id: `sub-${Date.now()}`,
    message: "Submission received. Benchmark starting...",
    estimated_time: "~120 seconds",
    team: body.team_name || "Anonymous",
  });
}
