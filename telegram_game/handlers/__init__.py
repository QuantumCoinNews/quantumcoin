from .start import router as start_router
from .mining import router as mining_router
from .wallet import router as wallet_router
from .referral import router as referral_router
from .leaderboard import router as leaderboard_router
from .claim import router as claim_router
from .profile import router as profile_router

all_routers = [
    start_router,
    mining_router,
    wallet_router,
    referral_router,
    leaderboard_router,
    claim_router,
    profile_router,
]
