import dotenv from 'dotenv'

dotenv.config()

import * as types from 'hardhat/internal/core/params/argumentTypes'
import { task } from 'hardhat/config'
import {
  providers,
  utils,
  Wallet,
  ContractFactory,
  constants,
  Contract,
  BigNumber,
} from 'ethers'
import { UniswapV3Deployer } from 'uniswap-v3-deploy-plugin/dist/deployer/UniswapV3Deployer'
import { abi as NFTABI } from '@uniswap/v3-periphery/artifacts/contracts/NonfungiblePositionManager.sol/NonfungiblePositionManager.json'
import { FeeAmount, TICK_SPACINGS } from '@uniswap/v3-sdk'

import {
  die,
  fundUser,
  getAddressManager,
  getL1Bridge,
  logStderr,
  l1Provider,
  l2Provider,
} from '../test/shared/utils'
import { initWatcher } from '../test/shared/watcher-utils'
import ERC20 from '../artifacts/contracts/ERC20.sol/ERC20.json'
import ERC721 from '../artifacts/contracts/NFT.sol/NFT.json'

const writeStderr = (input: string) => {
  process.stderr.write(`${input}\n`)
}

task(
  'check-block-hashes',
  'Compares the block hashes of two different replicas.'
)
  .addPositionalParam('replicaA', 'The first replica')
  .addPositionalParam('replicaB', 'The second replica')
  .setAction(async ({ replicaA, replicaB }) => {
    const providerA = new providers.JsonRpcProvider(replicaA)
    const providerB = new providers.JsonRpcProvider(replicaB)

    let netA
    let netB
    try {
      netA = await providerA.getNetwork()
    } catch (e) {
      console.error(`Error getting network from ${replicaA}:`)
      die(e)
    }
    try {
      netB = await providerA.getNetwork()
    } catch (e) {
      console.error(`Error getting network from ${replicaB}:`)
      die(e)
    }

    if (netA.chainId !== netB.chainId) {
      die('Chain IDs do not match')
      return
    }

    logStderr('Getting block height.')
    const heightA = await providerA.getBlockNumber()
    const heightB = await providerB.getBlockNumber()
    const endHeight = Math.min(heightA, heightB)
    logStderr(`Chose block height: ${endHeight}`)

    for (let n = endHeight; n >= 1; n--) {
      const blocks = await Promise.all([
        providerA.getBlock(n),
        providerB.getBlock(n),
      ])

      const hashA = blocks[0].hash
      const hashB = blocks[1].hash
      if (hashA !== hashB) {
        console.log(`HASH MISMATCH! block=${n} a=${hashA} b=${hashB}`)
        continue
      }

      console.log(`HASHES OK! block=${n} hash=${hashA}`)
      return
    }
  })

task('fund-l1')
  .addParam('recipient', 'Recipient of the deposit on L1.', null, types.string)
  .addParam('amount', 'Amount to deposit, in Ether.', null, types.string)
  .setAction(async (args) => {
    const l1Wallet = new Wallet(process.env.PRIVATE_KEY).connect(l1Provider)
    writeStderr(`Transferring ${args.amount} ETH to ${args.recipient}...`)
    const value = utils.parseEther(args.amount)
    await l1Wallet.sendTransaction({
      to: args.recipient,
      value,
    })
    writeStderr('Done.')
  })

task('deposit-l2')
  .addParam('recipient', 'Recipient of the deposit on L2.', null, types.string)
  .addParam('amount', 'Amount to deposit, in Ether.', null, types.string)
  .setAction(async (args) => {
    const l1Wallet = new Wallet(process.env.PRIVATE_KEY).connect(l1Provider)
    const l2Wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
    const addressManager = getAddressManager(l1Wallet)
    const watcher = await initWatcher(l1Provider, l2Provider, addressManager)
    const l1Bridge = await getL1Bridge(l1Wallet, addressManager)

    const value = utils.parseEther(args.amount)
    writeStderr(`Depositing ${args.amount} ETH onto L2...`)
    await fundUser(watcher, l1Bridge, value)
    writeStderr(`Transferring funds to ${args.recipient}...`)
    if (l1Wallet.address === args.recipient) {
      writeStderr('Done.')
      return
    }
    await l2Wallet.sendTransaction({
      to: args.recipient,
      value,
    })
    writeStderr('Done.')
  })

task('balance-l2')
  .addParam('address', 'Address to get the balance for.', null, types.string)
  .setAction(async (args) => {
    const balance = await l2Provider.getBalance(args.address)
    console.log(utils.formatEther(balance))
  })

// Below methods taken from the Uniswap test suite, see
// https://github.com/Uniswap/v3-periphery/blob/main/test/shared/ticks.ts
export const getMinTick = (tickSpacing: number) =>
  Math.ceil(-887272 / tickSpacing) * tickSpacing
