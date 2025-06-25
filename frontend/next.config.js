/** @type {import('next').NextConfig} */
const nextConfig = {
  // Enable standalone output for optimized Docker builds
  output: 'standalone',
  
  eslint: {
    ignoreDuringBuilds: true,
  },
  
  // Performance optimizations
  swcMinify: true, // Use SWC for faster minification
  reactStrictMode: true,
  poweredByHeader: false,
  
  // Optimize images
  images: {
    // Add image optimization settings
    formats: ['image/webp', 'image/avif'],
    deviceSizes: [640, 750, 828, 1080, 1200, 1920, 2048, 3840],
    imageSizes: [16, 32, 48, 64, 96, 128, 256, 384],
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
  
  // Add webpack optimizations
  webpack: (config, { dev, isServer }) => {
    // Optimize production builds
    if (!dev && !isServer) {
      config.optimization = {
        ...config.optimization,
        splitChunks: {
          chunks: 'all',
          minSize: 20000,
          maxSize: 244000,
          minChunks: 1,
          maxAsyncRequests: 30,
          maxInitialRequests: 30,
          cacheGroups: {
            defaultVendors: {
              test: /[\\/]node_modules[\\/]/,
              priority: -10,
              reuseExistingChunk: true,
            },
            default: {
              minChunks: 2,
              priority: -20,
              reuseExistingChunk: true,
            },
          },
        },
      };
    }
    return config;
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
      },
      // You can add a fallback rewrite if needed
      // {
      //   source: '/:path*',
      //   destination: `${process.env.FALLBACK_URL || 'https://your-default-api.com'}/:path*`,
      //   basePath: false
      // }
    ];
  },
};

const withBundleAnalyzer = require('@next/bundle-analyzer')({
  enabled: process.env.ANALYZE === 'true',
});

module.exports = withBundleAnalyzer(nextConfig);

