import React, { useState, useEffect } from 'react';
import { Widget } from '@skip-go/widget';
import { useAppSelector } from '@/custom-hooks/StateHooks';
import { getChainIdFromName } from '@/components/main-layout/SelectNetwork';

const SkipTransfer = () => {
  const selectedChain = useAppSelector((state) => state.common.selectedNetwork.chainName);
  const isTestnet = useAppSelector((state) => state.common.selectedNetwork.isTestnet);
  const nameToChainIDs = useAppSelector((state) => state.common.nameToChainIDs);
  
  // Get chainId directly from state, or derive it from chainName if not available
  const chainIdFromState = useAppSelector((state) => state.common.selectedNetwork.chainId);
  const derivedChainId = selectedChain ? getChainIdFromName(nameToChainIDs, selectedChain) : undefined;
  
  // Use the chainId from state if available, otherwise fall back to the derived value
  const chainId = chainIdFromState || derivedChainId;

  // Custom theme configuration to match Resolute styling
  const resoluteTheme = {
    brandColor: 'linear-gradient(48deg, rgb(255 255 255 / 3%) 0%, rgb(153 153 153 / 25%) 100%)', // Resolute's primary colours
    primary: {
      background: {
        normal: '#12131C', // Gradient background
      },
      text: {
        normal: '#FFFFFF', // White text on primary elements
        lowContrast: '#FFFFFFCC', // Slightly transparent white
        ultraLowContrast: '#FFFFFF80', // More transparent white
      },
      ghostButtonHover: '#6155B240', // Lighter purple for hover states
    },
    secondary: {
      background: {
        normal: 'linear-gradient(180deg, rgba(68, 83, 223, 0.1) 12.5%, rgba(127, 92, 237, 0.1) 100%)', // Gradient for secondary elements
        transparent: 'rgba(18, 19, 28, 0.8)', // Semi-transparent dark background
        hover: '#1D1E2C', // Slightly lighter on hover
      },
    },
    success: {
      text: '#00C853', // Green for success messages
    },
    warning: {
      background: '#FFC10780', // Yellow with transparency for warnings
      text: '#FFC107', // Yellow text for warnings
    },
    error: {
      background: '#FF5252', // Red background for errors
      text: '#FFFFFF', // White text on error backgrounds
    },
  };

  console.log('chainid', chainId);

  return (
    <div className="flex flex-col items-center w-full max-w-[500px] mx-auto">
      <div
        className="bg-card rounded-xl p-6 w-full"
        style={{
          boxShadow: '0px 4px 20px rgba(0, 0, 0, 0.05)',
        }}
      >
        <Widget
          theme={resoluteTheme}

          onlyTestnet={isTestnet}
        />
      </div>
    </div>
  );
};

export default SkipTransfer; 