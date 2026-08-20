import type { Metadata } from "next";
import "./globals.css";
import { Providers } from "../components/providers";

export const metadata: Metadata = {
  title: "ApiSentinel — Developer Security & Integration Platform",
  description: "Detect. Decide. Prevent. Test. Unified runtime webhook and API security platform.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-background font-sans antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
