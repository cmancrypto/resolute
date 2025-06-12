'use client';
import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import { getWalletAmino } from '../../../txns/execute';
import { isWalletInstalled } from './walletService';
import {
  getFeegrantMode,
  setConnected,
  setWalletName,
} from '../../../utils/localStorage';
import { TxStatus } from '@/types/enums';
import { setError } from '../common/commonSlice';
import { getKey } from '@leapwallet/cosmos-snap-provider';
import { getAddressByPrefix } from '@/utils/address';
import { getAuthzMode } from '@/utils/localStorage';
import { enableAuthzMode } from '../authz/authzSlice';
import { enableFeegrantMode } from '../feegrant/feegrantSlice';
import { NotSupportedMetamaskChainIds } from '@/utils/constants';

declare let window: WalletWindow;

interface ChainInfo {
  walletInfo: {
    name: string;
    isNanoLedger: boolean;
    pubKey: string;
    bech32Address: string;
    isKeystone: string;
    algo: string;
    address?: string;
  };
  network: Network;
}

interface WalletState {
  name: string;
  connectWalletOpen: boolean;
  connected: boolean;
  isLoading: boolean;
  isNanoLedger: boolean;
  pubKey: string;
  networks: Record<string, ChainInfo>;
  nameToChainIDs: Record<string, string>;
  status: TxStatus;
}

const initialState: WalletState = {
  name: '',
  connectWalletOpen: false,
  connected: false,
  isLoading: true,
  isNanoLedger: false,
  pubKey: '',
  networks: {},
  nameToChainIDs: {},
  status: TxStatus.INIT,
};

export const establishWalletConnection = createAsyncThunk(
  'wallet/connect',
  async (
    data: {
      networks: Network[];
      walletName: string;
    },
    { rejectWithValue, fulfillWithValue, dispatch }
  ) => {
    const networks = data.networks;
    if (!isWalletInstalled(data.walletName)) {
      dispatch(setError({ type: 'error', message: 'Wallet is not installed' }));
      return rejectWithValue('wallet is not installed');
    } else {
      window.wallet.defaultOptions = {
        sign: {
          preferNoSetMemo: true,
          preferNoSetFee: false,
          disableBalanceCheck: true,
        },
      };
      const chainIDs: string[] = networks.map(
        (mainnet) => mainnet.config.chainId
      );
      try {
        console.log('Attempting to enable chains:', chainIDs);
        await window.wallet.enable(chainIDs);
      } catch (error) {
        console.log('Error enabling chains:', error);
      }

      let walletName = '';
      let isNanoLedger = false;
      const chainInfos: Record<string, ChainInfo> = {};
      const nameToChainIDs: Record<string, string> = {};
      let anyNetworkAddress = '';

      for (let i = 0; i < networks.length; i++) {
        const chainId = networks[i].config.chainId;
        try {
          if (!networks[i].config.rpc || !networks[i].config.rpc.startsWith('http')) {
            console.warn(`Invalid RPC URL for ${networks[i].config.chainName}: ${networks[i].config.rpc}`);
            continue;
          }
          
          if (
            (data.walletName === 'keplr' ||
              data.walletName === 'cosmostation') &&
            networks[i].keplrExperimental
          ) {
            console.log(`Suggesting experimental chain for ${data.walletName}:`, networks[i].config);
            await window.wallet.experimentalSuggestChain(networks[i].config);
          }
          if (data.walletName === 'leap' && networks[i].leapExperimental) {
            console.log(`Suggesting experimental chain for leap:`, networks[i].config);
            await window.wallet.experimentalSuggestChain(networks[i].config);
          }
          await getWalletAmino(chainId);
          const walletInfo = await window.wallet.getKey(chainId);
          
          // Ensure all walletInfo fields are serializable
          const serializedWalletInfo = {
            name: walletInfo?.name || '',
            isNanoLedger: walletInfo?.isNanoLedger || false,
            pubKey: Buffer.from(walletInfo?.pubKey).toString('base64'),
            bech32Address: walletInfo?.bech32Address || '',
            isKeystone: walletInfo?.isKeystone || '',
            algo: walletInfo?.algo || '',
            // Handle potential Uint8Array address field
            address: (walletInfo?.address && walletInfo.address.constructor?.name === 'Uint8Array')
              ? Buffer.from(walletInfo.address).toString('hex')
              : walletInfo?.address || ''
          };
          
          walletName = serializedWalletInfo.name;
          isNanoLedger = serializedWalletInfo.isNanoLedger;
          chainInfos[chainId] = {
            walletInfo: serializedWalletInfo,
            network: networks[i],
          };
          if (anyNetworkAddress === '')
            anyNetworkAddress = serializedWalletInfo.bech32Address || '';
          nameToChainIDs[
            networks[i].config.chainName.toLowerCase().split(' ').join('')
          ] = chainId;
        } catch (error) {
          console.log(
            `unable to connect to network ${networks[i].config.chainName}: `,
            error
          );
        }
      }

      if (Object.keys(chainInfos).length === 0) {
        dispatch(
          setError({
            type: 'error',
            message: 'Permission denied for all the networks',
          })
        );
        return rejectWithValue('Permission denied for all the networks');
      } else {
        setConnected();
        setWalletName(data.walletName);

        const cosmosAddress = getAddressByPrefix(anyNetworkAddress, 'cosmos');
        const authzMode = getAuthzMode(cosmosAddress);
        if (authzMode.isAuthzModeOn)
          dispatch(enableAuthzMode({ address: authzMode.authzAddress }));
        const feegrantMode = getFeegrantMode(cosmosAddress);
        if (feegrantMode.isFeegrantModeOn)
          dispatch(
            enableFeegrantMode({ address: feegrantMode.feegrantAddress })
          );

        return fulfillWithValue({
          chainInfos,
          nameToChainIDs,
          walletName,
          isNanoLedger,
        });
      }
    }
  }
);

