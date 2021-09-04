/**
 * Predeploys are Solidity contracts that are injected into the initial L2 state and provide
 * various useful functions.
 *
 * Notes:
 * 0x42...04 was the address of the OVM_ProxySequencerEntrypoint. This contract is no longer in
 * use and has therefore been removed. We may place a new predeployed contract at this address
 * in the future. See https://github.com/ethereum-optimism/optimism/pull/549 for more info.
 */
export const predeploys = {
  OVM_L2ToL1MessagePasser: '0x4200000000000000000000000000000000000000',
  OVM_L1MessageSender: '0x4200000000000000000000000000000000000001',
  OVM_DeployerWhitelist: '0x4200000000000000000000000000000000000002',
  OVM_ETH: '0x4200000000000000000000000000000000000006',
  OVM_L2CrossDomainMessenger: '0x4200000000000000000000000000000000000007',
  OVM_GasPriceOracle: '0x420000000000000000000000000000000000000F',
  OVM_L2StandardBridge: '0x4200000000000000000000000000000000000010',
  OVM_SequencerFeeVault: '0x4200000000000000000000000000000000000011',
  OVM_GodKeyEntrypoint: '0x4200000000000000000000000000000000000012',
}
