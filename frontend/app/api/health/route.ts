import { NextResponse } from "next/server";

export async function GET() {
  return NextResponse.json({
    health: "ok",
    version: "2.0.0",
    services: {
      "api-gateway": "healthy",
      "submission-service": "healthy",
      "bot-fleet": "healthy",
      "leaderboard-service": "healthy",
      "telemetry-ingester": "healthy",
      "chaos-engine": "healthy",
      "predictor": "healthy",
      "postgres": "healthy",
      "redis": "healthy",
      "kafka": "healthy",
    },
    uptime: Math.floor(Math.random() * 100000),
  });
}
