// app/layout.tsx
import type { Metadata } from "next";
import { JetBrains_Mono, Space_Grotesk } from "next/font/google";
import "./globals.css";
import { Nav } from "@/components/ui/Nav";

const mono = JetBrains_Mono({ subsets:["latin"], variable:"--font-jetbrains", weight:["400","500","700"] });
const sans = Space_Grotesk({ subsets:["latin"], variable:"--font-space", weight:["300","400","500","600","700"] });

export const metadata: Metadata = {
  title: "IICPC 2026 — Distributed Benchmarking Platform",
  description: "Submit your exchange. 5,000 bots. Nanosecond telemetry. Adaptive chaos.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${mono.variable} ${sans.variable}`}>
      <body>
        <Nav />
        <main style={{ paddingTop:52 }}>{children}</main>
      </body>
    </html>
  );
}
