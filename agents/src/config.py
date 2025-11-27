"""Configuration management for NeoGraph agents."""
import os
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Environment configuration for agents service."""

    # Anthropic API
    anthropic_api_key: str = os.getenv("ANTHROPIC_API_KEY", "")

    # Neo4j connection
    neo4j_uri: str = os.getenv("NEO4J_URI", "bolt://localhost:7687")
    neo4j_user: str = os.getenv("NEO4J_USER", "neo4j")
    neo4j_password: str = os.getenv("NEO4J_PASSWORD", "password")

    # Service configuration
    host: str = "0.0.0.0"
    port: int = 8001

    # CORS
    cors_origins: list[str] = ["*"]

    class Config:
        env_file = ".env"


settings = Settings()
