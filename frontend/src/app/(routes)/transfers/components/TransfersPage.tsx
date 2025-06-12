import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '@/custom-hooks/StateHooks';
import useSortedAssets from '@/custom-hooks/useSortedAssets';
import { useSearchParams } from 'next/navigation';
import MultiSendPage from './multi-send/MultiSendPage';
import SkipTransferPage from './skip-transfers/SkipTransferPage';
import { setConnectWalletOpen } from '@/store/features/wallet/walletSlice';
import EmptyScreen from '@/components/common/EmptyScreen';
import PageHeader from '@/components/common/PageHeader';
import { TRANSFERS_TYPES } from '@/utils/constants';
import SingleSend from './single-send/SingleSend';
import useGetShowAuthzAlert from '@/custom-hooks/useGetShowAuthzAlert';

const TransfersPage = ({ chainIDs }: { chainIDs: string[] }) => {
  const [sortedAssets, authzSortedAssets] = useSortedAssets(chainIDs, {
    showAvailable: true,
    AuthzSkipIBC: true,
  });
  const paramsTransferType = useSearchParams()?.get('type') || 'single';

  const [transferType, setTransferType] = useState('single');

  const isAuthzMode = useAppSelector((state) => state.authz.authzModeEnabled);

  const dispatch = useAppDispatch();

  const isWalletConnected = useAppSelector((state) => state.wallet.connected);
  const showAuthzAlert = useGetShowAuthzAlert();


  const connectWalletOpen = () => {
    dispatch(setConnectWalletOpen(true));
  };

  useEffect(() => {
    if (paramsTransferType?.length) {
      setTransferType(paramsTransferType.toLowerCase());
    } else {
      setTransferType('single');
    }
  }, [paramsTransferType]);

  return (
    <div
      className={`space-y-10 flex flex-col py-10 ${showAuthzAlert ? 'min-h-[calc(100vh-118px)]' : 'min-h-[calc(100vh-64px)]'}`}
    >
      <PageHeader
        title={TRANSFERS_TYPES?.[transferType].title}
        description={TRANSFERS_TYPES?.[transferType].description}
      />
      {isWalletConnected ? (
        <div className="flex-1">
          {transferType === 'single' ? (
            <SingleSend
              sortedAssets={isAuthzMode ? authzSortedAssets : sortedAssets}
            />
          ) : null}
          {transferType === 'multi-send' ? (
            <MultiSendPage chainID={chainIDs[0]} />
          ) : null}
          {transferType === 'skip' ? <SkipTransferPage /> : null}
        </div>
      ) : (
        <div className="flex-1 flex items-center justify-center min-h-[80vh]">
          <EmptyScreen
            title="Connect your wallet"
            description="Connect your wallet to access your account on Resolute"
            hasActionBtn={true}
            btnText={'Connect Wallet'}
            btnOnClick={connectWalletOpen}
          />
        </div>
      )}
    </div>
  );
};

export default TransfersPage;
