# Install deps and build
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* pnpm-lock.yaml* yarn.lock* ./
RUN npm ci --silent || (npm install -g pnpm && pnpm i --frozen-lockfile) || yarn install --frozen-lockfile
COPY . .
RUN npm run build || pnpm build || yarn build

# Runtime
FROM node:22-alpine
ENV NODE_ENV=production
WORKDIR /app
COPY --from=builder /app .
EXPOSE 3000
CMD ["npm", "run", "start"]
