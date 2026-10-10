# CLAUDE.md — Thora Network

Guidance for Claude Code (and other AI assistants) working in this repository.
**Read `PROJECT_FULL_MAP.md` first.** It is the full architecture and code map; this file holds the rules.
`AGENTS.md` has the same rules for other AI agents (Codex/ChatGPT). **Keep both files in sync.**

## Project in one paragraph

`thora-network` is a fork of **go-ethereum v1.12.1** (forked at upstream `b058cf454`) running a public
EVM Proof-of-Authority chain. Binary `thora` (`cmd/thora`), mainnet "Thora Network" (chainId 686868, coin THA),
testnet "Oda" (`--oda`, chainId 696969). Consensus `consensus/thora` is a Clique clone that mints a block reward
to the `ThoraGovernance` contract (pre-deployed in genesis, source in `../thora-scontracts`). Deployment
material lives in `../thora-playground`. The Go module path is still `github.com/ethereum/go-ethereum`. Keep it.

## Current status & direction

- The network is **not running**. Genesis and consensus may be redesigned before relaunch.
- Planned workstreams, in order: (1) BFT consensus (QBFT/IBFT), (2) upstream geth rebase + Shanghai/Cancun EVM,
  (3) governance hardening (vault multisig/timelock, fork-gated reward params, reproducible genesis).
- Known risks and tech debt: `PROJECT_FULL_MAP.md` §11. Open questions: §13. Decisions: §14.

## Mandatory workflow

1. **Understand first.** Clarify the business requirement and the impact on consensus, state determinism,
   genesis/forkid, P2P, RPC compatibility and the sibling repos before proposing anything.
2. **Brainstorm and interview** the owner until requirements and approach are clear. Challenge assumptions and
   compare trade-offs; don't silently pick one.
3. **Plan file first.** Write `plans/<topic>.md` (goal, scope, design, files touched, test plan, risks, rollout).
4. **Wait for explicit approval.** Do **not** modify code, configs or sibling repos until the owner approves.
   Approval for one task does not carry over to the next. Never commit or push unless asked.
5. **Implement → test → update docs.** Every code change also updates `PROJECT_FULL_MAP.md`: the affected
   sections, §11/§14 where relevant, and the §15 checkpoint, which keeps only the latest 5–10 commits.

## Coding rules

- **No hardcoded values.** Chain params, addresses, amounts, timeouts, names go in `params/`
  (`ThoraConfig`, `PlatformChainInfo`), CLI flags or TOML config. Use `params.WaterMarkText` for branding strings.
- **Go & Ethereum standards.** Idiomatic Go, `gofmt`/`goimports`, `go vet`, `go run build/ci.go lint`.
  Handle every error. Follow EIPs and upstream geth conventions (engine interfaces, `ChainConfig` fork gating,
  `gencodec` codecs).
- **Consensus must be deterministic.** No wall-clock, randomness, map-iteration order or floats in state
  transitions. `Finalize` runs on both import and mining; verify and seal paths must stay symmetric.
- **Fork-gate consensus changes** once a chain is live: add a block or time activation field and include it in
  `ChainConfig.CheckCompatible`.
- **SOLID & patterns.** Small, single-purpose types; extend via interfaces such as `consensus.Engine`; inject
  config/DB/chain reader rather than using globals; composition over copy-paste.
- **Minimise upstream delta.** Prefer new files/packages over inline edits to geth code so rebases stay cheap.
  Mark unavoidable edits to upstream files clearly. Don't add more commented-out upstream code; delete it.
- **Security first.** Never commit keys, keystores, passwords, nodekeys or `.env` secrets. Never print secret
  values in output or docs.

## Testing (required before any commit)

- New features and bug fixes come with unit tests. Consensus changes also need multi-block, reorg and
  validator-change scenarios.
- Minimum suite:
  ```bash
  go test ./consensus/... ./core/... ./params/... ./cmd/thora/... ./miner/... ./eth/...
  make test          # full suite before merging
  ```
- Changing genesis or forks? Update `core/forkid/forkid_test.go` and `core/genesis_test.go` first.

## Build

```bash
make thora                                                    # -> build/bin/thora
make all                                                      # all tools
docker build -f Dockerfile.alltools -t giangbb/thora-all:1.1 .
```
**Toolchain:** use Go 1.20–1.22 (Docker uses `golang:1.20-alpine`). Go ≥ 1.23 fails to link
(`fjl/memsize: invalid reference to runtime.stopTheWorld`).

## Key locations

| Concern | Files |
|---|---|
| Consensus engine | `consensus/thora/` (`thora.go`, `snapshot.go`, `api.go`, `thora_css.go` = reward + validator check) |
| Engine selection | `eth/ethconfig/config.go` `CreateConsensusEngine` |
| Chain params | `params/config.go` (`ThoraConfig`, `Platform*ChainConfig`, genesis hashes), `params/gen_thora_config.go` (hand-maintained JSON codec) |
| Branding / ENR id | `params/platform_info.go` |
| Built-in genesis | `core/genesis.go` (`DefaultMainnetGenesisBlock`, `DefaultTestnetGenesisBlock`), `core/genesis_alloc.go` |
| Mining start / signer | `cmd/thora/main.go` `startNode`, `eth/backend.go` (`ValidateBeforeMining`, `StartMining`) |
| CLI flags | `cmd/utils/flags.go` (`--oda`, `--miner.coinbase`) |
| P2P network filter | `p2p/enode/localnode.go`, `p2p/dial.go`, `p2p/discover/table.go` |
| Console API | `internal/web3ext/web3ext.go` (`thora.*`) |

## Genesis change procedure (short)

1. `../thora-scontracts`: set the governance address and roles, run `go run .`, copy the hex alloc.
2. Write `genesis.json` (alloc, `extraData` = 32-byte vanity ‖ validators ‖ 65-byte seal, `config.thora`).
3. Get the genesis hash with `thora init` (temporary print in `cmd/thora/chaincmd.go` `initGenesis`).
4. Sync `params/config.go`, `core/genesis_alloc.go` and `core/genesis.go` with `genesis.json`.
5. Fix the forkid/genesis tests, then run the full suite.
Full details: `PROJECT_FULL_MAP.md` §9.1.

## Conventions

- Owner-authored commits: `giangbb`, `giangtpu`, `saivnct`. Existing custom edits are often tagged `//GIANGBB`.
- Contracts deployed to Thora must target `--evm-version paris` until Shanghai is enabled (no `PUSH0`).
- Sibling repos (`../thora-scontracts`, `../thora-playground`, …) are read-only unless the owner says otherwise.
- Answer in a critical-partner style: verify claims against code and specs, never invent implementation details.
