from __future__ import annotations

import pandas as pd
from sqlalchemy import text

from retailer_sales_forecasting_system.db.postgres import get_engine

DEFAULT_TABLE = "walmart_sales"


def load_sales(table: str = DEFAULT_TABLE) -> pd.DataFrame:
    """
    Loads the full sales table into a Pandas DataFrame.
    """
    engine = get_engine()
    query = text(f"SELECT * FROM {table};")

    with engine.connect() as conn:
        df = pd.read_sql(query, conn)

    # Optional: ensure date is datetime if the DB column is DATE
    if "date" in df.columns:
        df["date"] = pd.to_datetime(df["date"])

    return df
