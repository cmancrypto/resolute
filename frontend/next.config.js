/** @type {import('next').NextConfig} */
const nextConfig = {
  eslint: {
    ignoreDuringBuilds: true,
  },
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**',
      },
      {
        protocol: 'https',
        hostname: 'raw.githubusercontent.com',
        port: '',
        pathname: '/vitwit/**',
      },
      {
        protocol: 'https',
        hostname: 'dummyimage.com',
        port: '',
        pathname: '/**',
      },
      {
        protocol: 'https',
        hostname: 'raw.githubusercontent.com',
        port: '',
        pathname: '/cosmos/**',
      },
      {
        protocol: 'https',
        hostname: 'resolute.sgp1.cdn.digitaloceanspaces.com',
        port: '',
        pathname: '/**',
      },
    ],
  },
  
  // Add rewrites for API proxy to avoid CORS issues
  async rewrites() {
    return [
      {
        source: '/prysm-api/:path*',
        destination: 'https://prysm-api-devnet.kleomedes.network/:path*',
      },
      {
        source: '/prysm-rpc/:path*',
        destination: 'https://prysm-rpc-devnet.kleomedes.network/:path*',
      },
      // Additional fallback proxies for other Prysm providers
      {
        source: '/prysm-polkachu-api/:path*',
        destination: 'https://prysm-testnet-api.polkachu.com/:path*',
      },
      {
        source: '/prysm-polkachu-rpc/:path*',
        destination: 'https://prysm-testnet-rpc.polkachu.com/:path*',
      },
      // Proxy for Keybase API
      {
        source: '/keybase-api/:path*',
        destination: 'https://keybase.io/_/api/1.0/:path*',
      }
    ];
  },
};

const withBundleAnalyzer = require('@next/bundle-analyzer')({
  enabled: process.env.ANALYZE === 'true',
});

module.exports = withBundleAnalyzer(nextConfig);

