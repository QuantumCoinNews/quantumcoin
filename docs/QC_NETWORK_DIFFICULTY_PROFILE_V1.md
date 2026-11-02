# QuantumCoin Network and Difficulty Profile V1

Decision ID: `QC_NETWORK_DIFFICULTY_PROFILE_V1`

## Runtime profiles

| Profile | Base difficulty | Fixed override | API | P2P | Chain |
|---|---:|---|---:|---:|---|
| local-release | 16 | Temporarily allowed for protected local release flow | 8082 | 3001 | chain_data.dat |
| testnet-local | 8 | Not allowed | 18082 | 13001 | chain_data_testnet.dat |
| testnet-public | 8 | Not allowed | 18082 | 13001 | chain_data_testnet.dat |
| mainnet | Current canonical base 16 | Not allowed | 8082 | 3001 | chain_data.dat |

## Safety rules

- An unspecified process is `local-release`, not activated mainnet.
- Mainnet classification requires an explicit mainnet signal.
- Testnet always uses separate ports, network magic, and storage files.
- `QC_DIFFICULTY_BITS=12` in the protected release miner is a local fixed override, not the canonical network default.
- Testnet must clear inherited `QC_DIFFICULTY_BITS` so base difficulty 8 and dynamic adjustment remain effective.
- Public testnet remains disabled until explicit approval.
- This step does not launch mainnet or public testnet.
- This step does not change protected release files.