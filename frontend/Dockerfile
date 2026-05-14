# ─────────────────────────────────────────────────────────────────────────────
# Multi-stage build for Next.js Frontend
# ─────────────────────────────────────────────────────────────────────────────

FROM node:20-alpine AS base

# ─── Dependencies ─────────────────────────────────────────────────────────────
FROM base AS deps

WORKDIR /app

# Install dependencies
COPY package.json ./
RUN npm install

# ─── Builder ──────────────────────────────────────────────────────────────────
FROM base AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Build Next.js app
ENV NEXT_TELEMETRY_DISABLED 1
RUN mkdir -p public
RUN npm run build

# ─── Runner ───────────────────────────────────────────────────────────────────
FROM base AS runner

WORKDIR /app

ENV NODE_ENV production
ENV NEXT_TELEMETRY_DISABLED 1

# Create non-root user
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Copy built application
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT 3000
ENV HOSTNAME "0.0.0.0"

CMD ["node", "server.js"]
