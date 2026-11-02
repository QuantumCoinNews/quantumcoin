# NFT final content and approval preparation

The technical evidence flow is implemented. Actual final artwork/metadata,
authorized approval evidence and an independently operated witness deployment
remain preparation work. Catalog drafts do not enable minting.

Authenticated administrative routes provide:

- `GET /api/admin/nft/monthly/catalog-draft`: twelve draft periods derived from
  the verified node activation date, including year boundaries; no file writes.
- `POST /api/admin/nft/monthly/final-readiness`: read-only content/approval status.
- `POST /api/admin/nft/monthly/final-approval-statement`: exact unsigned bytes for
  separate AI/admin approval roles after the content and draw evidence validate.
- `POST /api/admin/nft/monthly/publish-draw-commitment`: durable publication intent,
  independent first record, public reread and local registration.
- `POST /api/admin/nft/monthly/draw-from-beacon`: pinned network and exact-round
  verification followed by an atomic draw receipt.

Before a real award: supply all twelve final images/metadata, verify unique catalog
identities and actual byte hashes, configure distinct authorized AI/admin public
keys and obtain real signed statements. Install the independent witness on an
independent operator's HTTPS host and pin its key/origin plus the chosen beacon
network parameters. Wallet challenge signing also needs its player-facing UX.

The statement/preflight routes do not create approval signatures or mint permission.
Missing content, signatures, trusted policy or chain confirmation prevents completion.
No real artwork, operator key, hosting deployment or live mint was created by the
source update. Preparation files are in `nft/preproduction/`.
