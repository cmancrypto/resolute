import { useEffect } from 'react';
import { useAppDispatch } from '../StateHooks';
import { setSelectedNetwork } from '@/store/features/common/commonSlice';
import { usePathname } from 'next/navigation';

const useHandleRouteChange = () => {
  const pathName = usePathname();
  const dispatch = useAppDispatch();

  useEffect(() => {
    // Handle case where pathName might be null
    const pathParts = pathName ? pathName.split('/') : [];
    // Default to PrysmDevnet instead of empty string (All Networks)
    const defaultNetwork = 'prysmdevnet';
    const getChainName = (index: number) => pathParts?.[index]?.toLowerCase() || '';

    if (pathParts.includes('validator')) {
      dispatch(setSelectedNetwork({ chainName: defaultNetwork }));
    } else if (pathParts.includes('feegrant') || pathParts.includes('authz')) {
      if (pathParts.length >= 4) {
        const isNewFeegrantOrAuthz = pathParts.includes('new-feegrant') || pathParts.includes('new-authz');
        dispatch(setSelectedNetwork({ chainName: isNewFeegrantOrAuthz ? defaultNetwork : getChainName(3) }));
      } else {
        dispatch(setSelectedNetwork({ chainName: defaultNetwork }));
      }
    } else if (pathParts.includes('builder') || pathParts.includes('history')) {
      dispatch(setSelectedNetwork({ chainName: pathParts.length >= 4 ? getChainName(3) : defaultNetwork }));
    } else if (pathParts.length >= 3) {
      dispatch(setSelectedNetwork({ chainName: getChainName(2) }));
    } else {
      dispatch(setSelectedNetwork({ chainName: defaultNetwork }));
    }
  }, [pathName]);
};

export default useHandleRouteChange;