export const establishMetamaskConnection = createAsyncThunk(
  'wallet/metamask-connection',
  async (
    data: {
      network: Network;
      walletName: string;
    },
    { rejectWithValue, dispatch }
  ) => {
    if (!isWalletInstalled(data.walletName)) {
      dispatch(setError({ type: 'error', message: 'Wallet is not installed' }));
      return rejectWithValue('wallet is not installed');
    } else {
      window.wallet.defaultOptions = {
        sign: {
          preferNoSetMemo: true,
          preferNoSetFee: false,
          disableBalanceCheck: true,
        },
      };
      const chainId = data.network.config.chainId;
      try {
        await window.wallet.enable(chainId);
      } catch (error) {
        console.log('caught', error);
      }

      try {
        if (NotSupportedMetamaskChainIds.indexOf(chainId) === -1) {
          const walletInfo = await getKey(chainId);
          
          // Ensure all walletInfo fields are serializable for MetaMask
          const serializedWalletInfo = {
            algo: walletInfo?.algo || '',
            bech32Address: walletInfo?.address || '',
            pubKey: Buffer.from(walletInfo?.pubkey).toString('base64'),
            isKeystone: '',
            isNanoLedger: false,
            name: walletInfo?.address || '',
            // Handle potential Uint8Array address field
            address: (walletInfo?.address && walletInfo.address.constructor?.name === 'Uint8Array')
              ? Buffer.from(walletInfo.address).toString('hex')
              : walletInfo?.address || ''
          };
          
          const chainInfo: ChainInfo = {
            walletInfo: serializedWalletInfo,
            network: data.network,
          };

          setConnected();
          setWalletName(data.walletName);
          dispatch(addChainInfo({ chainId, chainInfo }));
          dispatch(
            addNameToChainIDs({
              chainName: data.network.config.chainName
                .toLowerCase()
                .split(' ')
                .join(''),
              chainId,
            })
          );
        }
      } catch (error) {
        console.log(
          `unable to connect to network ${data.network.config.chainName}: `,
          error
        );
      }
    }
  }
);

const walletSlice = createSlice({
  name: 'wallet',
  initialState,
  reducers: {
    setWallet: (
      state
      // action: PayloadAction<{ address: string; chainInfo: any }>
    ) => {
      state.connected = true;
    },
    resetWallet: (state) => {
      state.connected = false;
      state.name = '';
      state.pubKey = '';
      state.nameToChainIDs = {};
      state.networks = {};
      state.status = TxStatus.INIT;
    },
    resetConnectWalletStatus: (state) => {
      state.status = TxStatus.INIT;
    },
    setIsLoading: (state) => {
      state.isLoading = true;
    },
    unsetIsLoading: (state) => {
      state.isLoading = false;
    },
    setConnectWalletOpen: (state, action: PayloadAction<boolean>) => {
      state.connectWalletOpen = action.payload;
    },
    addChainInfo: (
      state,
      action: PayloadAction<{ chainId: string; chainInfo: ChainInfo }>
    ) => {
      const { chainId, chainInfo } = action.payload;
      state.networks = { ...state.networks, [chainId]: chainInfo };
      state.connected = true;
      state.isLoading = false;
      state.status = TxStatus.IDLE;
      if (!state.name?.length) {
        const cosmosAddress = getAddressByPrefix(
          chainInfo.walletInfo.name || '',
          'cosmos'
        );
        state.name = cosmosAddress;
      }
    },
    addNameToChainIDs: (
      state,
      action: PayloadAction<{ chainName: string; chainId: string }>
    ) => {
      const { chainName, chainId } = action.payload;
      state.nameToChainIDs = { ...state.nameToChainIDs, [chainName]: chainId };
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(establishWalletConnection.pending, (state) => {
        state.status = TxStatus.PENDING;
        state.isLoading = true;
      })
      .addCase(establishWalletConnection.fulfilled, (state, action) => {
        if (!action.payload) {
          state.connected = true;
          state.status = TxStatus.IDLE;
          state.isLoading = false;
          return;
        }
        const networks = action.payload.chainInfos;
        const nameToChainIDs = action.payload.nameToChainIDs;
        state.networks = networks;
        state.nameToChainIDs = nameToChainIDs;
        state.connected = true;
        state.isNanoLedger = action.payload.isNanoLedger;
        state.name = action.payload.walletName;
        state.status = TxStatus.IDLE;
        state.isLoading = false;
      })
      .addCase(establishWalletConnection.rejected, (state) => {
        state.status = TxStatus.REJECTED;
        state.isLoading = false;
      });
  },
});

export const {
  setWallet,
  resetWallet,
  resetConnectWalletStatus,
  unsetIsLoading,
  setConnectWalletOpen,
  setIsLoading,
  addChainInfo,
  addNameToChainIDs,
} = walletSlice.actions;

export default walletSlice.reducer;
