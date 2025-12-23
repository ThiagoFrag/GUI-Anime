import type React from "react"
import type { Metadata } from "next"
import { Permanent_Marker, Noto_Sans_JP } from "next/font/google"
import { Analytics } from "@vercel/analytics/next"
import "./globals.css"

const permanentMarker = Permanent_Marker({
  weight: "400",
  subsets: ["latin"],
  variable: "--font-marker",
})

const notoSansJP = Noto_Sans_JP({
  subsets: ["latin"],
  weight: ["400", "700", "900"],
  variable: "--font-jp",
})

export const metadata: Metadata = {
  title: "アニメスケッチ - Anime Sketch",
  description: "Discover the world of anime with our interactive sketch-style platform",
  generator: "v0.app",
  icons: {
    icon: [
      {
        url: "/icon-light-32x32.png",
        media: "(prefers-color-scheme: light)",
      },
      {
        url: "/icon-dark-32x32.png",
        media: "(prefers-color-scheme: dark)",
      },
      {
        url: "/icon.svg",
        type: "image/svg+xml",
      },
    ],
    apple: "/apple-icon.png",
  },
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="pt-BR">
      <body className={`${permanentMarker.variable} ${notoSansJP.variable} font-sans antialiased`}>
        {children}
        <Analytics />
      </body>
    </html>
  )
}
