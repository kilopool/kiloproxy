# Kiloproxy
[![Github All Releases](https://img.shields.io/github/downloads/kilopool/kiloproxy/total.svg)](https://github.com/kilopool/kiloproxy/releases)
[![GitHub release](https://img.shields.io/github/release/kilopool/kiloproxy/all.svg)](https://github.com/kilopool/kiloproxy/releases)
[![GitHub Release Date](https://img.shields.io/github/release-date-pre/kilopool/kiloproxy.svg)](https://github.com/kilopool/kiloproxy/releases)
[![GitHub license](https://img.shields.io/github/license/kilopool/kiloproxy.svg)](https://github.com/kilopool/kiloproxy/blob/master/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/kilopool/kiloproxy.svg)](https://github.com/kilopool/kiloproxy/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/kilopool/kiloproxy.svg)](https://github.com/kilopool/kiloproxy/network)

Extremely high performance Monero (XMR) and Cryptonote Stratum mining protocol proxy.
Kiloproxy takes full advantage of device multithreading.
Reduces pool connections by up to 256 times, 1000 connected miners become just 4 for the pool.

## Download
https://github.com/kilopool/kiloproxy/releases

## Notes
- If you are using Linux and want to handle more than 1000 connections, you need to [increase the open files limit](ulimit.md)
- Miners MUST support Nicehash mode.
- Kiloproxy is still in beta, please report any issue.

## Donations
Kiloproxy has **0% fee by default**.
However, any donation is greatly appreciated!
`86Cyc69WoNa71qjStepJ83PKpR2PFEpviZJaxqT8cuvv1RLhJhf6aZAXkFA2btmHkXULyZ3bDu6uzJX2DuVkVeUwTN2M5g3`

## Comparison with other proxies

| Proxy          | Performance | Multithread | Default Fee | Cross-platform | Reduces pool load | 1-step setup |
|----------------|-------------|-------------|-------------|----------------|-------------------|--------------|
| Kiloproxy      | **High**    | **Yes**     | **0%**      | **Yes**        | **Yes**           | **Yes**      |
| XMRig-Proxy    | **High**    | No          | 2%          | **Yes**        | **Yes**           | No           |
| Snipa22/xmr-node-proxy | Moderate | No     | 1%          | No             | **Yes**           | No           |

## License
Kiloproxy is licensed under [GPLv3](LICENSE).

## Contact
https://kilopool.com/contact/en