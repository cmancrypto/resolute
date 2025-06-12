import { ChainConfig } from '@/types/swaps';
import axios from 'axios';
import { useEffect, useState } from 'react';

interface ChainData {
  chainId: string;
  chainType: string;
  axelarChainName: string;
  chainIconURI?: string;
}

const useGetChains = () => {
  const [chainsInfo, setChainInfo] = useState<ChainConfig[]>([]);
  const [chainsData, setChainsData] = useState<ChainData[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchChainsInfo();
  }, []);

  const fetchChainsInfo = async () => {
    try {
      // TODO: Replace with new chain info API endpoint
      const result = await axios.get('YOUR_NEW_CHAINS_API_ENDPOINT');
      const chains: ChainData[] = result.data.chains;
      setChainsData(chains);
      const chainsData = chains
        .filter((chain) => chain.chainType === 'cosmos') // To filter cosmos chains
        .map((chain): ChainConfig => {
          return {
            label: chain.axelarChainName,
            logoURI: chain.chainIconURI || '',
            chainID: chain.chainId,
          };
        })
        .sort((chainA, chainB) => {
          return chainA.label.localeCompare(chainB.label);
        });
      setChainInfo(chainsData);
    } catch (error) {
      console.log('error while fetching data', error);
    } finally {
      setLoading(false);
    }
  };

  const getChainConfig = (chainID: string) => {
    const chainConfig = chainsData.filter((chain) => chain.chainId === chainID);
    return chainConfig[0];
  };

  const getChainLogoURI = (chainID: string) => {
    const chainConfig = getChainConfig(chainID);
    const logoURI = chainConfig?.chainIconURI || '';
    return logoURI;
  };

  return {
    loading,
    chainsInfo,
    getChainConfig,
    getChainLogoURI,
  };
};

export default useGetChains;
