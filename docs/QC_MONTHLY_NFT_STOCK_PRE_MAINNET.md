# QuantumCoin Monthly NFT Stock Pre-Mainnet

Status: NFT_MONTHLY_STOCK_PRE_MAINNET_V1

Date: 20260618_220020

## Purpose

This file tracks the 12 monthly rare NFT stock slots before mainnet.

## Stock file

- nft/catalog/monthly_rare_nft_stock_2026.json

## Rules

- 12 total monthly NFT stock items.
- 1 rare NFT per month.
- Every NFT must be independent and unique.
- All stock starts as TO_PREPARE.
- Mainnet mint status starts as NOT_MINTED.
- Final image hash and final metadata hash are empty until artwork/metadata are finished.
- FinalReady stays false until mainnet-prep validation.
- AI review and admin approval are required before distribution.
- This file is preparation only and does not mint anything.

## Mainnet-before fill checklist

For each item before mainnet:

1. Create final artwork.
2. Create final metadata.
3. Verify it is unique against the other 11 NFTs.
4. Record final image SHA256.
5. Record final metadata SHA256.
6. Set artworkStatus to FINAL_READY.
7. Set metadataStatus to FINAL_READY.
8. Set uniquenessCheck to PASSED.
9. Set aiReviewStatus to APPROVED.
10. Set adminApprovalStatus to APPROVED.
11. Set finalReady to true.
12. Keep mainnetMintStatus as NOT_MINTED until actual mainnet monthly distribution.
