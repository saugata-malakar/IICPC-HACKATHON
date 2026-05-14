"use client";
import { motion } from "framer-motion";
import Link from "next/link";
import { usePathname } from "next/navigation";

const LINKS = [
  { href:"/leaderboard", label:"LEADERBOARD" },
  { href:"/submit",      label:"SUBMIT"      },
  { href:"/analytics",   label:"ANALYTICS"   },
];

export function Nav() {
  const path = usePathname();
  return (
    <motion.nav initial={{y:-40,opacity:0}} animate={{y:0,opacity:1}}
      style={{
        position:"fixed", top:0, left:0, right:0, height:52, zIndex:100,
        background:"var(--color-background-primary)",
        borderBottom:"0.5px solid var(--color-border-tertiary)",
        display:"flex", alignItems:"center", padding:"0 1.5rem",
        backdropFilter:"blur(8px)", fontFamily:"var(--font-mono)",
      }}>
      {/* Logo */}
      <Link href="/" style={{ display:"flex", alignItems:"center", gap:8, marginRight:24, textDecoration:"none" }}>
        <div style={{
          width:24, height:24, borderRadius:6,
          background:"var(--color-background-info)",
          display:"flex", alignItems:"center", justifyContent:"center",
          fontSize:10, fontWeight:700, color:"var(--color-text-info)",
        }}>IC</div>
        <span style={{ fontSize:12, fontWeight:500, color:"var(--color-text-primary)", letterSpacing:".04em" }}>
          IICPC<span style={{ color:"var(--color-text-info)" }}>26</span>
        </span>
      </Link>

      {/* Links */}
      <div style={{ display:"flex", gap:2 }}>
        {LINKS.map(l => {
          const active = path.startsWith(l.href);
          return (
            <Link key={l.href} href={l.href} style={{ textDecoration:"none" }}>
              <motion.div whileHover={{ background:"var(--color-background-secondary)" }}
                style={{
                  position:"relative", padding:"5px 10px",
                  borderRadius:"var(--border-radius-md)",
                  fontSize:10, letterSpacing:".1em",
                  color: active ? "var(--color-text-info)" : "var(--color-text-tertiary)",
                  transition:"color .15s",
                }}>
                {l.label}
                {active && (
                  <motion.div layoutId="nav-bar"
                    style={{
                      position:"absolute", bottom:0, left:6, right:6,
                      height:1, background:"var(--color-background-info)",
                      borderRadius:1,
                    }}
                    transition={{ type:"spring", stiffness:400, damping:30 }}
                  />
                )}
              </motion.div>
            </Link>
          );
        })}
      </div>

      {/* Right: live indicator */}
      <div style={{ marginLeft:"auto", display:"flex", alignItems:"center", gap:12 }}>
        <motion.div
          animate={{ opacity:[1,.3,1] }} transition={{ duration:2, repeat:Infinity }}
          style={{ display:"flex", alignItems:"center", gap:5, fontSize:10, color:"var(--color-text-success)" }}>
          <div style={{
            width:6, height:6, borderRadius:"50%",
            background:"var(--color-background-success)",
          }}/>
          LIVE
        </motion.div>
        <Link href="/submit">
          <motion.button whileHover={{scale:1.03}} whileTap={{scale:.97}}
            style={{
              padding:"6px 14px", borderRadius:"var(--border-radius-md)",
              fontSize:10, fontWeight:500, letterSpacing:".08em",
              border:"0.5px solid var(--color-border-info)",
              background:"var(--color-background-info)",
              color:"var(--color-text-info)", cursor:"pointer",
            }}>
            SUBMIT
          </motion.button>
        </Link>
      </div>
    </motion.nav>
  );
}