export const getMaxTick = (tickSpacing: number) =>
  Math.floor(887272 / tickSpacing) * tickSpacing

task('deploy-erc20')
  .addOptionalParam('name', 'Name of the ERC20.', 'OVM Test', types.string)
  .addOptionalParam('symbol', 'Symbol of the ERC20.', 'OVM', types.string)
  .addOptionalParam('decimals', 'Decimals of the ERC20.', 18, types.int)
  .addOptionalParam(
    'initialSupply',
    'Token initial supply.',
    constants.MaxUint256.toString(),
    types.string
  )
  .setAction(async (args) => {
    writeStderr(`Deploying ERC20 ${args.name}...`)
    const wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
    const factory = new ContractFactory(ERC20.abi, ERC20.bytecode, wallet)
    const token = await factory.deploy(
      args.initialSupply,
      args.name,
      args.decimals,
      args.symbol
    )
    await token.deployed()
    writeStderr(`Successfully deployed ERC20 ${args.name}.`)

    console.log(
      JSON.stringify(
        {
          address: token.address,
          name: args.name,
          symbol: args.symbol,
          decimals: args.decimals,
          initialSupply: args.initialSupply,
        },
        null,
        2
      )
    )
  })

task('deploy-erc721').setAction(async () => {
  const wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
  const factory = new ContractFactory(ERC721.abi, ERC721.bytecode, wallet)
  writeStderr('Deploying ERC-721...')
  const contract = await factory.deploy()
  await contract.deployed()
  writeStderr('Done')
  console.log(
    JSON.stringify({
      address: contract.address,
    })
  )
})

task('approve-erc20')
  .addParam('tokenAddress', 'Address of the token.', '', types.string)
  .addParam('approvingAddress', 'Address to approve.', '', types.string)
  .addParam(
    'amount',
    'Amount to approve',
    constants.MaxUint256.toString(),
    types.string
  )
  .setAction(async (args) => {
    const wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
    writeStderr(
      `Approving ${args.approvingAddress} to spend ${args.amount} from ${args.tokenAddress}...`
    )
    const contract = new Contract(args.tokenAddress, ERC20.abi).connect(wallet)
    const tx = await contract.approve(args.approvingAddress, args.amount)
    await tx.wait()
    writeStderr('Done.')
  })

task('deploy-uniswap').setAction(async () => {
  const wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
  writeStderr('Deploying Uniswap ecosystem...')
  const contracts = await UniswapV3Deployer.deploy(wallet)
  writeStderr('Done.')
  console.log(
    JSON.stringify(
      Object.entries(contracts).reduce((acc, [k, v]) => {
        acc[k] = v.address
        return acc
      }, {}),
      null,
      2
    )
  )
})

task('bootstrap-uniswap-pool')
  .addParam(
    'positionManagerAddress',
    'Address of the position manager.',
    '',
    types.string
  )
  .addParam('token0Address', 'Address of the first token', '', types.string)
  .addParam('token1Address', 'Address of the second token', '', types.string)
  .addParam(
    'initialRatio',
    'Initial price ratio.',
    BigNumber.from('79228162514264337593543950336').toString(),
    types.string
  )
  .addParam(
    'amount0',
    'Amount of the first token to put in the position',
    '1000000000',
    types.string
  )
  .addParam(
    'amount1',
    'Amount of the second token to put in the position',
    '1000000000',
    types.string
  )
  .setAction(async (args) => {
    let tokensAmounts = [
      {
        address: args.token0Address,
        amount: args.amount0,
      },
      {
        address: args.token1Address,
        amount: args.amount1,
      },
    ]

    if (tokensAmounts[0].address > tokensAmounts[1].address) {
      tokensAmounts = [tokensAmounts[1], tokensAmounts[0]]
    }

    const wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
    const positionManager = new Contract(
      args.positionManagerAddress,
      NFTABI
    ).connect(wallet)
    writeStderr('Creating pool...')
    let tx = await positionManager.createAndInitializePoolIfNecessary(
      tokensAmounts[0].address,
      tokensAmounts[1].address,
      FeeAmount.MEDIUM,
      BigNumber.from(args.initialRatio)
    )
    await tx.wait()

    writeStderr('Minting position...')
    tx = await positionManager.mint(
      {
        token0: tokensAmounts[0].address,
        token1: tokensAmounts[1].address,
        tickLower: getMinTick(TICK_SPACINGS[FeeAmount.MEDIUM]),
        tickUpper: getMaxTick(TICK_SPACINGS[FeeAmount.MEDIUM]),
        fee: FeeAmount.MEDIUM,
        recipient: wallet.address,
        amount0Desired: tokensAmounts[0].amount,
        amount1Desired: tokensAmounts[1].amount,
        amount0Min: 0,
        amount1Min: 0,
        deadline: Date.now() * 2,
      },
      {
        gasLimit: 10000000,
      }
    )
    await tx.wait()
  })
