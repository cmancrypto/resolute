import { COSMOSTATION, KEPLR, LEAP } from "@/constants/wallet";

export const networks: Network[] = [
  // Add Prysm Devnet
  {
    enableModules: {
      authz: true,
      feegrant: true,
      group: false,
    },
    aminoConfig: {
      authz: false,
      feegrant: false,
      group: false,
    },
    showAirdrop: false,
    logos: {
      menu: 'https://raw.githubusercontent.com/cosmos/chain-registry/master/testnets/prysmdevnet/images/prysm.png',
      toolbar: 'https://raw.githubusercontent.com/cosmos/chain-registry/master/testnets/prysmdevnet/images/prysm.svg',
    },
    supportedWallets: [KEPLR, LEAP, COSMOSTATION],
    keplrExperimental: true,
    leapExperimental: true,
    isTestnet: true,
    govV1: false,
    isCustomNetwork: false,
    explorerTxHashEndpoint: 'https://explorer.kleomedes.network/prysm/tx/',
    config: {
      chainId: 'prysm-devnet-1',
      chainName: 'PrysmDevnet',
      rest: '/prysm-api',
      rpc: '/prysm-rpc',
      restURIs: [
        '/prysm-api',
        '/prysm-polkachu-api',
        'https://prysm-testnet-api.synergynodes.com'
      ],
      rpcURIs: [
        '/prysm-rpc',
        '/prysm-polkachu-rpc',
        'https://prysm-testnet-rpc.synergynodes.com'
      ],
      currencies: [
        {
          coinDenom: 'PRYSM',
          coinMinimalDenom: 'uprysm',
          coinDecimals: 6,
        },
      ],
      bech32Config: {
        bech32PrefixAccAddr: 'prysm',
        bech32PrefixAccPub: 'prysmpub',
        bech32PrefixValAddr: 'prysmvaloper',
        bech32PrefixValPub: 'prysmvaloperpub',
        bech32PrefixConsAddr: 'prysmvalcons',
        bech32PrefixConsPub: 'prysmvalconspub',
      },
      feeCurrencies: [
        {
          coinDenom: 'PRYSM',
          coinMinimalDenom: 'uprysm',
          coinDecimals: 6,
          gasPriceStep: {
            low: 0,
            average: 0,
            high: 0,
          },
        },
      ],
      bip44: {
        coinType: 118,
      },
      stakeCurrency: {
        coinDenom: 'PRYSM',
        coinMinimalDenom: 'uprysm',
        coinDecimals: 6,
      },
      image: 'https://raw.githubusercontent.com/cosmos/chain-registry/master/testnets/prysmdevnet/images/prysm.svg',
      theme: {
        primaryColor: '#cf654f',
        gradient: 'linear-gradient(180deg, #cf654f60 0%, #12131C80 100%)',
      },
    },
  },
];
