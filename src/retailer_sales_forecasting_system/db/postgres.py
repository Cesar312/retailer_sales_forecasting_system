from __future__ import annotations

import os
from pathlib import Path

from dotenv import load_dotenv
from sqlalchemy import create_engine
from sqlalchemy.engine import Engine

def _load_repo_env() -> None:
    """
    Loads the repo-root .env file. This keeps notebooks/scripts consistent
    regardless of current working directory.
    """
    # repo_root/.../src/retailer_sales_forecasting_system/db/postgres.py -> go up 4 levels to repo root
    repo_root = Path(__file__).resolve().parents[4]
    env_path = repo_root / ".env"
    load_dotenv(dotenv_path=env_path, override=False)


def get_engine() -> Engine:
    """
    Returns a SQLAlchemy Engine for the Postgres instance defined in .env.
    """
    _load_repo_env()

    host = os.getenv("DB_HOST", "localhost")
    port = os.getenv("DB_PORT", "5433")
    user = os.getenv("DB_USER")
    password = os.getenv("DB_PASSWORD")
    dbname = os.getenv("DB_NAME")

    missing = [k for k, v in {
        "DB_USER": user,
        "DB_PASSWORD": password,
        "DB_NAME": dbname,
    }.items() if not v]

    if missing:
        raise ValueError(f"Missing required env var(s): {', '.join(missing)}. Check repo-root .env.")

    url = f"postgresql+psycopg2://{user}:{password}@{host}:{port}/{dbname}"
    return create_engine(url, pool_pre_ping=True, future=True)
