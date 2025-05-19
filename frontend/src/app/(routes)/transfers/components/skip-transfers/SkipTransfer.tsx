import React, { useState, useEffect } from 'react';
import { Widget } from '@skip-go/widget';
import { useAppSelector } from '@/custom-hooks/StateHooks';

const SkipTransfer = () => {
  const selectedChain = useAppSelector((state) => state.common.selectedNetwork.chainName);
  const [srcChainId, setSrcChainId] = useState<string | undefined>(undefined);
  
  useEffect(() => {
    if (selectedChain) {
      setSrcChainId(selectedChain);
    }
  }, [selectedChain]);

  // Custom theme configuration to match Resolute styling
  const resoluteTheme = {
    brandColor: '#6155B2', // Resolute's primary purple
    primary: {
      background: {
        normal: '#6155B2', // Primary button background
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
        normal: '#12131C', // Dark background
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
          defaultRoute={{
            srcChainId: srcChainId
          }}
        />
      </div>
    </div>
  );
};

export default SkipTransfer; 