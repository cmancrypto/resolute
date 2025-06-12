import { useState } from 'react';
import axios from 'axios';
import { cleanURL } from '@/utils/util';

interface TokenData {
  symbol?: string;
  ibcDenom?: string;
  logoURI?: string;
  decimals?: number;
  name?: string;
}

interface AssetConfig {
  label: string;
  symbol: string;
  logoURI: string;
  denom: string;
  decimals: number;
  name: string;
}

const useGetAssets = () => {
  const [srcAssetsLoading, setSrcAssetsLoading] = useState(false);
  const [destAssetLoading, setDestAssetsLoading] = useState(false);

  const fetchAssetsInfo = async (chainID: string, isSource: boolean) => {
    try {
      if (isSource) {
        setSrcAssetsLoading(true);
      } else {
        setDestAssetsLoading(true);
      }
      // TODO: Replace with new assets API endpoint
      const result = await axios.get(
        `YOUR_NEW_ASSETS_API_ENDPOINT?chainId=${chainID}`
      );
      const assets: TokenData[] = result.data.tokens;
      return assets;
    } catch (error) {
      console.log('error while fetching data', error);
    } finally {
      if (isSource) {
        setSrcAssetsLoading(false);
      } else {
        setDestAssetsLoading(false);
      }
    }
  };

  const getTokensByChainID = async (chainID: string, isSource: boolean) => {
    if (!chainID?.length) return [];
    const assets = await fetchAssetsInfo(chainID, isSource);
    const formattedAssets = assets ? getFormattedAssetsList(assets) : [];
    return formattedAssets;
  };
  return {
    getTokensByChainID,
    srcAssetsLoading,
    destAssetLoading,
  };
};

const getFormattedAssetsList = (data: TokenData[]): AssetConfig[] => {
  const assetsList = data
    .map((asset): AssetConfig => {
      return {
        symbol: asset.symbol || '',
        label: asset.ibcDenom || '',
        logoURI: asset.logoURI || '',
        denom: asset.ibcDenom || '',
        decimals: asset.decimals || 0,
        name: asset.name || '',
      };
    })
    .sort((assetA, assetB) => {
      return assetA.label.localeCompare(assetB.label);
    });
  return assetsList;
};

export default useGetAssets;
