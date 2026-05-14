"use client";

import { useEffect, useState, useRef } from "react";
import { motion, AnimatePresence, LayoutGroup } from "framer-motion";
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";

interface Submission {
  id: string;
  team_name: string;
  rank: number;
  score: number;
  p50_latency: number;
  p90_latency: number;
  p99_latency: number;
  tps: number;
  correctness: number;
  chaos_bonus: number;
  status: "running" | "completed" | "failed";
  timestamp: number;
  sparkline: number[];
  predicted_saturation_tps?: number;
}

export default function LeaderboardPage() {
  const [submissions, setSubmissions] = useState<Submission[]>([]);
  const [connected, setConnected] = useState(false);
  const [filter, setFilter] = useState<"all" | "running" | "completed">("all");
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    // Connect to WebSocket
    const ws = new WebSocket(process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8003/ws");
    wsRef.current = ws;

    ws.onopen = () => {
      console.log("✓ WebSocket connected");
      setConnected(true);
    };

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === "leaderboard_update") {
        setSubmissions(data.submissions);
      }
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      setConnected(false);
    };

    ws.onclose = () => {
      console.log("WebSocket disconnected");
      setConnected(false);
    };

    // Cleanup
    return () => {
      ws.close();
    };
  }, []);

  const filteredSubmissions = submissions.filter((sub) => {
    if (filter === "all") return true;
    return sub.status === filter;
  });

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-blue-950 to-slate-900 text-white p-8">
      {/* Header */}
      <div className="max-w-7xl mx-auto mb-8">
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          className="flex items-center justify-between mb-6"
        >
          <div>
            <h1 className="text-5xl font-black bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-400">
              Live Leaderboard
            </h1>
            <p className="text-slate-400 mt-2">Real-time rankings • WebSocket powered</p>
          </div>

          {/* Connection Status */}
          <div className="flex items-center gap-3">
            <div className={`w-3 h-3 rounded-full ${connected ? "bg-green-400 animate-pulse" : "bg-red-400"}`} />
            <span className="text-sm text-slate-400">{connected ? "Connected" : "Disconnected"}</span>
          </div>
        </motion.div>

        {/* Filters */}
        <div className="flex gap-3 mb-6">
          {["all", "running", "completed"].map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f as any)}
              className={`px-6 py-2 rounded-lg font-semibold transition-all ${
                filter === f
                  ? "bg-blue-500 text-white shadow-lg shadow-blue-500/50"
                  : "bg-slate-800/50 text-slate-400 hover:bg-slate-700/50"
              }`}
            >
              {f.charAt(0).toUpperCase() + f.slice(1)}
            </button>
          ))}
        </div>

        {/* Leaderboard Table */}
        <LayoutGroup>
          <motion.div layout className="space-y-3">
            <AnimatePresence mode="popLayout">
              {filteredSubmissions.map((sub, idx) => (
                <motion.div
                  key={sub.id}
                  layout
                  layoutId={sub.id}
                  initial={{ opacity: 0, scale: 0.9 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.9 }}
                  transition={{ type: "spring", stiffness: 300, damping: 30 }}
                  className="bg-slate-900/50 backdrop-blur-xl border border-slate-700/50 rounded-xl p-6 hover:border-blue-500/50 transition-all"
                >
                  <div className="flex items-center gap-6">
                    {/* Rank */}
                    <motion.div
                      layout
                      className={`w-16 h-16 rounded-xl flex items-center justify-center font-black text-2xl ${
                        sub.rank === 1
                          ? "bg-gradient-to-br from-yellow-400 to-yellow-600 text-slate-900"
                          : sub.rank === 2
                          ? "bg-gradient-to-br from-slate-300 to-slate-500 text-slate-900"
                          : sub.rank === 3
                          ? "bg-gradient-to-br from-orange-400 to-orange-600 text-slate-900"
                          : "bg-slate-800 text-slate-400"
                      }`}
                    >
                      {sub.rank}
                    </motion.div>

                    {/* Team Info */}
                    <div className="flex-1">
                      <h3 className="text-xl font-bold text-white mb-1">{sub.team_name}</h3>
                      <div className="flex items-center gap-4 text-sm text-slate-400">
                        <span>ID: {sub.id.slice(0, 8)}</span>
                        <span className="flex items-center gap-1">
                          <div
                            className={`w-2 h-2 rounded-full ${
                              sub.status === "running"
                                ? "bg-green-400 animate-pulse"
                                : sub.status === "completed"
                                ? "bg-blue-400"
                                : "bg-red-400"
                            }`}
                          />
                          {sub.status}
                        </span>
                      </div>
                    </div>

                    {/* Metrics */}
                    <div className="grid grid-cols-4 gap-6 flex-1">
                      <MetricCard label="Score" value={sub.score.toFixed(3)} color="blue" />
                      <MetricCard label="p99 Latency" value={`${sub.p99_latency.toFixed(2)}ms`} color="purple" />
                      <MetricCard label="TPS" value={sub.tps.toLocaleString()} color="green" />
                      <MetricCard
                        label="Correctness"
                        value={`${(sub.correctness * 100).toFixed(1)}%`}
                        color="cyan"
                      />
                    </div>

                    {/* Sparkline */}
                    <div className="w-32 h-16">
                      <ResponsiveContainer width="100%" height="100%">
                        <AreaChart data={sub.sparkline.map((v, i) => ({ value: v }))}>
                          <defs>
                            <linearGradient id={`gradient-${sub.id}`} x1="0" y1="0" x2="0" y2="1">
                              <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.8} />
                              <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
                            </linearGradient>
                          </defs>
                          <Area
                            type="monotone"
                            dataKey="value"
                            stroke="#3b82f6"
                            strokeWidth={2}
                            fill={`url(#gradient-${sub.id})`}
                            isAnimationActive={false}
                          />
                        </AreaChart>
                      </ResponsiveContainer>
                    </div>

                    {/* Chaos Bonus */}
                    {sub.chaos_bonus > 0 && (
                      <motion.div
                        initial={{ scale: 0 }}
                        animate={{ scale: 1 }}
                        className="bg-gradient-to-br from-orange-500/20 to-red-500/20 border border-orange-400/30 rounded-lg px-3 py-2"
                      >
                        <div className="text-xs text-orange-300 uppercase tracking-wider">Chaos Bonus</div>
                        <div className="text-lg font-bold text-orange-200">+{(sub.chaos_bonus * 100).toFixed(0)}%</div>
                      </motion.div>
                    )}
                  </div>

                  {/* Predicted Saturation */}
                  {sub.predicted_saturation_tps && (
                    <motion.div
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: "auto" }}
                      className="mt-4 pt-4 border-t border-slate-700/50 text-sm text-slate-400"
                    >
                      <span className="text-blue-300">🔮 ML Prediction:</span> Will saturate at{" "}
                      <span className="font-bold text-white">{sub.predicted_saturation_tps.toLocaleString()} TPS</span>
                    </motion.div>
                  )}
                </motion.div>
              ))}
            </AnimatePresence>
          </motion.div>
        </LayoutGroup>

        {/* Empty State */}
        {filteredSubmissions.length === 0 && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="text-center py-20 text-slate-500"
          >
            <div className="text-6xl mb-4">📊</div>
            <p className="text-xl">No submissions yet</p>
            <p className="text-sm mt-2">Be the first to submit your trading infrastructure!</p>
          </motion.div>
        )}
      </div>
    </div>
  );
}

function MetricCard({ label, value, color }: { label: string; value: string; color: string }) {
  const colorMap: Record<string, string> = {
    blue: "from-blue-500/20 to-blue-600/20 border-blue-400/30 text-blue-300",
    purple: "from-purple-500/20 to-purple-600/20 border-purple-400/30 text-purple-300",
    green: "from-green-500/20 to-green-600/20 border-green-400/30 text-green-300",
    cyan: "from-cyan-500/20 to-cyan-600/20 border-cyan-400/30 text-cyan-300",
  };

  return (
    <div className={`bg-gradient-to-br ${colorMap[color]} border rounded-lg p-3`}>
      <div className="text-xs text-slate-400 uppercase tracking-wider mb-1">{label}</div>
      <div className="text-lg font-bold">{value}</div>
    </div>
  );
}
