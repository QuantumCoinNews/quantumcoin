# QuantumCoin Immutable Economy Lock V1

Decision ID: QC_IMMUTABLE_ECONOMY_LOCK_V1

This file records the locked QuantumCoin economy decision.

## Locked supply

- Total supply: 25,500,000 QC
- Initial reward: 50 QC
- Coinbase maturity: 10 blocks
- Burn address: QC_BURN_SINK

## Founder balance lock / vesting

Decision ID: QC_FOUNDER_BALANCE_LOCK_VESTING_STANDARD_V1

- Founder total: 5% = 1,275,000 QC
- Founder liquid/open at start: 1% = 255,000 QC
- Founder locked: 4% = 1,020,000 QC
- Cliff: 12 months
- Vesting: 48 months after cliff
- Total schedule: 60 months
- Monthly unlock after cliff: 21,250 QC
- Unlock starts at month 13
- Unlock ends at month 60

## Turkish explanation

Toplam arz 25.500.000 QC olarak sabittir.

Kurucu/founder toplam payı %5'tir. Bu da 1.275.000 QC eder.

Bu toplam pay ikiye ayrılır:

1. Başlangıçta serbest olan pay:
   %1 = 255.000 QC

2. Kilitli kurucu payı:
   %4 = 1.020.000 QC

İlk 12 ay kilitli paydan hiçbir QC açılmaz. Buna cliff denir.

12 ay bittikten sonra kilitli olan 1.020.000 QC, 48 ay boyunca aylık açılır.

Aylık açılım:

1.020.000 QC / 48 ay = 21.250 QC / ay

60. ay sonunda kilitli kurucu payının tamamı açılmış olur.

## Immutable safety rule

AI, Telegram Game, TGWT, NFT, token factory, autopublisher, admin tools, or workers must not:

- Increase total supply
- Mint extra QC outside fixed supply
- Unlock founder locked balance early
- Bypass the 12-month cliff
- Bypass the 48-month vesting schedule
- Spend locked founder balance before unlock
- Change burn policy
- Change consensus balances
