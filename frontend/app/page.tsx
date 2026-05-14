"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import Link from "next/link";

export default function HomePage() {
  const [countdown, setCountdown] = useState({ days: 0, hours: 0, minutes: 0, seconds: 0 });
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    const targetDate = new Date("2026-06-10T23:59:59").getTime();

    const interval = setInterval(() => {
      const now = new Date().getTime();
      const distance = targetDate - now;

      if (distance < 0) {
        clearInterval(interval);
        return;
      }

      setCountdown({
        days: Math.floor(distance / (1000 * 60 * 60 * 24)),
        hours: Math.floor((distance % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60)),
        minutes: Math.floor((distance % (1000 * 60 * 60)) / (1000 * 60)),
        seconds: Math.floor((distance % (1000 * 60)) / 1000),
      });
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  if (!mounted) return null;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-blue-950 to-slate-900 text-white overflow-hidden">
      {/* Animated grid background */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#1e293b_1px,transparent_1px),linear-gradient(to_bottom,#1e293b_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_80%_50%_at_50%_0%,#000_70%,transparent_110%)] opacity-20" />

      {/* Glowing orbs */}
      <div className="absolute top-0 left-1/4 w-96 h-96 bg-blue-500/30 rounded-full blur-[128px] animate-pulse" />
      <div className="absolute bottom-0 right-1/4 w-96 h-96 bg-purple-500/30 rounded-full blur-[128px] animate-pulse delay-1000" />

      <div className="relative z-10 container mx-auto px-6 py-20">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: -50 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8 }}
          className="text-center mb-16"
        >
          <h1 className="text-7xl font-black mb-4 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 via-purple-400 to-pink-400">
            IICPC SUMMER HACKATHON 2026
          </h1>
          <p className="text-2xl text-slate-300 font-light tracking-wide">
            Distributed Benchmarking & Hosting Platform
          </p>
          <div className="mt-6 flex items-center justify-center gap-4">
            <div className="h-px w-24 bg-gradient-to-r from-transparent via-blue-400 to-transparent" />
            <span className="text-sm text-slate-400 uppercase tracking-widest">Championship Edition</span>
            <div className="h-px w-24 bg-gradient-to-r from-transparent via-blue-400 to-transparent" />
          </div>
        </motion.div>

        {/* Countdown */}
        <motion.div
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.8, delay: 0.2 }}
          className="max-w-4xl mx-auto mb-20"
        >
          <div className="bg-slate-900/50 backdrop-blur-xl border border-slate-700/50 rounded-2xl p-8 shadow-2xl">
            <h2 className="text-center text-xl text-slate-400 mb-6 uppercase tracking-wider">
              Submission Deadline
            </h2>
            <div className="grid grid-cols-4 gap-6">
              {[
                { label: "Days", value: countdown.days },
                { label: "Hours", value: countdown.hours },
                { label: "Minutes", value: countdown.minutes },
                { label: "Seconds", value: countdown.seconds },
              ].map((item, idx) => (
                <motion.div
                  key={item.label}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.4 + idx * 0.1 }}
                  className="text-center"
                >
                  <div className="bg-gradient-to-br from-blue-500/20 to-purple-500/20 border border-blue-400/30 rounded-xl p-6 mb-3">
                    <span className="text-5xl font-black bg-clip-text text-transparent bg-gradient-to-br from-blue-300 to-purple-300">
                      {String(item.value).padStart(2, "0")}
                    </span>
                  </div>
                  <span className="text-sm text-slate-400 uppercase tracking-widest">{item.label}</span>
                </motion.div>
              ))}
            </div>
          </div>
        </motion.div>

        {/* Feature Cards */}
        <motion.div
          initial={{ opacity: 0, y: 50 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 0.6 }}
          className="grid md:grid-cols-3 gap-6 mb-16"
        >
          {[
            {
              title: "Live Leaderboard",
              desc: "Real-time rankings with WebSocket updates",
              icon: "📊",
              link: "/leaderboard",
              gradient: "from-blue-500/20 to-cyan-500/20",
              border: "border-blue-400/30",
            },
            {
              title: "Submit Code",
              desc: "Upload your trading infrastructure",
              icon: "🚀",
              link: "/submit",
              gradient: "from-purple-500/20 to-pink-500/20",
              border: "border-purple-400/30",
            },
            {
              title: "Analytics",
              desc: "Deep dive into performance metrics",
              icon: "📈",
              link: "/analytics",
              gradient: "from-green-500/20 to-emerald-500/20",
              border: "border-green-400/30",
            },
          ].map((card, idx) => (
            <Link key={card.title} href={card.link}>
              <motion.div
                whileHover={{ scale: 1.05, y: -5 }}
                whileTap={{ scale: 0.98 }}
                className={`bg-gradient-to-br ${card.gradient} backdrop-blur-xl border ${card.border} rounded-2xl p-8 cursor-pointer transition-all duration-300 hover:shadow-2xl hover:shadow-blue-500/20`}
              >
                <div className="text-6xl mb-4">{card.icon}</div>
                <h3 className="text-2xl font-bold mb-2">{card.title}</h3>
                <p className="text-slate-400">{card.desc}</p>
              </motion.div>
            </Link>
          ))}
        </motion.div>

        {/* Tech Stack Showcase */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 1, delay: 1 }}
          className="bg-slate-900/30 backdrop-blur-xl border border-slate-700/50 rounded-2xl p-8"
        >
          <h2 className="text-3xl font-bold mb-6 text-center">
            <span className="bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-400">
              Championship Tech Stack
            </span>
          </h2>
          <div className="grid md:grid-cols-5 gap-4 text-center">
            {[
              { name: "Go 1.22", desc: "Fiber v2" },
              { name: "Rust 1.77", desc: "Tokio" },
              { name: "Next.js 14", desc: "React 18" },
              { name: "TimescaleDB", desc: "Hypertables" },
              { name: "Kafka", desc: "12 Partitions" },
            ].map((tech) => (
              <div key={tech.name} className="p-4 bg-slate-800/50 rounded-xl border border-slate-700/50">
                <div className="font-bold text-blue-300">{tech.name}</div>
                <div className="text-sm text-slate-400">{tech.desc}</div>
              </div>
            ))}
          </div>
        </motion.div>

        {/* Footer */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 1, delay: 1.2 }}
          className="text-center mt-16 text-slate-500 text-sm"
        >
          <p>Built with 💙 for IICPC Summer Hackathon 2026</p>
          <p className="mt-2">May 9 - June 10, 2026</p>
        </motion.div>
      </div>
    </div>
  );
}
