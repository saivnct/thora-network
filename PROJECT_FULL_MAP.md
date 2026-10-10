# THORA NETWORK — Project Full Map

> **Purpose of this file.** A fast-onboarding architecture and code map for humans and AI coding assistants
> working on `thora-network` (a Go-Ethereum fork running a Proof-of-Authority chain) and its sibling repos.
> Read this first, then jump to the referenced files.
>
> **Last surveyed:** 2026-10-10 against `master @ 51848760a` ("change mainnet genesis", 2024-04-18).
> Keep this file updated when you change consensus, genesis, params, or the deployment flow (see §12).
> **Read the Development Rules below before touching any code.**

---

## ⚠ Development Rules (MANDATORY for every developer and AI assistant)

### Workflow
1. **Understand before you build.** Before starting any feature or task, make sure you understand the business requirements and the impact of the change on the existing code base: consensus safety, state determinism, genesis/forkid, P2P, RPC compatibility, and the sibling repos.
2. **Brainstorm, discuss, interview.** Discuss the task with the project owner and interview them until the requirements and the implementation plan are clear. Write the plan as a plan file (e.g. `plans/<topic>.md`) for review.
3. **Get explicit confirmation.** Do **not** implement or modify code until the project owner has approved the plan. Approval for one task does not carry over to the next.
4. **Keep this map current.** Every change to the code base must update `PROJECT_FULL_MAP.md` in the same commit or PR: the affected sections, §11 (risks), §14 (decisions) and §15 (checkpoint).

