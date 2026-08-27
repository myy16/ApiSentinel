/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  transpilePackages: ["@apisentinel/shared"],
};

export default nextConfig;
