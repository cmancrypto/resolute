import Axios from 'axios';
import { cleanURL } from './util';

export const axiosGetRequestWrapper = async (
  baseURIs: string[],
  endPoint: string
) => {
  let errMsg = '';

  try {
    // Remove chainID param from the request URL to use direct Cosmos endpoints
    const cleanedEndpoint = endPoint.replace(/[?&]chain=[\w-]+(&|$)/, (match, p1) => p1 ? p1 : '');
    const uri = `${cleanURL(baseURIs[0])}${cleanedEndpoint}`;
    return await Axios.get(uri);
    /* eslint-disable @typescript-eslint/no-explicit-any */
  } catch (err: any) {
    errMsg = err.message;
  }

  throw new Error(errMsg);
};