### Code quality
5. **No hardcoded values.** Put chain parameters, addresses, amounts, timeouts, names and network identifiers in named constants or config: `params/` (`ThoraConfig`, `PlatformChainInfo`), CLI flags or TOML config. Never use magic numbers or literal addresses inside logic.
6. **Follow Go and Ethereum standards.** Use idiomatic Go (Effective Go, `gofmt`/`goimports`, `go vet`, upstream geth's lint config `build/ci.go lint`), wrap errors and don't ignore them, and avoid global mutable state. Follow Ethereum specs and EIPs plus upstream geth conventions: engine interfaces, `params.ChainConfig` fork gating, RLP/JSON codecs generated with `gencodec`. Consensus code must be deterministic: no wall-clock, map-iteration-order or float logic in state transitions.
7. **Follow SOLID and proven design patterns.** Small, single-purpose types. Extend through interfaces such as `consensus.Engine` rather than editing upstream code. Inject dependencies (config, DB, chain reader) instead of reaching for globals. Prefer composition and adapters over copy-paste. Keep the fork delta vs upstream minimal and isolated (new files/packages over inline edits) so upstream rebases stay cheap.
8. **Tests are mandatory.** Every new feature and every bug fix comes with unit tests, plus regression tests for bugs. Consensus changes also need multi-block or reorg scenario tests. Before committing, the existing suites must pass: at minimum `go test ./consensus/... ./core/... ./params/... ./cmd/thora/... ./miner/... ./eth/...`, ideally `make test`.

---

## 0. TL;DR

| Item | Value |
|---|---|
| What | Public EVM chain, fork of **go-ethereum v1.12.1-unstable** (forked at upstream commit `b058cf454`, 2023-07-15) |
| Binary | `thora` (was `geth`); source in `cmd/thora/` |
| Go module path | still `github.com/ethereum/go-ethereum` (not renamed — keeps upstream imports working) |
| Consensus | `consensus/thora` = **Clique PoA clone + block reward** (wrapped in `beacon.New(...)`) |
| Validator set | Clique-style: genesis `extraData` + on-chain header votes (`thora.propose`). **Not** controlled by the governance contract |
| Block reward | `BlockReward` (100 THA) minted each block via `state.AddBalance` to `RewardRecipient` (the **ThoraGovernance** contract) — or to the parent block signer if no recipient is set |
| Governance contract | `ThoraGovernance` (Solidity, sibling repo `../thora-scontracts`), pre-deployed in genesis at `0x…0088` (mainnet) / `0x…0089` (testnet). A role-gated vault: ADMIN/OPERATOR can `transfer()` the accumulated rewards |
| Mainnet | "Thora Network", chainId/networkId **686868**, coin **THA**, period 9s, epoch 30000, genesis `0xb5b81a33…449c` |
| Testnet | "Oda Test Network" (`--oda`), chainId **696969**, genesis `0xbced0bac…989e` |
| EVM level | **London** (all forks up to GrayGlacier at block 0). Shanghai/Cancun are **rejected** by the engine → no `PUSH0`; contracts must be compiled with `--evm-version paris` (or lower) |
| Build | `make thora` / `make all`; Docker `golang:1.20-alpine`. **Does not link with Go ≥ 1.23** (`fjl/memsize` → `runtime.stopTheWorld`) |
| P2P | Custom ENR entry `thora=v1.0.0`; dialer/discovery only keep peers advertising it |

---

## 1. Workspace and repositories

All live under `/home/giangbb/gbb/geth/`:

| Repo | Remote | Role |
|---|---|---|
| **thora-network** (this repo) | — | The node client (Geth fork) |
| **thora-scontracts** | `gitlab.com:thora1/thora-scontracts` | `ThoraGovernance.sol`, abigen bindings, and the Go tool `genesissmt.MakeAllocData` that produces the genesis alloc JSON (hex-encoded) embedded in `core/genesis_alloc.go` |
| **thora-playground/** (folder, not a repo) | contains 2 repos + 1 dir | Deployment material: genesis files, node TOML configs, docker-compose/swarm stacks, clef setup, operational notes |
| ├─ `thora-playground/thora-playground` | `gitlab.com:thora1/thora-playground` | **Mainnet** (686868) deployment |
| ├─ `thora-playground/thora-playground-oda` | `gitlab.com:thora1/thora-playground-oda` | **Oda testnet** (696969) deployment (compose + docker swarm) |
| └─ `thora-playground/testnet` | none | Legacy (Jul 2023) 3-signer **clique** testnet |
| thora-ansible | none | Ansible playbooks (host `thora-01` @ DigitalOcean); `06-thora` ships the encrypted validator docker bundle |
| thora-explorer | `gitlab thora1/thora-explorer-be` | Go 1.20 indexer (REST + GraphQL, Postgres, Redis) |
| thora-explorer-fe | — | Angular 16 explorer UI |
| thora-wallet | — | MetaMask extension fork (THORA chainId 686868 / `0xA7B14`) |
| thora-contract-verifier | — | Sourcify fork |
| thora-ide | — | Remix IDE fork |
| thora-docs | `gitlab thora1/thora-docs` | Docusaurus site (`docs.thora.network`) — mostly template content, **does not document the network yet** |
| thora-landing-page | — | Next.js static landing page (S3) |
| thora-domains | — | Screenshots only |

```
            ┌────────────────────────┐   MakeAllocData(govAddr)   ┌─────────────────────────────┐
            │   thora-scontracts      │ ─────────────────────────▶ │ thora-network               │
            │ ThoraGovernance.sol     │  hex(JSON GenesisAlloc)    │ core/genesis_alloc.go       │
            │ genesissmt/make_alloc.go│  (copied by hand)          │ core/genesis.go, params/... │
            └────────────────────────┘                            └──────────────┬──────────────┘
                                                                                 │ make all / docker build
                                                                                 ▼
                                                                  giangbb/thora-all:1.1 image
                                                                                 │
            ┌────────────────────────────────────────────────────────────────────┴───────────┐
            │ thora-playground(-oda): docker compose / swarm stacks                            │
            │  bootnode  ──  clef (keystore, rules.js)  ──  thora --signer http://clef --mine  │
            │  sync node + nginx + explorer (thoraidx) + verifier + postgres + redis (AIO)     │
            └──────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Fork delta vs upstream geth (what is *ours*)

Our authors: `giangbb <giangtpu@gmail.com>`, `giangtpu <giangtpu@gmail.com>`, `saivnct <saivnct@gmail.com>`
(`lexaivramovich` appears as author in thora-scontracts).
Inspect the whole delta: `git diff --stat b058cf454 HEAD` (106 files, +3760 / −1612).

### 2.1 Change inventory by area

| Area | Files | Change |
|---|---|---|
| **Consensus** | `consensus/thora/{thora.go,snapshot.go,api.go,thora_css.go}` + tests | New engine (see §3) |
| | `consensus/beacon/consensus.go` | comment only |
| **Params** | `params/config.go` | `ThoraConfig`, `ChainConfig.Thora`, `PlatformMainnetChainConfig`, `PlatformTestnetChainConfig`, `AllThoraProtocolChanges`, genesis hashes, reward recipients; Sepolia/Goerli commented out; fork list in `Description()` commented out |
| | `params/gen_thora_config.go` | Hand-tuned gencodec JSON (Period/Epoch as HexOrDecimal64, BlockReward as HexOrDecimal256) |
| | `params/platform_info.go` | `PlatformChainInfo` branding struct + `WaterMarkText()` template helper |
| | `params/bootnodes.go` | `MainnetBootnodes`/`TestnetBootnodes` = **empty**; Ethereum lists commented out; DNS discovery effectively disabled |
| | `params/version.go` | version → `1.0.1-unstable` |
| **Genesis** | `core/genesis.go` | `DefaultMainnetGenesisBlock()`, `DefaultTestnetGenesisBlock()`, `decodeThoraPreAlloc()` (hex→JSON), default genesis = Thora mainnet, "can't start thora chain without signers" check; Ethereum default genesis funcs commented out |
| | `core/genesis_alloc.go` | Only `platformMainnetAllocData` and `platformTestnetAllocData` (hex-encoded JSON strings) |
| **DB** | `core/rawdb/schema.go`, `database.go` | `ThoraSnapshotPrefix = "thora-"`, inspect stats |
| **State helpers** | `core/state/state_reader.go` | Solidity storage-slot calculators (simple var, mapping, dyn/fixed arrays, struct). Same file exists in thora-scontracts |
| **Simulated backend** | `accounts/abi/bind/backends/simulated.go` | `ForEachStorageAt()` (dump contract storage) |
| **eth backend** | `eth/backend.go` | `ValidateBeforeMining()` (is etherbase in current signer snapshot?), `StartMining(onThoraSignerFnErr)` authorizes Thora engine |
| | `eth/api_backend.go`, `eth/api_miner.go` | plumbing for the above (`miner_start` RPC passes `nil` callback) |
| | `eth/ethconfig/config.go` | `CreateConsensusEngine`: `config.Thora != nil` → `beacon.New(thora.New(...))` (checked **before** Clique); default `NetworkId = 686868` |
| **CLI** | `cmd/geth → cmd/thora` | Rename; branding via `WaterMarkText`; LES/legacy flags removed from flag list; `js` command removed; startup **refuses to mine if node isn't a current validator**; signer-error callback stops the node when using an external signer (clef) |
| | `cmd/thora/config.go` | Catalyst (Engine API) registration **disabled** (dev-mode simulated beacon kept) |
| | `cmd/utils/flags.go` | `--goerli/--sepolia` → `--oda` (`TestnetFlag`); `--miner.etherbase` renamed **`--miner.coinbase`**; mainnet/testnet genesis + networkId wiring |
| | `cmd/clef/main.go` | default chainid 686868 |
| | `cmd/bootnode`, `cmd/devp2p` | Print ENR platform version; `-thora-network` filter (mainnet/testnet) |
| | `cmd/faucet` | **deleted** |
| **P2P** | `p2p/enode/localnode.go` | Local ENR gets entry `thora = "v1.0.0"` |
| | `p2p/dial.go` | Discovered dial candidates without matching ENR entry are discarded |
| | `p2p/discover/table.go`, `v4_udp.go` | Revalidation drops nodes whose ENR lacks/mismatches the entry (`errPlatformProtocolVersion`) |
| | `p2p/server.go` | extra startup logs |
| **Console** | `internal/web3ext/web3ext.go` | `thora` JS namespace (`thora.getSigners`, `thora.propose`, …) |
| | `console/console.go`, `internal/flags/*` | branding |
| **Misc** | `node/defaults.go` | Default datadir `~/.thora` (`~/Library/Thora`, `%APPDATA%\Thora`) |
| | `metrics/config.go` | InfluxDB db/bucket/org default `thora` |
| **Build** | `Makefile`, `Dockerfile*`, `build/ci.go`, `build/nsis.thora*.nsh/.nsi`, `circle.yml`, `.gitignore` | Rename geth→thora |
| **Tests** | `core/block_validator_test.go`, `core/forkid/forkid_test.go`, `core/genesis_test.go`, `miner/worker_test.go`, `miner/miner_test.go`, `p2p/*_test.go`, `cmd/thora/*_test.go`, `les/peer_test.go`, `core/rawdb/accessors_indexes_test.go` | Thora variants / branding fixes |

### 2.2 Commit timeline (ours)

| Date | Commit(s) | Milestone |
|---|---|---|
| 2023-07-16 | `26663f895`, `775311ce4` | `cmd/thora`, remove faucet; validator-permission check on start |
| 2023-07-16→24 | `06d23c77d`…`865e903e1` | Basic block reward for previous signer (initially in clique) |
| 2023-07-26→30 | `8dad02e82`…`10cd49acc` | Thora config, defaults, disable LES/catalyst/deprecated flags, Clique-only mining, custom ENR entry |
| 2023-08-11→14 | `924632c31`…`1a015ae1d` | Platform protocol ENR branding & filtering; dev mode re-enabled; static node fix |
| 2023-08-17→20 | `4a2de8756`, `1d9e24289`, `846ca065f` | Close stack on seal error (clef failure) |
| 2023-08-22→27 | `5b2d85030`…`75329588a` | Genesis alloc calculator; mainnet/testnet genesis; Oda testnet |
| 2023-08-28 | `7e8d2a752`, `b078bd145`, `168b6492d`, `c566232b1` | **`consensus/thora` engine** split from clique (clique rolled back to upstream), web3 `thora` APIs, reward, JSON marshal fixes |
| 2023-08-29 | `26ccd6f88` | Update thora genesis |
| 2024-04-18 | `51848760a` | **Change mainnet genesis** (new governance roles 0x7947…/0xdA72…, validator 0x9296…) |

---

## 3. Consensus engine — `consensus/thora`

### 3.1 Files

| File | Content |
|---|---|
| `thora.go` | Engine. **Line-for-line copy of `consensus/clique/clique.go`** except: `OnSignerFnErr` callback in `Authorize`/`Seal`, and `Finalize` moved to `thora_css.go` |
| `snapshot.go` | **Identical to clique's** snapshot (voting, recents, inturn), except DB key prefix `thora-` |
| `thora_css.go` | Thora-specific: `IsCurrentValidator()` and **`Finalize()` (block reward)** |
| `api.go` | RPC namespace `thora`: `getSnapshot`, `getSnapshotAtHash`, `getSigners`, `getSignersAtHash`, `proposals`, `propose`, `discard`, `status`, `getSigner` (same as `clique_*`) |
| `*_test.go` | Ported clique tests (`TestThora` voting scenarios, `TestReimportMirroredState`, `TestSealHash`) |

### 3.2 Protocol rules (inherited from Clique / EIP-225)

- `header.Extra = vanity(32) ‖ [signers(N×20) only on epoch blocks] ‖ seal(65)`.
- Seal = secp256k1 signature over `keccak256(RLP(header without seal))` (`SealHash`, `encodeSigHeader` at `consensus/thora/thora.go:710-760`). Signed with mime type `accounts.MimetypeClique` → **clef rules must allow `application/x-clique-header`** (they do, see playground `rules.js`).
- Difficulty: `2` in-turn (`number % len(signers) == index`), `1` out-of-turn; out-of-turn sealers wait random wiggle `(N/2+1)*500ms`.
- A signer may sign at most once per `floor(N/2)+1` consecutive blocks (`Recents`).
- Votes: `Coinbase` = candidate, `Nonce` = `0xff..ff` (add) / `0x00..00` (drop); majority `> N/2` applies; votes reset at each epoch (30000). Epoch blocks must have zero coinbase/nonce and list current signers in extra.
- `Prepare` sets `Time = max(parent.Time + Period, now)`.
- Rejected: uncles, non-zero MixDigest, withdrawals, **Shanghai and Cancun** (`thora.go:302-307`).
- Snapshots: in memory LRU (128), persisted every 1024 blocks under `thora-<hash>`.
- Engine is always wrapped: `beacon.New(thora.New(cfg, db))` (`eth/ethconfig/config.go:173`). With `TerminalTotalDifficulty == nil` beacon simply delegates to Thora.

### 3.3 Block reward — `Finalize` (`consensus/thora/thora_css.go:25`)

```go
if cfg.BlockReward == nil { return }
if cfg.RewardRecipient != nil && *cfg.RewardRecipient != (common.Address{}) {
    if number > 0 { state.AddBalance(*cfg.RewardRecipient, cfg.BlockReward) }   // governance contract
} else if number > 1 {
    parentSigner := Author(parentHeader); state.AddBalance(parentSigner, cfg.BlockReward) // previous signer
}
```

- Pure mint (no tx, no log, no `receive()` call). Tx fees: with London, base fee is **burned**; priority tips go to the block **sealer** (EVM coinbase = `engine.Author(header)`), not to the governance contract.
- Reward amount/recipient are read from `params.ThoraConfig` — **not fork-gated**. Changing them in code or genesis changes consensus for every block processed afterwards (see §11 risks).

### 3.4 Mining start flow

```
thora --mine --miner.coinbase X --signer http://clef:8550
  cmd/thora/main.go startNode()
    ├─ ethBackend.ValidateBeforeMining()  → thora.IsCurrentValidator(X)  (Fatal if not a signer)
    └─ ethBackend.StartMining(onThoraSignerFnErr)
          └─ thora.Authorize(X, wallet.SignData, cb)
Seal(): signFn fails → cb(err) → if ExternalSigner set → stack.Close()  (node stops if clef is down/locked)
```

`miner_start` via RPC (`eth/api_miner.go`) skips the validator check and passes a nil callback.

---

## 4. Chain parameters — `params/`

`params/config.go`:

| Var | Notes |
|---|---|
| `PlatformMainNetGenesisHash` (l.35) | `0xb5b81a3397d8fa323d2ff79a37fe94288427840bc9a7d2b2cb4d558e7a74449c` |
| `PlatformTestNetGenesisHash` | `0xbced0bacc7f2710aabcaeee6b150f19f2107705d79aa1f3996b30f30495e989e` |
| `MainnetRewardRecipient` (l.42) | `0x0000000000000000000000000000000000000088` |
| `TestnetRewardRecipient` | `0x0000000000000000000000000000000000000089` |
| `PlatformMainnetChainConfig` (l.46) | ChainID 686868, all forks ≤ London at 0, Arrow/GrayGlacier nil, no Shanghai/Cancun, `Thora{Period:9, Epoch:30000, BlockReward:100e18, RewardRecipient:&Mainnet…}` |
| `PlatformTestnetChainConfig` (l.81) | Same with ChainID 696969 and `…0089` |
| `AllThoraProtocolChanges` (l.272) | Test config: ChainID 1337, `Period:0`, reward 100e18, no recipient (→ signer reward) |
| `MainnetChainConfig` / `MainnetGenesisHash` | Ethereum mainnet values are **still present** (used by some upstream tests / `NetworkNames`) |
| `ThoraConfig` (l.451) | `Period`, `Epoch`, `BlockReward *big.Int` (`gencodec:"required"`), `RewardRecipient *common.Address` |

`ChainConfig` JSON key: `"thora": {"period":9,"epoch":30000,"blockReward":"100000000000000000000","rewardRecipient":"0x…88"}`.

`params/platform_info.go` — `PlatformChainInfo`:

| Field | Value |
|---|---|
| PlatformName / ShortName / lower | `Thora Network` / `Thora` / `thora` |
| CoinName | `THA` |
| GETHCmd | `thora` (must equal `clientIdentifier` in `cmd/thora/main.go`) |
| IPCPath | `thora.ipc` |
| ENRPlatformProtocolNetworkOnly / Name / Version | `true` / `thora` / `v1.0.0` |
| TestPlatformName / ShortName / lower | `Oda Test Network` / `Oda` / `oda` (→ `--oda` flag, datadir `~/.thora/oda`) |

`WaterMarkText("… {{.PlatformShortName}} …")` renders branding into CLI strings and logs.

---

## 5. Genesis

### 5.1 Code path

- `core/genesis.go:592` `DefaultMainnetGenesisBlock()` — Config `PlatformMainnetChainConfig`, Nonce 0, GasLimit 30,000,000, Difficulty 1, Timestamp 0, ExtraData with 1 signer **`0x92965d3910E55b21b0B42fD405916a8e56eFEA0E`**, Alloc `decodeThoraPreAlloc(platformMainnetAllocData)`.
- `core/genesis.go:604` `DefaultTestnetGenesisBlock()` — Timestamp 1583482152, 1 signer **`0xf202ae205bBF80148a8c615670eBEF9ec63191f9`**, testnet alloc.
- `decodeThoraPreAlloc` (`core/genesis.go:654`): `hexutil.Decode` → `json.Unmarshal` into `GenesisAlloc` (upstream RLP `decodePrealloc` kept but unused).
- When no genesis is given, the node uses the Thora **mainnet** (`SetupGenesisBlockWithOverride`, `LoadChainConfig`). `configOrDefault` maps genesis hash → built-in config. Because of this, docker deployments **do not run `thora init`**: the genesis is built into the binary.

### 5.2 Embedded alloc contents (decoded)

**Mainnet** (`platformMainnetAllocData`) — produced by current thora-scontracts HEAD:

| Address | Balance | Code/Storage |
|---|---|---|
| `0x…0088` ThoraGovernance | 0 | runtime code (4065 B, solc 0.8.21, paris); slot0 `_creator = 0x7947e7ff…c862`; `_roles[0x7947…]=ADMIN`, `_roles[0xdA72…]=OPERATOR` |
| `0x7947e7fFa451F116a38225ed0a9B8924F508c862` (admin/creator) | 100 THA | — |
| `0xdA7286Fd787bb2052d4D08b3376BA765b29d96B2` (operator) | 100 THA | — |

**Testnet** (`platformTestnetAllocData`) — produced by an **older** `make_alloc.go` (pre 2024-04-18; not reproducible from HEAD):

| Address | Balance | Code/Storage |
|---|---|---|
| `0x…0089` ThoraGovernance | 99,999,999 THA | same code; creator `0xA3DFe2d2…1255`; 3 ADMIN + 2 OPERATOR role slots |
| `0xeD19d9CE…42b8`, `0xA3DFe2d2…1255`, `0x4ac14B57…094B` | 999 THA each | — |

Note: genesis **validators are not governance roles**; the two sets are independent.

### 5.3 Matching JSON genesis files

- Mainnet: `../thora-playground/thora-playground/genesis.json`
- Testnet: `../thora-playground/thora-playground-oda/genesis-oda.json` (= `genesis-test.json`)

---

## 6. Governance contract — `../thora-scontracts`

### 6.1 `ThoraGovernance.sol` (`truffle/contracts/ThoraGovernance.sol`, pragma `^0.8.0`, inherits OZ `Context` copy)

| Slot | Var |
|---|---|
| 0 | `address _creator` |
| 1 | `mapping(address => bytes32) _roles` (one role per account) |

Constants: `ADMIN_ROLE = keccak256("ADMIN_ROLE") = 0xa498…1775`, `OPERATOR_ROLE = keccak256("OPERATOR_ROLE") = 0x9766…b929`.

| Function | Access | Behaviour |
|---|---|---|
| `balance()` view | operator/admin | contract balance |
| `hasRole(role, acc)` view | operator/admin | |
| `grantRole(role, acc)` | admin | not creator; role ∈ {ADMIN, OPERATOR}; overwrites |
| `revokeRole(role, acc)` | admin | not creator; must hold role |
| `transfer(to, amount)` | operator/admin | `to.transfer(amount)` (2300 gas stipend); emits `Transfer` |
| `receive()` | anyone | accepts THA |

No proxy/upgradeability, no multisig/timelock/limits, no validator management. Creator is permanent and can't be revoked.

### 6.2 Alloc generator (`genesissmt/make_alloc.go`)

1. Deploy ThoraGovernance in an upstream-geth v1.12.2 `SimulatedBackend` with a random key → read **runtime code** via `CodeAt`.
2. Compute storage manually with `state.GetLocSimpleVariable(0)` / `GetLocMappingAtKey(addr, 1)`.
3. Hard-coded creator/roles/balances (currently creator+ADMIN `0x7947…`, OPERATOR `0xdA72…`, contract balance 0, 100 THA each EOA).
4. `json.Marshal(map[Address]GenesisAccount)` → `hexutil.Encode` → printed by `log.Printf` → **copied by hand** into `core/genesis_alloc.go`.

Build bindings: `contract/build.sh` (`solc --evm-version paris` → `abigen`). Truffle project in `truffle/` (tests, migrations; chainIds 686868/696969).

---

## 7. Networking (P2P)

- Every local node record carries ENR key `thora` = `v1.0.0` (`p2p/enode/localnode.go:89`).
- `p2p/dial.go:249`: discovered candidates without a matching entry are not dialed. Static/trusted nodes are dialed regardless (`startStaticDials`).
- `p2p/discover/table.go:355`: on revalidation, the ENR is fetched when its seq is higher. Nodes without a matching entry are treated as failed and evicted.
- Inbound connections are **not** filtered by ENR. eth protocol handshake still enforces networkId + genesis + forkid.
- Bootnodes: none in code (`params/bootnodes.go`). They are set via TOML `BootstrapNodes` / `--bootnodes` (playground bootnode container on udp 30301).
- `cmd/bootnode` prints the ENR platform version on start.

---

## 8. CLI & operator surface

| Item | Value |
|---|---|
| Networks | default = Thora mainnet; `--oda` = testnet; `--dev` = ephemeral **PoS simulated-beacon** dev chain (upstream `AllDevChainProtocolChanges`, *not* Thora) |
| Coinbase flag | `--miner.coinbase` (renamed from `--miner.etherbase`) |
| Datadir | `~/.thora` (testnet `~/.thora/oda`) |
| IPC | `thora.ipc` |
| Console | `thora attach` → `thora.getSigners()`, `thora.propose(addr, true)`, `thora.discard(addr)`, `thora.status()`, `thora.proposals` |
| RPC namespace | `thora_*` (enable with `--http.api eth,net,web3,thora`) |
| Engine API | disabled (`cmd/thora/config.go:208`) |
| LES | flags hidden; light serving effectively off |

---

## 9. Build, test, release

```bash
make thora                      # build/bin/thora
make all                        # all tools (thora, bootnode, clef, abigen, evm, devp2p, rlpdump …)
go test ./core/forkid           # update forkid_test.go first when genesis/forks change
go test -v ./core ./cmd/thora ./consensus/thora ./miner
make test
docker build -f Dockerfile.alltools -t giangbb/thora-all:1.1 .
```

- **Toolchain:** go.mod says `go 1.19`. Docker uses `golang:1.20-alpine`. Linking fails on host Go ≥ 1.23 (verified with Go 1.27.1: `link: github.com/fjl/memsize: invalid reference to runtime.stopTheWorld`). Build in Docker or with Go 1.20–1.22.
- Genesis hash: `thora init` doesn't print it. The documented trick is a temporary `fmt.Printf("Genesis hash: %v\n", hash)` at the end of `initGenesis` in `cmd/thora/chaincmd.go` (a commented line is already there).

### 9.1 Network (re)configuration procedure (current practice)

1. Choose governance address (e.g. `0x…88`).
2. In thora-scontracts: set `main.go` address + roles in `make_alloc.go`; `go run .` → copy the hex output.
3. `make thora`; `thora --datadir node1 account new` → validator address.
4. Write `genesis.json` (alloc code/storage/balances; `extraData = 0x00×32 ‖ validator ‖ 0x00×65`; `config.thora{…}`).
5. Get the genesis hash (`thora init` with the temporary print).
6. Update `params/config.go` (`*RewardRecipient`, `Platform*GenesisHash`, `ThoraConfig` same as genesis.json), `core/genesis_alloc.go` (`platform*AllocData`), `core/genesis.go` (`Default*GenesisBlock` same as genesis.json).
7. Fix `core/forkid/forkid_test.go`, `core/genesis_test.go`; run tests.
8. `make all`; build & push the docker image.

---

## 10. Deployment topology (`../thora-playground`)

**Validator stack** (`thora-playground/docker/docker-compose.yml`, image `giangbb/thora-all:1.1`):

| Service | Command |
|---|---|
| `thora-boot-node` | `bootnode --nodekey … --nat extip:$NODE_EXT_IP` (udp 30301) |
| `thora-clef` | `clef --keystore … --rules rules.js --http --http.addr 0.0.0.0 --nousb --ipcdisable`; master password via stdin (env from `start.sh` prompt, or swarm secret `clef_master_pw`) |
| `thora-node` | `thora --log.rotate --nodekey … --config node-config.toml --nat extip:… --signer http://thora-clef:8550 --miner.coinbase $COIN_BASE_ADDR --mine` (+ `--oda` on testnet) |

- Clef `rules.js` auto-approves only clique-header signing and account listing.
- **Sync/RPC node** (`docker-syn-node*`): `thora --config … --nat extip:…`, HTTP 8545 / WS 8546.
- **AIO** (`docker-syn-node-aio`): sync node + nginx (`dataseeds.thorachain.io` → RPC, `verifier.thorachain.io`) + `giangbb/thora-explorer:1.1` (thoraidx) + `giangbb/thora-contract-verifier:1.1` + postgres 16 + redis.
- Oda testnet also has a **docker swarm** stack (`docker-swarm/thora-stack.yml`, image `thora/thora-all-ext:1.0`).
- TOML configs: NetworkId, SyncMode full, HTTP `net,web3,eth`, GasPrice 1 gwei, GasCeil 30M, MaxPeers 50.
- Known hosts: mainnet `thora-01` @ `143.198.217.49` (Ansible inventory).
- Validators are added with `thora.propose(addr,true)` from existing signers. Notes list candidate validators for mainnet (`0x91cbD606…`, `0xefd100b8…`) and Oda.

---

## 11. Known gaps, risks & tech debt (found during survey)

Severity is a first estimate. Items marked **(Q)** have open questions in §13.

### Consensus / protocol
1. **[Med] `shouldPreserve` not adapted for Thora** — `eth/backend.go:368` returns `false` only for an unwrapped `*clique.Clique`. Thora is a different type and is wrapped in `beacon.Beacon`, so validators *prefer their own blocks* in equal-TD fork choice (`core/forkchoice.go`, wired at `eth/backend.go:204`). That is the deadlock scenario upstream disabled for clique (see the comment block there). Fix: unwrap beacon and match `*thora.Thora`.
2. **[High] Reward/recipient changes are not fork-gated** — `ThoraConfig` changes alter state transitions for all past and future blocks. `ChainConfig.CheckCompatible` does not compare `Thora` fields. Any future change (amount, halving, recipient) needs a block-number activation field.
3. **[Med] Single validator on mainnet and testnet genesis** → no liveness fault tolerance. With N signers Clique tolerates fewer than N/2 faulty signers for safety *and* liveness, with no finality. Clique-style PoA has known reorg/fork issues; BFT (QBFT/IBFT) gives instant finality. **(Q)**
4. **[Med] Stuck at London EVM** — Shanghai/Cancun are rejected. Modern toolchains (solc ≥0.8.20 default target shanghai, PUSH0) produce bytecode that fails. Wallets and dev tools must target `paris`. Upgrading requires a header-format review: `WithdrawalsHash` is nil-only in `encodeSigHeader`; Cancun adds blob fields and the beacon root.
5. **[Med] Validator set is off-chain (header votes), the governance contract is reward-only** — two independent authority systems. **(Q)**
6. **[Info] Tx fees:** base fee is burned (London). Priority tips go to the **sealer**, because `NewEVMBlockContext` uses `engine.Author(header)` = recovered signer (`core/evm.go:47`). In mining, the worker uses the etherbase, which is the same address. `header.Coinbase` itself only carries the vote candidate.
7. **[Low] `miner/worker.go:454,554`** special-cases `Clique.Period == 0` only. Thora with `Period 0` (used by `AllThoraProtocolChanges` tests) won't get the same handling.
8. **[Low] `params/gen_thora_config.go` UnmarshalJSON** dereferences `dec.Period`/`dec.Epoch` without a nil check (panic on a genesis missing those keys). `BlockReward` "required" is not enforced. File is hand-edited; re-running `go generate` would overwrite it.
9. **[Low] Banner** (`ChainConfig.Description`) prints "Consensus: unknown" for Thora. The fork list printing is commented out.
10. **[Low] `ValidateBeforeMining` only at startup**; `miner_start` RPC bypasses it. A removed signer keeps trying and gets `errUnauthorizedSigner`.

### Governance contract
11. **[High] Any single ADMIN/OPERATOR key can drain all rewards** (100 THA every ~9s). No multisig, timelock or limits. The creator is irrevocable.
12. **[Med] `transfer` uses `.transfer` (2300 gas)** → payouts to smart wallets (Safe) can fail.
13. **[Low] Operator-gated `view`s** (`balance`, `hasRole`) give no privacy and break tooling. There are no reward events (mint is silent).

### Reproducibility / ops
14. **[Med] Genesis alloc is hand-copied from log output**. There is no automated check that `genesis.json`, `core/genesis_alloc.go`, `core/genesis.go` and `params/config.go` agree. The testnet alloc can't be regenerated from thora-scontracts HEAD. Compiler drift: Truffle 0.8.18 vs bindings 0.8.21.
15. **[Med] Toolchain pin**: Go ≤1.22 required. Upstream geth is ~3 years ahead, so security patches are missing (p2p, txpool, RPC DoS fixes).
16. **[High, ops] Secrets committed in sibling repos** (paths only): `thora-playground-oda` (`password.txt`, `docker.zip`, `node*/keystore`, clef masterseed/credentials, nodekeys; a raw private key in history), `thora-playground` (nodekeys, `docker-syn-node-aio/*.env`, redis.conf), `thora-scontracts` (`truffle/wallet/*`, `keystore/`, raw key in `play_ground.go`), `thora-ide/.git/config` (GitLab token), `thora-ansible` (TLS keys, user password). Needs key rotation + history rewrite. Validator 0xf202… (testnet) keystore and clef material are in git.
17. **[Low] Branding leftovers**: README is still upstream geth's; `MainnetChainConfig` (Ethereum) is still referenced; Ethereum code is commented out rather than deleted (merge noise); "GIANGBB TEST" logs in `p2p/server.go`.
18. **[Low] ENR filter is hygiene, not security** — trivially spoofable. Inbound and static peers are not filtered. Real isolation comes from networkId/genesis/forkid.

---

## 12. How-to recipes for contributors

| Task | Touch |
|---|---|
| Change consensus rule | `consensus/thora/thora.go` (verify + prepare/seal must stay symmetric); add a fork-activation param in `ThoraConfig` + `ChainConfig.CheckCompatible`; tests in `consensus/thora` and `core/block_validator_test.go` |
| Change reward logic | `consensus/thora/thora_css.go` `Finalize` (runs for both import and mining) — keep deterministic, gate by block number |
| New chain param | `params/config.go` `ThoraConfig` + `params/gen_thora_config.go` (hand-maintained!) + genesis JSON files |
| New RPC method | `consensus/thora/api.go` + `internal/web3ext/web3ext.go` (`ThoraJs`) |
| Rebrand string | `params/platform_info.go` + `WaterMarkText` |
| Change genesis | follow §9.1; update `core/forkid/forkid_test.go`, `core/genesis_test.go` |
| P2P peer filtering | `p2p/dial.go`, `p2p/discover/table.go`, `p2p/enode/localnode.go` |
| Mining start behaviour | `cmd/thora/main.go` `startNode`, `eth/backend.go` |
| Upstream merge | expect conflicts in `cmd/utils/flags.go`, `params/config.go`, `core/genesis.go`, `eth/backend.go`, `p2p/*`, `build/ci.go`. `consensus/thora` must mirror upstream clique changes manually |

Conventions: our edits are often marked `//GIANGBB` / `/* Giangbb: ... */`; removed upstream code is commented out rather than deleted; branding goes through `params.PlatformChainInfo`.

---

## 13. Open questions (to confirm with the project owner)

- Which BFT flavour: QBFT (Besu/Quorum-compatible, IBFT-2.0 successor) vs. a custom/HotStuff design? Is validator-set management meant to be on-chain (contract) from day one?
- Target upstream geth version for the rebase (affects how consensus plugs in: post-1.13 geth removed much pre-merge machinery, and the beacon wrapper/TTD paths differ).
- Reward economics: keep a fixed 100 THA/block minted to the governance vault, or redesign (cap, halving, validator share)? Should the vault become a multisig/timelock?
- Should the Oda testnet be relaunched together with mainnet on the new design?
- Who are the long-term governance key holders, and is the irrevocable `_creator` acceptable?

## 14. Decisions & answers log

| Date | Decision / answer |
|---|---|
| 2026-10-10 | **Network is not running** (neither mainnet nor Oda). Genesis and consensus can be redesigned freely before launch; no hard-fork gating needed for the relaunch. Gating is still required for changes after launch |
| 2026-10-10 | **Validator governance → move to BFT (QBFT/IBFT)**, replacing Clique-style header voting |
| 2026-10-10 | **Next workstreams:** (1) consensus upgrade, (2) upstream rebase + EVM (Shanghai/Cancun, modern Go), (3) governance hardening |
| 2026-10-10 | **Working rule:** no codebase changes without the owner's explicit approval. Write plan files first for review |
| 2026-10-10 | Development Rules section added (no hardcoding, Go/Ethereum standards, SOLID, mandatory tests, keep this map current) |

## 15. Checkpoint

Keep only the **latest 5–10 commits** on `master`: newest first, drop the oldest. Use `git show <hash>` for details.
Refresh with: `git log -10 --format='| %ad | %h | %s |' --date=format:'%Y-%m-%d %H:%M'`

| Date (local) | Commit | Summary |
|---|---|---|
| 2024-04-18 15:41 | `51848760a` | change mainnet genesis |
| 2023-08-29 00:21 | `26ccd6f88` | update thora genesis |
| 2023-08-28 21:29 | `c566232b1` | rollback clique update test cases |
| 2023-08-28 18:16 | `168b6492d` | thora engine reward; config marshal fix |
| 2023-08-28 16:43 | `b078bd145` | thora engine to mainnet; config JSON fix |
| 2023-08-28 14:42 | `caba7566b` | fix test |
| 2023-08-28 12:31 | `7e8d2a752` | thora consensus engine + web3 thora APIs |
| 2023-08-28 07:05 | `75329588a` | update oda testnet genesis |
| 2023-08-27 07:04 | `1af934ffb` | rollback oda testnet |
| 2023-08-26 08:04 | `90af54284` | apply msc to oda |

Uncommitted: `PROJECT_FULL_MAP.md`, `CLAUDE.md`, `AGENTS.md` (created 2026-10-10).
