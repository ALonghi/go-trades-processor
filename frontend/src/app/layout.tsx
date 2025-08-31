import Link from "next/link";
import "./globals.css";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Liquid Glass Holdings",
  description: "Kafka → Go → Postgres → Next.js (liquid glass UI)",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        {/* liquid glass background */}
        <div className="liquid">
          <div className="blob b1" />
          <div className="blob b2" />
          <div className="blob b3" />
        </div>

        <header className="sticky top-0 z-20 container">
          <nav className="glass mx-auto mt-4  sm:w-auto sm:rounded-2xl mx-auto">
            <div className="px-4 sm:px-6 py-3 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-white/20 ring-1 ring-white/20">
                  ⚡
                </span>
                <span className="text-sm font-semibold tracking-tight mx-auto">
                  Investments Holdings
                </span>
              </div>
              <div className="flex items-center gap-4 text-sm">
                <Link href="/" className="text-white/80 hover:text-white">
                  Dashboard
                </Link>
                <Link href="/trades" className="text-white/80 hover:text-white">
                  Trades
                </Link>
              </div>
            </div>
          </nav>
        </header>

        <main className="container py-8 space-y-6">{children}</main>
      </body>
    </html>
  );
}
