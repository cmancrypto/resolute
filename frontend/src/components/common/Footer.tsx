import React from 'react';
import Image from 'next/image';

const Footer = () => {
  return (
    <div className="footer-bg mt-8">
      <div>
        {/* <Image
            src="resolute-logo-vitwit.svg"
            width={100}
            height={20}
            alt="resolute-logo"
          /> */}
        <p className="text-h1 !font-semibold tracking-[1.5px]">PARITYSCAN</p>
      </div>

      <div className="text-b1">
        ParityScan is a fork of the Resolute project by VitWit with added functionality using SkipGo and built to support Prysm. 
      </div>
      <div className="flex gap-10 justify-between">
        <a
          className="flex gap-2 items-center"
          href="https://t.me/+3bXmS6GE4HRjYmU1"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Image
            src="/telegram-icon.png"
            width={20}
            height={20}
            alt="telegram-logo"
            // className="w-5 h-5"
          />
          <p className="capitalize">telegram</p>
        </a>
        <div className="v-line !h-10"></div>
        <a
          className="flex gap-2 items-center"
          href="https://github.com/vitwit/resolute"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Image
            src="/github-logo.png"
            width={20}
            height={20}
            alt="github-logo"
            // className="w-6 h-6"
          />
          <p className="capitalize"> github</p>
        </a>
        <div className="v-line !h-10"></div>
        <a
          className="flex gap-2 items-center"
          href="https://twitter.com/vitwit_"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Image
            src="/twitter-icon.png"
            width={20}
            height={20}
            alt="twitter-logo"
            // className="w-6 h-6"
          />
          <p className="capitalize">twitter</p>
        </a>
        <div className="v-line !h-10"></div>
        <a
          className="flex gap-2 items-center"
          href="https://www.youtube.com/@vitwit8623"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Image
            src="/youtube-logo.png"
            width={20}
            height={24}
            alt="youtube-logo"
            className="w-6 h-6"
          />
          <p className="capitalize">youtube</p>
        </a>
      </div>
    </div>
  );
};

export default Footer;
