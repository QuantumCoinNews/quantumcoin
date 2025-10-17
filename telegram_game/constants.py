from datetime import date, timedelta

# TGWT program: 2,000,000 over 1 year
TGWT_TOTAL = 2_000_000
TGWT_START = date.today()
TGWT_END   = TGWT_START + timedelta(days=365)
TGWT_DAYS  = (TGWT_END - TGWT_START).days or 365
TGWT_DAILY_POOL = TGWT_TOTAL // TGWT_DAYS
USER_DAILY_TGWT_CAP = 20

# Flood / anti-spam
FLOOD_COOLDOWN_SEC = 1.5

# User daily session limit
USER_DAILY_SESSION_CAP = 60

# Moon sectors (duration sec, energy cost, base_qc, sector multiplier)
SECTORS = {
    "Sector 1": {"duration": 45,  "energy": 8,  "base_qc": 0.40, "sector_mul": 1.00},
    "Sector 2": {"duration": 60,  "energy": 10, "base_qc": 0.45, "sector_mul": 1.05},
    "Sector 3": {"duration": 75,  "energy": 11, "base_qc": 0.50, "sector_mul": 1.10},
    "Sector 4": {"duration": 90,  "energy": 12, "base_qc": 0.55, "sector_mul": 1.15},
    "Sector 5": {"duration": 120, "energy": 14, "base_qc": 0.60, "sector_mul": 1.20},
    "Sector 6": {"duration": 150, "energy": 16, "base_qc": 0.70, "sector_mul": 1.25},
    "Sector 7": {"duration": 180, "energy": 18, "base_qc": 0.80, "sector_mul": 1.30},
    "Sector 8": {"duration": 240, "energy": 20, "base_qc": 0.95, "sector_mul": 1.40},
}

# Social tasks (TGWT only via socials)
SOCIAL_TASKS = {
    "Follow YouTube": 5,
    "Follow X": 5,
    "Follow Telegram": 5,
    "Watch Short": 3,
}
