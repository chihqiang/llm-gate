/** @type {import('next').NextConfig} */
const nextConfig = {
  output: process.env.BUILD_OUTPUT,
  trailingSlash: true,
  images: { unoptimized: true },
  skipTrailingSlashRedirect: true,
  // 开发环境下将 API 请求代理到后端，避免 CORS 问题
  async rewrites() {
    if (process.env.NODE_ENV === "production") return []
    return [
      {
        source: "/api/:path*",
        destination: `${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"}/api/:path*`,
      },
      {
        source: "/v1/:path*",
        destination: `${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"}/v1/:path*`,
      },
    ]
  },
}

export default nextConfig
